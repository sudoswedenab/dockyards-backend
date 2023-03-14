package cluster

import (
	"fmt"
	"net/http"

	"bitbucket.org/sudosweden/backend/api/v1/model"
	"bitbucket.org/sudosweden/backend/internal"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

func (h *handler) CreateCluster(c *gin.Context) {
	// Get the cookie
	tokenString, err := c.Cookie(h.accessTokenName)
	if err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	// Parse takes the token string and a function for looking up the key.
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(internal.Secret), nil //TODO

	})
	if err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	claims := token.Claims.(jwt.MapClaims)

	var body model.ClusterData
	if c.Bind(&body) != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to read Body",
		})
		return
	}

	bearerToken := claims["aud"].(string)

	nodePool, err := h.rancherService.RancherCreateCluster(body, bearerToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to read Body",
		})
		return
	}

	clusterTwos, err := h.rancherService.RancherCreateNodePool(bearerToken, nodePool.Id, nodePool.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to read Body",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"cluster":     "created successfully",
		"clusterName": clusterTwos.Name,
		"clusterId":   clusterTwos.Id,
	})
}
