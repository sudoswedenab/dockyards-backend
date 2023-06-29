package handlers

import (
	"net/http"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1/model"
	"bitbucket.org/sudosweden/dockyards-backend/internal/names"
	"github.com/gin-gonic/gin"
)

func (h *handler) PostOrgClusters(c *gin.Context) {
	org := c.Param("org")
	if org == "" {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	var organization model.Organization
	err := h.db.Take(&organization, "name = ?", org).Error
	if err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	var clusterOptions model.ClusterOptions
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

	for _, nodePoolOptions := range clusterOptions.NodePoolOptions {
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

	cluster, err := h.clusterService.CreateCluster(&organization, &clusterOptions)
	if err != nil {
		h.logger.Error("error creating cluster", "name", clusterOptions.Name, "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	var controlPlaneNodePoolOptions model.NodePoolOptions
	if clusterOptions.SingleNode {
		controlPlaneNodePoolOptions = model.NodePoolOptions{
			Name:         "single-node",
			Quantity:     1,
			ControlPlane: true,
			Etcd:         true,
		}
	} else {
		controlPlaneNodePoolOptions = model.NodePoolOptions{
			Name:                       "control-plane",
			Quantity:                   3,
			ControlPlane:               true,
			Etcd:                       true,
			ControlPlaneComponentsOnly: true,
		}
	}

	controlPlaneNodePool, err := h.clusterService.CreateNodePool(&organization, cluster, &controlPlaneNodePoolOptions)
	if err != nil {
		h.logger.Error("error creating control plane node pool", "err", err)

		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	h.logger.Debug("created cluster control plane node pool", "name", controlPlaneNodePool.Name)

	if !clusterOptions.SingleNode {
		nodePoolOptions := clusterOptions.NodePoolOptions
		if len(nodePoolOptions) == 0 {
			nodePoolOptions = []model.NodePoolOptions{
				{
					Name:     "worker",
					Quantity: 2,
				},
				{
					Name:         "load-balancer",
					Quantity:     2,
					LoadBalancer: true,
				},
			}
		}

		for _, nodePoolOption := range nodePoolOptions {
			h.logger.Debug("creating cluster node pool", "name", nodePoolOption.Name)

			nodePool, err := h.clusterService.CreateNodePool(&organization, cluster, &nodePoolOption)
			if err != nil {
				h.logger.Error("error creating node pool", "name", nodePoolOption.Name, "err", err)

				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}
			h.logger.Debug("created cluster node pool", "name", nodePool.Name)
		}
	}

	if !clusterOptions.NoClusterApps {
		clusterApps, err := h.cloudService.GetClusterApps(&organization, cluster)
		if err != nil {
			h.logger.Error("error getting cloud service cluster apps", "err", err)

			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		for _, clusterApp := range *clusterApps {
			h.logger.Debug("creating cluster app", "name", clusterApp.Name)

			err := h.db.Create(&clusterApp).Error
			if err != nil {
				h.logger.Error("error creating cluster app in database", "name", clusterApp.Name, "err", err)

				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}

			h.logger.Debug("created cluster app", "name", clusterApp.Name, "id", clusterApp.ID)
		}
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

	cluster := model.Cluster{
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

	cluster := model.Cluster{
		Organization: org,
		Name:         clusterName,
	}

	err := h.clusterService.DeleteCluster(&cluster)
	if err != nil {
		h.logger.Error("unexpected error deleting cluster", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	h.logger.Debug("successfully deleted cluster", "organization", org, "name", clusterName)

	c.Status(http.StatusAccepted)
}

func (h *handler) GetClusters(c *gin.Context) {
	user, err := h.getUserFromContext(c)
	if err != nil {
		h.logger.Debug("error fetching user from context", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	var organizations []model.Organization
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

	filteredClusters := []model.Cluster{}
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

func (s *sudo) GetClusters(c *gin.Context) {
	clusters, err := s.clusterService.GetAllClusters()
	if err != nil {
		c.AbortWithStatus(http.StatusTeapot)
		return
	}

	c.JSON(http.StatusOK, clusters)
}

func (s *sudo) GetKubeconfig(c *gin.Context) {
	org := c.Param("org")
	name := c.Param("name")
	cluster := model.Cluster{
		Organization: org,
		Name:         name,
	}

	kubeconfig, err := s.clusterService.GetKubeConfig(&cluster)
	if err != nil {
		s.logger.Debug("error getting kubeconfig", "org", org, "name", name, "err", err)
		c.AbortWithStatus(http.StatusTeapot)
		return
	}

	c.JSON(http.StatusOK, kubeconfig)
}
