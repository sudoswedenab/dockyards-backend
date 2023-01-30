package cluster

import (
	"Backend/api/v1/model"
	"crypto/tls"
	b64 "encoding/base64"
	"encoding/json"
	"io/ioutil"

	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

type RancherResponses struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	UserId string `json:"userId"`
}

func CreateClusters(c *gin.Context, cluster model.Cluster) string {
	// reqBody, err := json.Marshal(cluster)
	// if err != nil {
	// 	c.JSON(http.StatusBadRequest, gin.H{
	// 		"error": "Not valid JSON! Failed to marshal Body",
	// 	})
	// 	return ""
	// }
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

		return nil, nil
	})
	if err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return ""
	}
	claims := token.Claims.(jwt.MapClaims)
	fmt.Println(claims)
	bearerToken := claims["aud"]
	rancherURL := os.Getenv("CATTLE_URL")

	//Do external request
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, _ := http.NewRequest("GET", rancherURL+"/v3/clusters", nil)
	req.Header.Set(
		"Authorization", "Basic "+b64.StdEncoding.EncodeToString([]byte(bearerToken.(string))),
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

	fmt.Println("EASY FIND", string(data))
	var valuetok RancherResponses
	json.Unmarshal(data, &valuetok)

	fmt.Println(valuetok)
	fmt.Printf("%T\n", valuetok.Id)
	c.JSON(http.StatusOK, gin.H{

		"VALUE OK": valuetok,
		"TokenID":  valuetok.Id,
	})
	return valuetok.Id

}
