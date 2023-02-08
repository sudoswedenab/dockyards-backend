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
	Id   string `json:"id"`
	Name string `json:"name"`
}

func CreatedCluster(c *gin.Context) (string, string, error) {

	//Get the cookie
	tokenString, err := c.Cookie("access_token")
	if err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return "", "", err
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
		return "", "", err
	}

	// fmt.Println("lalal", token)
	claims := token.Claims.(jwt.MapClaims)
	fmt.Println(claims)

	var body model.NewClusterorius

	if c.Bind(&body) != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to read Body",
		})
		return "", "", err
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
		return "", "", err
	}

	fmt.Println("BODDY / BAYWATCH OLALALALA VI SKAPAR", string(reqBody))

	bearerToken := claims["aud"]
	rancherURL := os.Getenv("CATTLE_URL")

	//Do external request
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, _ := http.NewRequest("POST", rancherURL+"/v3/clusters", bytes.NewBuffer(reqBody))

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
		"Authorization": {"Basic " + b64.StdEncoding.EncodeToString([]byte(bearerToken.(string)))},
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
		return "", "", err
	}

	fmt.Println("Response HERE", resp)
	data, _ := ioutil.ReadAll(resp.Body)

	respErr := resp.Body.Close()
	if respErr != nil {
		return "", "", err
	}

	fmt.Println("COPY THAT,ROGER ROGER", string(data))
	// fmt.Println("EASY FIND", string(data))
	var responseBody Clusterino

	json.Unmarshal(data, &responseBody)

	fmt.Println("JETLAGG", responseBody)

	return responseBody.Id, responseBody.Name, err
}
