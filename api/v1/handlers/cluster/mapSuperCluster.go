package cluster

import (
	"Backend/api/v1/model"
	"Backend/internal"
	"crypto/tls"
	b64 "encoding/base64"
	"encoding/json"
	"io/ioutil"

	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

func MapSuperClusters(c *gin.Context) string {

	bearerToken := internal.CattleBearerToken
	rancherURL := internal.CattleUrl

	//Do external request
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, _ := http.NewRequest("GET", rancherURL+"/v3/clusters", nil)
	req.Header.Set(
		"Authorization", "Basic "+b64.StdEncoding.EncodeToString([]byte(bearerToken)),
	)
	// Response from the external request
	resp, extErr := client.Do(req)
	if extErr != nil {
		c.String(http.StatusBadGateway, fmt.Sprintf("There was an external error: %s", extErr.Error()))
		return ""
	}
	data, _ := ioutil.ReadAll(resp.Body)

	respErr := resp.Body.Close()
	if respErr != nil {
		return ""
	}

	var valuetok model.ReturnClusterResponse
	json.Unmarshal(data, &valuetok)

	c.JSON(http.StatusOK, gin.H{
		"clusters": valuetok.Data,
	})
	return string("")

}
