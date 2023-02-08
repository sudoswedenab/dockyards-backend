package cluster

import (
	"Backend/api/v1/model"
	"bytes"
	"crypto/tls"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"io/ioutil"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

type ClusterTwos struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

func CreatedNodePool(c *gin.Context, Name string, Id string, Err error) string {

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

	//GeT FROM CLUSTER ONE INFO
	time.Sleep(5 * time.Second)
	// ClusterOne := CreatedCluster(c)(Name, Id, err)
	// Id, Name, err := CreatedCluster(c)
	// fmt.Println(Id)
	// fmt.PrintIn(Name)

	body := model.NodePoolbody{
		ControlPlane:            true,
		DeleteNotReadyAfterSecs: 0,
		DrainBeforeDelete:       false,
		Etcd:                    true,
		Quantity:                3,
		Worker:                  true,
		Type:                    "nodePool",
		ClusterId:               Id,
		HostnamePrefix:          Name + "-node-",
		NodeTemplateId:          "cattle-global-nt:nt-zd2tl",
	}
	// {"controlPlane": "true",
	// "deleteNotReadyAfterSecs": 0,
	// "drainBeforeDelete": "false",
	// "etcd": "true",
	// "quantity": 3,
	// "worker": "true",
	//  "type": "nodePool",
	//  "clusterId": ID ,
	//  "hostnamePrefix": NAME + "-node-",
	//  "nodeTemplateId": "cattle-global-nt:nt-zd2tl"}

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

	bearerToken := claims["aud"]
	rancherURL := os.Getenv("CATTLE_URL")

	//Do external request
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, _ := http.NewRequest("POST", rancherURL+"/v3/nodepool", bytes.NewBuffer(reqBody))

	//Setting the header
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
		return ""
	}
	// The Response from the header are we Captureing here
	fmt.Println("Response HERE", resp)
	data, _ := ioutil.ReadAll(resp.Body)

	// Then closing the body and check for errors, if we get errors we print it here.
	respErr := resp.Body.Close()
	if respErr != nil {
		return ""
	}

	//Printing out the data from api and convert it to a string of data instead of byte. In our console
	fmt.Println("COPY THAT,ROGER ROGER", string(data))
	// fmt.Println("EASY FIND", string(data))

	//Binding the struct in the top of the code to a varibel.
	var responseBody ClusterTwos
	// Converting the data from ones and zeros and binding it to the struct Responsbody
	json.Unmarshal(data, &responseBody)

	fmt.Println("JETLAGG", responseBody)
	//Printing it out in our body (the response from the function)
	c.JSON(http.StatusOK, gin.H{
		"clusterID":   responseBody.Id,
		"clusterName": responseBody.Name,
	})
	//also printing it out as a string of data
	return ""
}
