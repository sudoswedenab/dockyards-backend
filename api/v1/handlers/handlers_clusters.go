package handlers

import (
	"net/http"

	"bitbucket.org/sudosweden/backend/api/v1/model"
	"bitbucket.org/sudosweden/backend/internal"
	"github.com/gin-gonic/gin"
)

func (h *handler) PostClusters(c *gin.Context) {
	var clusterOptions model.ClusterOptions
	if c.BindJSON(&clusterOptions) != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to read Body",
		})
		return
	}

	h.logger.Debug("create cluster", "name", clusterOptions.Name, "version", clusterOptions.Version)

	if !internal.IsValidName(clusterOptions.Name) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":   "name is not valid",
			"details": "name must contain only alphanumber characters and the '-' character. name must be max 63 characters long",
		})
		return
	}

	cluster, err := h.clusterService.CreateCluster(&clusterOptions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
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

	controlPlaneNodePool, err := h.clusterService.CreateNodePool(cluster, &controlPlaneNodePoolOptions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
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
			}
		}

		for _, nodePoolOption := range nodePoolOptions {
			nodePool, err := h.clusterService.CreateNodePool(cluster, &nodePoolOption)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
				})
				return
			}
			h.logger.Debug("created cluster node pool", "name", nodePool.Name)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"cluster":     "created successfully",
		"clusterName": cluster.Name,
	})
}

func (h *handler) GetClusterKubeConfig(c *gin.Context) {
	name := c.Param("name")
	h.logger.Debug("get kubeconfig for cluster", "name", name)

	cluster := model.Cluster{
		Name: name,
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

func (h *handler) DeleteCluster(c *gin.Context) {
	name := c.Param("name")

	err := h.clusterService.DeleteCluster(name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "Cluster Deleted",
	})
}

func (h *handler) GetAllClusters(c *gin.Context) {
	// If filter len is 0, list all
	clusters, err := h.clusterService.GetAllClusters()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"Error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"clusters": clusters,
	})
}
