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

type ClusterTwoos struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

func CreatedClusterTwo(c *gin.Context) string {

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

	// fmt.Println("lalal", token)
	claims := token.Claims.(jwt.MapClaims)
	fmt.Println(claims)

	//GeT FROM CLUSTER ON INFO
	ClusterOne := CreatedCluster(c)
	fmt.Println(ClusterOne)

	var body model.ClusterTwo

	if c.Bind(&body) != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to read Body",
		})
		return ""
	}

	// jsonData, err := io.ReadAll(c.Request.Body)
	// if err != nil {
	// 	c.JSON(http.StatusBadRequest, gin.H{
	// 		"error": "Not valid JSON! Failed to READ Body",
	// 	})
	// 	return ""
	// }

	reqBody, err := json.Marshal(body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Not valid JSON! Failed to marshal Body",
		})
		return ""
	}

	fmt.Println("BODDY / BAYWATCH OLALALALA VI SKAPAR", string(reqBody))

	bearerToken := os.Getenv("CATTLE_BEARER_TOKEN")
	rancherURL := os.Getenv("CATTLE_URL")

	//Do external request
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, _ := http.NewRequest("POST", rancherURL+"v3/nodepool", bytes.NewBuffer(reqBody))

	// req.Header.Set(
	// 	"Authorization", "Basic "+b64.StdEncoding.EncodeToString([]byte(bearerToken)),
	// )
	// req.Header.Set("Content-Type", "application/json")
	// req.Header.Set("Accept", "application/json")
	// req.Header.Set("Origin", "https://ss-di-rancher.sudobash.io")
	// req.Header.Set("Connection", "keep-alive")
	// req.Header.Set("Referer", "https://ss-di-rancher.sudobash.io/g/clusters/add/launch/openstack?clusterTemplateRevision=cattle-global-data%3Actr-7xnpl")
	// req.Header.Set("TE", "trailers")

	req.Header = http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {"Basic " + b64.StdEncoding.EncodeToString([]byte(bearerToken))},
		"Accept":        {"application/json"},
		"Origin":        {"https://ss-di-rancher.sudobash.io"},
		"Connection":    {"keep-alive"},
		// "Referer":       {"https://ss-di-rancher.sudobash.io/g/clusters/add/launch/openstack?clusterTemplateRevision=cattle-global-data%3Actr-7xnpl"},
		"TE": {"trailers"},
	}

	fmt.Println("HEADERN VI SKAPAR", req.Header)
	// Response from the external request
	resp, extErr := client.Do(req)
	if extErr != nil {
		c.String(http.StatusBadGateway, fmt.Sprintf("There was an external error: %s", extErr.Error()))
		return ""
	}

	fmt.Println("Response HERE", resp)
	data, _ := ioutil.ReadAll(resp.Body)

	respErr := resp.Body.Close()
	if respErr != nil {
		return ""
	}

	fmt.Println("COPY THAT,ROGER ROGER", string(data))
	// fmt.Println("EASY FIND", string(data))
	var responseBody ClusterTwoos

	json.Unmarshal(data, &responseBody)

	fmt.Println("JETLAGG", responseBody)

	c.JSON(http.StatusOK, gin.H{
		"clusterID":   responseBody.Id,
		"clusterName": responseBody.Name,
	})
	return ""
}
