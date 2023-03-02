package cluster

import (
	"bytes"
	"crypto/tls"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"bitbucket.org/sudosweden/backend/api/v1/model"
	"bitbucket.org/sudosweden/backend/internal"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

type ClusterTwos struct {
	Name string `json:"name"`
	Id   string `json:"id"`
}

func CreatedNodePool(c *gin.Context, Id string, Name string, Err error) string {

	// Get the cookie
	tokenString, err := c.Cookie(internal.AccessTokenName)
	if err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return ""
	}

	// Parse takes the token string and a function for looking up the key.
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(internal.Secret), nil

	})
	if err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return ""
	}

	claims := token.Claims.(jwt.MapClaims)

	time.Sleep(2 * time.Second)

	//GeT FROM CREATECLUSTER  INFO
	body := model.NodePoolbody{
		ClusterId:               Id,
		ControlPlane:            true,
		DeleteNotReadyAfterSecs: 0,
		DrainBeforeDelete:       true,
		Etcd:                    true,
		HostnamePrefix:          Name + "-node-",
		Name:                    "",
		NamespaceId:             "",
		NodeTaints:              make([]string, 0),
		NodeTemplateId:          "cattle-global-nt:nt-5hxd5",
		Quantity:                3,
		Worker:                  true,
	}

	reqBody, err := json.Marshal(body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Not valid JSON! Failed to marshal Body",
		})
		return ""
	}

	bearerToken := claims["aud"]
	rancherURL := internal.CattleUrl

	//Do external request
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, _ := http.NewRequest("POST", rancherURL+"/v3/nodepools", bytes.NewBuffer(reqBody))

	//Setting the header
	req.Header = http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {"Basic " + b64.StdEncoding.EncodeToString([]byte(bearerToken.(string)))},
		"Accept":        {"application/json"},
		"Origin":        {internal.CattleUrl},
		"Connection":    {"keep-alive"},
		"TE":            {"trailers"},
	}

	// Response from the external request
	resp, extErr := client.Do(req)
	if extErr != nil {
		c.String(http.StatusBadGateway, fmt.Sprintf("There was an external error: %s", extErr.Error()))
		return ""
	}
	// The Response from the header are we Captureing here

	data, _ := ioutil.ReadAll(resp.Body)

	// Then closing the body and check for errors, if we get errors we print it here.
	respErr := resp.Body.Close()
	if respErr != nil {
		return ""
	}

	//Binding the struct in the top of the code to a varibel.
	var responseBody ClusterTwos
	// Converting the data from ones and zeros and binding it to the struct Responsbody
	json.Unmarshal(data, &responseBody)

	//Printing it out in our body (the response from the function)
	c.JSON(http.StatusOK, gin.H{
		"cluster":     "created successfully",
		"clusterName": responseBody.Name,
		"clusterId":   responseBody.Id,
	})
	//also printing it out as a string of data
	return ""
}
