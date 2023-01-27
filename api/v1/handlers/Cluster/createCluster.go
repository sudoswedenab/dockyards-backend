package cluster

import (
	"Backend/api/v1/model"
	"bytes"
	"crypto/tls"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

type RancherResponse struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	UserId string `json:"userId"`
}

func CreateCluster(c *gin.Context, cluster model.Cluster) string {
	reqBody, err := json.Marshal(cluster)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Not valid JSON! Failed to marshal Body",
		})
		return ""
	}

	bearerToken := os.Getenv("CATTLE_BEARER_TOKEN")
	rancherURL := os.Getenv("CATTLE_URL")
	// Do external request
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, _ := http.NewRequest("GET", rancherURL+"/v3/clusters", bytes.NewBuffer(reqBody))
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
	var valuetok RancherResponse
	json.Unmarshal(data, &valuetok)

	fmt.Println(valuetok)
	fmt.Printf("%T\n", valuetok.Id)
	c.JSON(http.StatusOK, gin.H{
		"UserStatus": "Created Cluster",
		"UserToken":  "Created a token",
		"token":      valuetok,
		"TokenID":    valuetok.Id,
	})
	return valuetok.Id
}
