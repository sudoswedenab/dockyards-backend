package handlers

import (
	"net/http"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/names"
	"bitbucket.org/sudosweden/dockyards-backend/internal/util"
	"github.com/gin-gonic/gin"
)

func (h *handler) PostOrgClusters(c *gin.Context) {
	org := c.Param("org")
	if org == "" {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	var organization v1.Organization
	err := h.db.Take(&organization, "name = ?", org).Error
	if err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	user, err := h.getUserFromContext(c)
	if err != nil {
		h.logger.Debug("error fetching user from context", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	isMember, err := h.isMember(&user, &organization)
	if err != nil {
		h.logger.Error("error getting user membership", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
	}

	if !isMember {
		h.logger.Debug("user is not a member of organization", "user", user.ID, "organization", organization.ID)

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

	details, validName := names.IsValidName(clusterOptions.Name)
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
			details, validName := names.IsValidName(nodePoolOptions.Name)
			if !validName {
				c.JSON(http.StatusUnprocessableEntity, gin.H{
					"error":   "node pool name is not valid",
					"name":    nodePoolOptions.Name,
					"details": details,
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

	h.logger.Debug("forcing no ingress provider")
	clusterOptions.NoIngressProvider = util.Ptr(true)

	cluster, err := h.clusterService.CreateCluster(&organization, &clusterOptions)
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

		nodePool, err := h.clusterService.CreateNodePool(&organization, cluster, &nodePoolOption)
		if err != nil {
			h.logger.Error("error creating node pool", "name", nodePoolOption.Name, "err", err)

			hasErrors = true
			break
		}

		h.logger.Debug("created cluster node pool", "name", nodePool.Name)
	}

	if clusterOptions.NoClusterApps == nil || !*clusterOptions.NoClusterApps {
		clusterApps, err := h.cloudService.GetClusterDeployments(&organization, cluster)
		if err != nil {
			h.logger.Error("error getting cloud service cluster deployments ", "err", err)

			hasErrors = true
		}

		if !hasErrors {
			for _, clusterApp := range *clusterApps {
				h.logger.Debug("creating cluster app", "name", clusterApp.Name)

				err := h.db.Create(&clusterApp).Error
				if err != nil {
					h.logger.Error("error creating cluster app in database", "name", clusterApp.Name, "err", err)

					hasErrors = true
					break
				}

				h.logger.Debug("created cluster app", "name", clusterApp.Name, "id", clusterApp.ID)
			}
		}
	}

	if hasErrors {
		h.logger.Error("deleting cluster", "id", cluster.ID)

		err := h.clusterService.DeleteCluster(&organization, cluster)
		if err != nil {
			h.logger.Warn("unexpected error deleting cluster", "err", err)
		}

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusCreated, cluster)
}

func (h *handler) GetOrgClusterKubeConfig(c *gin.Context) {
	org := c.Param("org")
	if org == "" {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	clusterName := c.Param("cluster")
	if clusterName == "" {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	h.logger.Debug("get kubeconfig for cluster", "org", org, "cluster", clusterName)

	cluster := v1.Cluster{
		Organization: org,
		Name:         clusterName,
	}

	kubeConfig, err := h.clusterService.GetKubeConfig(&cluster)
	if err != nil {
		h.logger.Error("unexpected error getting kubeconfig", "err", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"kubeconfig": kubeConfig,
	})
}

func (h *handler) DeleteOrgClusters(c *gin.Context) {
	org := c.Param("org")
	if org == "" {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	clusterName := c.Param("cluster")
	if clusterName == "" {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	var organization v1.Organization
	err := h.db.Take(&organization, "name = ?", org).Error
	if err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)

		return
	}

	user, err := h.getUserFromContext(c)
	if err != nil {
		h.logger.Debug("error fetching user from context", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	isMember, err := h.isMember(&user, &organization)
	if err != nil {
		h.logger.Error("error getting user membership", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
	}

	if !isMember {
		h.logger.Debug("user is not a member of organization", "user", user.ID, "organization", organization.ID)

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	cluster := v1.Cluster{
		Organization: org,
		Name:         clusterName,
	}

	err = h.clusterService.DeleteCluster(&organization, &cluster)
	if err != nil {
		h.logger.Error("unexpected error deleting cluster", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	h.logger.Debug("successfully deleted cluster", "organization", org, "name", clusterName)

	c.JSON(http.StatusAccepted, gin.H{})
}

func (h *handler) GetClusters(c *gin.Context) {
	user, err := h.getUserFromContext(c)
	if err != nil {
		h.logger.Debug("error fetching user from context", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	var organizations []v1.Organization
	err = h.db.Model(&user).Association("Organizations").Find(&organizations)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	// create a map with organization names for quick lookup
	// the map using bools has no functional use, bool is the smallest datatype
	orgs := make(map[string]bool)
	for _, organization := range organizations {
		orgs[organization.Name] = true
	}

	clusters, err := h.clusterService.GetAllClusters()
	if err != nil {
		h.logger.Error("unexpected error when getting clusters", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"Error": err.Error(),
		})
		return
	}

	filteredClusters := []v1.Cluster{}
	for _, cluster := range *clusters {
		_, isMember := orgs[cluster.Organization]
		h.logger.Debug("checking cluster membership", "organization", cluster.Organization, "is_member", isMember)
		if isMember {
			filteredClusters = append(filteredClusters, cluster)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"clusters": filteredClusters,
	})
}

func (h *handler) GetCluster(c *gin.Context) {
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

	var organization v1.Organization
	err = h.db.Take(&organization, "name = ?", cluster.Organization).Error
	if err != nil {
		h.logger.Error("error taking organization from database", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	user, err := h.getUserFromContext(c)
	if err != nil {
		h.logger.Error("error fetching user from context", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	isMember, err := h.isMember(&user, &organization)
	if err != nil {
		h.logger.Error("error getting user membership", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
	}

	if !isMember {
		h.logger.Debug("user is not a member of organization", "user", user.ID, "organization", organization.ID)

		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	c.JSON(http.StatusOK, cluster)
}
