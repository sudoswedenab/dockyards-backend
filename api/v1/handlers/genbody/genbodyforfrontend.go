package genbody

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func GenBodyForCluster(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"dockerRootDir":             "/var/lib/docker",
		"type":                      "cluster",
		"name":                      "frontend",
		"clusterTemplateRevisionId": "cattle-global-data:ctr-8cf7d",
		"clusterTemplateId":         "cattle-global-data:ct-mk2bd",
	})
}
