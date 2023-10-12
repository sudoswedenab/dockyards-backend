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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

// +kubebuilder:rbac:groups=dockyards.io,resources=organizations,verbs=get;list;watch
// +kubebuilder:rbac:groups=dockyards.io,resources=clusters,verbs=get;list;watch;create;delete
// +kubebuilder:rbac:groups=dockyards.io,resources=nodepools,verbs=create

func (h *handler) toV1Cluster(organization *v1alpha1.Organization, cluster *v1alpha1.Cluster, nodePoolList *v1alpha1.NodePoolList) *v1.Cluster {
	v1Cluster := v1.Cluster{
		Id:           string(cluster.UID),
		Name:         cluster.Name,
		Organization: organization.Name,
		CreatedAt:    cluster.CreationTimestamp.Time,
		Version:      cluster.Status.Version,
	}

	condition := meta.FindStatusCondition(cluster.Status.Conditions, v1alpha1.ReadyCondition)
	if condition != nil {
		v1Cluster.State = condition.Message
	}

	if nodePoolList != nil && len(nodePoolList.Items) > 0 {
		nodePools := make([]v1.NodePool, len(nodePoolList.Items))
		for i, nodePool := range nodePoolList.Items {
			nodePools[i] = *h.toV1NodePool(&nodePool, cluster, nil)
		}

		v1Cluster.NodePools = nodePools
	}

	return &v1Cluster
}

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

	cluster := v1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterOptions.Name,
			Namespace: organization.Status.NamespaceRef,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         v1alpha1.GroupVersion.String(),
					Kind:               v1alpha1.OrganizationKind,
					Name:               organization.Name,
					UID:                organization.UID,
					BlockOwnerDeletion: util.Ptr(true),
				},
			},
		},
		Spec: v1alpha1.ClusterSpec{},
	}

	err = h.controllerClient.Create(ctx, &cluster)
	if err != nil {
		h.logger.Error("error creating cluster", "err", err)

		if apierrors.IsAlreadyExists(err) {
			c.AbortWithStatus(http.StatusConflict)
			return
		}

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

	nodePoolList := v1alpha1.NodePoolList{
		Items: make([]v1alpha1.NodePool, len(*nodePoolOptions)),
	}

	hasErrors := false
	for i, nodePoolOption := range *nodePoolOptions {
		nodePool := v1alpha1.NodePool{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cluster.Name + "-" + nodePoolOption.Name,
				Namespace: cluster.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion:         v1alpha1.GroupVersion.String(),
						Kind:               v1alpha1.ClusterKind,
						Name:               cluster.Name,
						UID:                cluster.UID,
						BlockOwnerDeletion: util.Ptr(true),
					},
				},
			},
			Spec: v1alpha1.NodePoolSpec{
				Replicas: util.Ptr(int32(nodePoolOption.Quantity)),
			},
		}

		if nodePoolOption.ControlPlane != nil {
			nodePool.Spec.ControlPlane = *nodePoolOption.ControlPlane
		}

		if nodePoolOption.Etcd != nil {
			nodePool.Spec.ControlPlane = *nodePoolOption.Etcd
		}

		if nodePoolOption.LoadBalancer != nil {
			nodePool.Spec.LoadBalancer = *nodePoolOption.LoadBalancer
		}

		if nodePoolOption.ControlPlaneComponentsOnly != nil {
			nodePool.Spec.DedicatedRole = *nodePoolOption.ControlPlaneComponentsOnly
		}

		err := h.controllerClient.Create(ctx, &nodePool)
		if err != nil {
			h.logger.Error("error creating node pool", "err", err)

			hasErrors = true
			break
		}

		nodePoolList.Items[i] = nodePool

		h.logger.Debug("created cluster node pool", "id", nodePool.UID)
	}

	if clusterOptions.NoClusterApps == nil || !*clusterOptions.NoClusterApps {
		clusterDeployments, err := h.cloudService.GetClusterDeployments(&organization, &cluster, &nodePoolList)
		if err != nil {
			h.logger.Error("error getting cloud service cluster deployments", "err", err)

			hasErrors = true
		}

		if !hasErrors {
			for _, clusterDeployment := range *clusterDeployments {
				h.logger.Debug("creating cluster deployment", "name", *clusterDeployment.Name)

				clusterDeployment.Id = uuid.New()

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

				h.logger.Debug("created cluster deployment", "name", *clusterDeployment.Name, "id", clusterDeployment.Id)

				deploymentStatus := v1.DeploymentStatus{
					Id:           uuid.New(),
					DeploymentId: clusterDeployment.Id,
					State:        util.Ptr("created"),
					Health:       util.Ptr(v1.DeploymentStatusHealthWarning),
				}

				err = h.db.Create(&deploymentStatus).Error
				if err != nil {
					h.logger.Warn("error creating cluster deployment status", "err", err)

					continue
				}

				h.logger.Debug("created cluster deployment status", "id", deploymentStatus.Id)
			}
		}
	}

	if hasErrors {
		h.logger.Error("deleting cluster", "id", cluster.UID)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	v1Cluster := v1.Cluster{
		Id: string(cluster.UID),
	}

	c.JSON(http.StatusCreated, v1Cluster)
}

