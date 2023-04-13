package handlers

import (
	"fmt"
	"github.com/golang-jwt/jwt"
	"net/http"
	"time"

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
			"name":    clusterOptions.Name,
			"details": "name must contain only alphanumber characters and the '-' character. name must be max 63 characters long",
		})
		return
	}

	for _, nodePoolOptions := range clusterOptions.NodePoolOptions {
		if !internal.IsValidName(nodePoolOptions.Name) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"error":   "node pool name is not valid",
				"name":    nodePoolOptions.Name,
				"details": "name must contain only alphanumber characters and the '-' character. name must be max 63 characters long",
			})
			return
		}
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
		h.logger.Error("unexpected error deleting cluster", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	h.logger.Debug("successfully deleted cluster", "name", name)
	c.JSON(http.StatusOK, gin.H{
		"status": "Cluster Deleted",
	})
}

func (h *handler) GetClusters(c *gin.Context) {
	// If filter len is 0, list all
	clusters, err := h.clusterService.GetAllClusters()
	if err != nil {
		h.logger.Error("unexpected error when getting clusters", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"Error": err.Error(),
		})
		return
	}

	h.logger.Debug("successfully got clusters", "clusters", clusters)
	c.JSON(http.StatusOK, gin.H{
		"clusters": clusters,
	})
}

func (h *handler) PostRefresh(c *gin.Context) {
	// Get the cookie
	refreshToken, err := c.Cookie(internal.RefreshTokenName)

	if err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
	}

	// Parse the token string and a function for looking for the key.
	token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// hmacSampleSecret is a []byte containing your incl secret key
		return []byte(internal.RefSecret), nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		// Get the user record from database or
		// why float64 and not int64?
		if float64(time.Now().Unix()) > claims["exp"].(float64) {
			c.AbortWithStatus(http.StatusUnauthorized)
		}

		//Find the user with token sub
		var user model.User

		First := h.db.First(&user, claims["sub"])

		// replace with jwt response
		if First.Error == nil {
			newTokenPair, err := h.generateTokenPair(user)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": err.Error(),
				})
			}
			c.JSON(http.StatusOK, gin.H{
				"Login":                   "Success",
				internal.AccessTokenName:  newTokenPair[internal.AccessTokenName],
				internal.RefreshTokenName: newTokenPair[internal.RefreshTokenName],
			})
		}
	}
}

func (h *handler) generateTokenPair(user model.User) (map[string]string, error) {
	// Create token
	token := jwt.New(jwt.SigningMethodHS256)

	// Set claims
	claims := token.Claims.(jwt.MapClaims)
	claims["sub"] = user.ID
	claims["name"] = user.Name
	claims["admin"] = false
	claims["exp"] = time.Now().Add(time.Minute * 15).Unix()

	// Generate encoded token and send it as response.
	t, err := token.SignedString([]byte(internal.Secret))
	if err != nil {
		return nil, err
	}

	refreshToken := jwt.New(jwt.SigningMethodHS256)
	rtClaims := refreshToken.Claims.(jwt.MapClaims)
	rtClaims["sub"] = user.ID
	rtClaims["exp"] = time.Now().Add(time.Hour * 1).Unix()

	rt, err := refreshToken.SignedString([]byte(internal.RefSecret))
	if err != nil {
		return nil, err
	}

	return map[string]string{
		internal.AccessTokenName:  t,
		internal.RefreshTokenName: rt,
	}, nil
}
