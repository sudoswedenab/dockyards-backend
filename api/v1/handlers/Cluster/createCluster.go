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
	"github.com/golang-jwt/jwt"
)

type Clusterino struct {
	model.ClusterRespAll
}

func CreatedCluster(c *gin.Context, cluster model.ClusterData) string {

	//Get the cookie
	tokenString, err := c.Cookie("access_token")
	if err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return ""
	}

	// Parse takes the token string and a function for looking up the key.
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(os.Getenv("SECERET")), nil

	})
	if err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return ""
	}

	fmt.Println("lalal", token)
	claims := token.Claims.(jwt.MapClaims)
	fmt.Println(claims)

	reqBody, err := json.Marshal(cluster)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Not valid JSON! Failed to marshal Body",
		})
		return ""
	}

	bearerToken := os.Getenv("CATTLE_BEARER_TOKEN")
	rancherURL := os.Getenv("CATTLE_URL")

	//Do external request
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, _ := http.NewRequest("POST", rancherURL+"/v3/clusters", bytes.NewBuffer(reqBody))
	req.Header.Set(
		"Authorization", "Basic "+b64.StdEncoding.EncodeToString([]byte(bearerToken)),
	)
	req.Header.Set("Content-Type","application/json")
	req.Header.Set("Accept","application/json")
	req.Header.Set("Origin","https://ss-di-rancher.sudobash.io")
	req.Header.Set("Connection", "keep-alive" )
	req.Header.Set("Referer","https://ss-di-rancher.sudobash.io/g/clusters/add/launch/openstack?clusterTemplateRevision=cattle-global-data%3Actr-7xnpl', 'TE': 'trailers")
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

	// fmt.Println("EASY FIND", string(data))
	var valuetok Clusterino

	json.Unmarshal(data, &valuetok)

	// fmt.Println(valuetok)

	c.JSON(http.StatusOK, gin.H{
		"clusters": valuetok.Data,
	})
	return Clusterino.Data
}