func (h *handler) GetClusterKubeconfig(c *gin.Context) {
	ctx := context.Background()

	clusterID := c.Param("clusterID")
	if clusterID == "" {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	matchingFields := client.MatchingFields{
		"metadata.uid": clusterID,
	}

	var clusterList v1alpha1.ClusterList
	err := h.controllerClient.List(ctx, &clusterList, matchingFields)
	if err != nil {
		h.logger.Error("error listing clusters", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	if len(clusterList.Items) != 1 {
		h.logger.Debug("expected exactly one cluster", "count", len(clusterList.Items))

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	cluster := clusterList.Items[0]

	organization, err := h.getOwnerOrganization(ctx, &cluster)
	if err != nil {
		h.logger.Error("error getting owner organization", "err", err)

		if apierrors.IsNotFound(err) {
			c.AbortWithStatus(http.StatusUnauthorized)
		}

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	subject, err := h.getSubjectFromContext(c)
	if err != nil {
		h.logger.Error("error fetching user from context", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	isMember := h.isMember(subject, organization)
	if !isMember {
		h.logger.Debug("subject is not a member of organization", "subject", subject, "organization", organization.Name)

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	kubeconfig, err := h.clusterService.GetKubeconfig(cluster.Status.ClusterServiceID, time.Duration(time.Hour))
	if err != nil {
		h.logger.Error("error getting kubeconfig", "err", err)

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

	matchingFields := client.MatchingFields{
		"metadata.uid": clusterID,
	}

	var clusterList v1alpha1.ClusterList
	err := h.controllerClient.List(ctx, &clusterList, matchingFields)
	if err != nil {
		h.logger.Error("error listing clusters", "err", err)

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	if len(clusterList.Items) != 1 {
		h.logger.Debug("expected exactly one cluster", "count", len(clusterList.Items))

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	cluster := clusterList.Items[0]

	organization, err := h.getOwnerOrganization(ctx, &cluster)
	if err != nil {
		h.logger.Error("error getting owner organization", "err", err)
	}

	subject, err := h.getSubjectFromContext(c)
	if err != nil {
		h.logger.Debug("error fetching user from context", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	isMember := h.isMember(subject, organization)
	if !isMember {
		h.logger.Debug("subject is not a member of organization", "subject", subject, "organization", organization.Name)

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	err = h.controllerClient.Delete(ctx, &cluster)
	if err != nil {
		h.logger.Error("error deleting cluster", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	h.logger.Debug("deleted cluster", "id", cluster.UID)

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

	matchingFields := client.MatchingFields{
		"spec.memberRefs": subject,
	}

	var organizationList v1alpha1.OrganizationList
	err = h.controllerClient.List(ctx, &organizationList, matchingFields)
	if err != nil {
		h.logger.Error("error listing organizations", "err", err)

		c.Status(http.StatusInternalServerError)
		return
	}

	clusters := []v1.Cluster{}

	for _, organization := range organizationList.Items {
		var clusterList v1alpha1.ClusterList
		err = h.controllerClient.List(ctx, &clusterList, client.InNamespace(organization.Status.NamespaceRef))
		if err != nil {
			h.logger.Error("error listing clusters", "err", err)

			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		for _, cluster := range clusterList.Items {
			clusters = append(clusters, *h.toV1Cluster(&organization, &cluster, nil))
		}
	}

	c.JSON(http.StatusOK, clusters)
}

func (h *handler) GetCluster(c *gin.Context) {
	ctx := context.Background()

	clusterID := c.Param("clusterID")
	if clusterID == "" {
		h.logger.Error("empty cluster id")

		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	matchingFields := client.MatchingFields{
		"metadata.uid": clusterID,
	}

	var clusterList v1alpha1.ClusterList
	err := h.controllerClient.List(ctx, &clusterList, matchingFields)
	if err != nil {
		h.logger.Error("error listing clusters", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	if len(clusterList.Items) != 1 {
		h.logger.Debug("expected exactly one cluster", "count", len(clusterList.Items))

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	cluster := clusterList.Items[0]

	organization, err := h.getOwnerOrganization(ctx, &cluster)
	if err != nil {
		h.logger.Error("error getting owner organization", "err", err)

		if apierrors.IsNotFound(err) {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	subject, err := h.getSubjectFromContext(c)
	if err != nil {
		h.logger.Error("error fetching user from context", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	isMember := h.isMember(subject, organization)
	if !isMember {
		h.logger.Debug("subject is not a member of organization", "subject", subject, "organization", organization.Name)

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	matchingFields = client.MatchingFields{
		"metadata.ownerReferences.uid": clusterID,
	}

	var nodePoolList v1alpha1.NodePoolList
	err = h.controllerClient.List(ctx, &nodePoolList, matchingFields)
	if err != nil {
		h.logger.Error("error listing node pools", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	v1Cluster := h.toV1Cluster(organization, &cluster, &nodePoolList)

	c.JSON(http.StatusOK, v1Cluster)
}
