package handlers

import (
	"context"
	"net/http"
	"time"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/util"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/util/deployment"
	"bitbucket.org/sudosweden/dockyards-backend/pkg/util/name"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/google/uuid"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

func (h *handler) PostOrgClusters(c *gin.Context) {
	ctx := context.Background()

	org := c.Param("org")
	if org == "" {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	objectKey := client.ObjectKey{
		Name: org,
	}

	var organization v1alpha1.Organization
	err := h.controllerClient.Get(ctx, objectKey, &organization)
	if err != nil {
		h.logger.Error("error getting organization from kubernetes", "err", err)

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	subject, err := h.getSubjectFromContext(c)
	if err != nil {
		h.logger.Debug("error fetching subject from context", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	isMember := h.isMember(subject, &organization)
	if !isMember {
		h.logger.Debug("subject is not a member of organization", "subject", subject, "organization", organization.Name)

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	var clusterOptions v1.ClusterOptions
	if c.BindJSON(&clusterOptions) != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to read Body",
		})
		return
	}

	h.logger.Debug("create cluster", "organization", organization.Name, "name", clusterOptions.Name, "version", clusterOptions.Version)

	details, validName := name.IsValidName(clusterOptions.Name)
	if !validName {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":   "name is not valid",
			"name":    clusterOptions.Name,
			"details": details,
		})
		return
	}

	if clusterOptions.NodePoolOptions != nil {
		for _, nodePoolOptions := range *clusterOptions.NodePoolOptions {
			details, validName := name.IsValidName(nodePoolOptions.Name)
			if !validName {
				c.JSON(http.StatusUnprocessableEntity, gin.H{
					"error":   "node pool name is not valid",
					"name":    nodePoolOptions.Name,
					"details": details,
				})
				return
			}

			if nodePoolOptions.Quantity > 9 {
				h.logger.Debug("quantity too large", "quantity", nodePoolOptions.Quantity)

				c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{
					"error":    "node pool quota exceeded",
					"quantity": nodePoolOptions.Quantity,
					"details":  "quantity must be lower than 9",
				})
				return
			}
		}
	}

	existingClusters, err := h.clusterService.GetAllClusters()
	if err != nil {
		h.logger.Error("unexpected error getting existing clusters", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	for _, existingCluster := range *existingClusters {
		if existingCluster.Organization != organization.Name {
			continue
		}

		if existingCluster.Name == clusterOptions.Name {
			c.AbortWithStatus(http.StatusConflict)
			return
		}
	}

	v1Organization := v1.Organization{
		ID:   string(organization.UID),
		Name: organization.Name,
	}

	cluster, err := h.clusterService.CreateCluster(&v1Organization, &clusterOptions)
	if err != nil {
		h.logger.Error("error creating cluster", "name", clusterOptions.Name, "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	nodePoolOptions := clusterOptions.NodePoolOptions
	if nodePoolOptions == nil || len(*nodePoolOptions) == 0 {
		h.logger.Debug("using recommended node pool options")

		nodePoolOptions = util.Ptr(h.getRecommendedNodePools())
	}

	if clusterOptions.SingleNode != nil && *clusterOptions.SingleNode {
		h.logger.Debug("using single node pool")

		nodePoolOptions = util.Ptr([]v1.NodePoolOptions{
			{
				Name:         "single-node",
				Quantity:     1,
				ControlPlane: util.Ptr(true),
				Etcd:         util.Ptr(true),
			},
		})
	}

	hasErrors := false
	for _, nodePoolOption := range *nodePoolOptions {
		h.logger.Debug("creating cluster node pool", "name", nodePoolOption.Name)

		nodePool, err := h.clusterService.CreateNodePool(&v1Organization, cluster, &nodePoolOption)
		if err != nil {
			h.logger.Error("error creating node pool", "name", nodePoolOption.Name, "err", err)

			hasErrors = true
			break
		}

		h.logger.Debug("created cluster node pool", "name", nodePool.Name)

		cluster.NodePools = append(cluster.NodePools, *nodePool)
	}

	if clusterOptions.NoClusterApps == nil || !*clusterOptions.NoClusterApps {
		clusterDeployments, err := h.cloudService.GetClusterDeployments(&v1Organization, cluster)
		if err != nil {
			h.logger.Error("error getting cloud service cluster deployments", "err", err)

			hasErrors = true
		}

		if !hasErrors {
			for _, clusterDeployment := range *clusterDeployments {
				h.logger.Debug("creating cluster deployment", "name", *clusterDeployment.Name)

				clusterDeployment.ID = uuid.New()

				if clusterDeployment.Type == v1.DeploymentTypeContainerImage || clusterDeployment.Type == v1.DeploymentTypeKustomize {
					h.logger.Debug("creating repository for cluster deployment")

					err := deployment.CreateRepository(&clusterDeployment, h.gitProjectRoot)
					if err != nil {
						h.logger.Error("error creating repository for cluster deployment")

						hasErrors = true
						break
					}
				}

				err := h.db.Create(&clusterDeployment).Error
				if err != nil {
					h.logger.Error("error creating cluster deployment in database", "name", *clusterDeployment.Name, "err", err)

					hasErrors = true
					break
				}

				h.logger.Debug("created cluster deployment", "name", *clusterDeployment.Name, "id", clusterDeployment.ID)

				deploymentStatus := v1.DeploymentStatus{
					ID:           uuid.New(),
					DeploymentID: clusterDeployment.ID,
					State:        util.Ptr("created"),
					Health:       util.Ptr(v1.DeploymentStatusHealthWarning),
				}

				err = h.db.Create(&deploymentStatus).Error
				if err != nil {
					h.logger.Warn("error creating cluster deployment status", "err", err)

					continue
				}

				h.logger.Debug("created cluster deployment status", "id", deploymentStatus.ID)
			}
		}
	}

	if hasErrors {
		h.logger.Error("deleting cluster", "id", cluster.ID)

		err := h.clusterService.DeleteCluster(&v1Organization, cluster)
		if err != nil {
			h.logger.Warn("unexpected error deleting cluster", "err", err)
		}

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusCreated, cluster)
}

func (h *handler) GetClusterKubeconfig(c *gin.Context) {
	ctx := context.Background()

	clusterID := c.Param("clusterID")
	if clusterID == "" {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	cluster, err := h.clusterService.GetCluster(clusterID)
	if err != nil {
		h.logger.Error("error getting cluster from cluster service", "id", clusterID, "err", err)

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	objectKey := client.ObjectKey{
		Name: cluster.Organization,
	}

	var organization v1alpha1.Organization
	err = h.controllerClient.Get(ctx, objectKey, &organization)
	if err != nil {
		h.logger.Error("error getting organization from kubernetes", "err", err)

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	subject, err := h.getSubjectFromContext(c)
	if err != nil {
		h.logger.Error("error fetching user from context", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	isMember := h.isMember(subject, &organization)
	if !isMember {
		h.logger.Debug("subject is not a member of organization", "subject", subject, "organization", organization.Name)

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	h.logger.Debug("getting kubeconfig for cluster", "id", clusterID)

	kubeconfig, err := h.clusterService.GetKubeconfig(clusterID, time.Duration(time.Hour))
	if err != nil {
		h.logger.Error("unexpected error getting kubeconfig", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	b, err := yaml.Marshal(kubeconfig)
	if err != nil {
		h.logger.Error("error marshalling kubeconfig to yaml", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.Data(http.StatusOK, binding.MIMEYAML, b)
}

func (h *handler) DeleteCluster(c *gin.Context) {
	ctx := context.Background()

	clusterID := c.Param("clusterID")
	if clusterID == "" {
		c.AbortWithStatus(http.StatusBadRequest)

		return
	}

	cluster, err := h.clusterService.GetCluster(clusterID)
	if err != nil {
		h.logger.Error("error getting cluster from cluster service", "id", clusterID, "err", err)

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	objectKey := client.ObjectKey{
		Name: cluster.Organization,
	}

	var organization v1alpha1.Organization
	err = h.controllerClient.Get(ctx, objectKey, &organization)
	if err != nil {
		h.logger.Error("error getting organization from kubernetes", "err", err)

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	subject, err := h.getSubjectFromContext(c)
	if err != nil {
		h.logger.Debug("error fetching user from context", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	isMember := h.isMember(subject, &organization)
	if !isMember {
		h.logger.Debug("subject is not a member of organization", "subject", subject, "organization", organization.Name)

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	v1Organization := v1.Organization{
		ID:   string(organization.UID),
		Name: organization.Name,
	}

	err = h.clusterService.DeleteCluster(&v1Organization, cluster)
	if err != nil {
		h.logger.Error("unexpected error deleting cluster", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	h.logger.Debug("successfully deleted cluster", "id", cluster.ID)

	c.JSON(http.StatusAccepted, gin.H{})
}

func (h *handler) GetClusters(c *gin.Context) {
	ctx := context.Background()

	subject, err := h.getSubjectFromContext(c)
	if err != nil {
		h.logger.Debug("error getting subject from context", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	var organizationList v1alpha1.OrganizationList
	err = h.controllerClient.List(ctx, &organizationList)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	orgs := make(map[string]*v1alpha1.Organization)
	for i, organization := range organizationList.Items {
		orgs[organization.Name] = &organizationList.Items[i]
	}

	clusters, err := h.clusterService.GetAllClusters()
	if err != nil {
		h.logger.Error("unexpected error when getting clusters", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	filteredClusters := []v1.Cluster{}
	for _, cluster := range *clusters {
		if cluster.Organization == "" {
			continue
		}

		isMember := h.isMember(subject, orgs[cluster.Organization])
		if isMember {
			filteredClusters = append(filteredClusters, cluster)
		}
	}

	c.JSON(http.StatusOK, filteredClusters)
}

func (h *handler) GetCluster(c *gin.Context) {
	ctx := context.Background()

	id := c.Param("clusterID")
	if id == "" {
		h.logger.Error("empty cluster id")

		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	cluster, err := h.clusterService.GetCluster(id)
	if err != nil {
		h.logger.Error("error getting cluster from cluster service", "id", id, "err", err)

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	objectKey := client.ObjectKey{
		Name: cluster.Organization,
	}

	var organization v1alpha1.Organization
	err = h.controllerClient.Get(ctx, objectKey, &organization)
	if err != nil {
		h.logger.Error("error getting organization from kubernetes", "err", err)

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	subject, err := h.getSubjectFromContext(c)
	if err != nil {
		h.logger.Error("error fetching user from context", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	isMember := h.isMember(subject, &organization)
	if !isMember {
		h.logger.Debug("subject is not a member of organization", "subject", subject, "organization", organization.Name)

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	c.JSON(http.StatusOK, cluster)
}
