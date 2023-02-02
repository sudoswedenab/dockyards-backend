package cluster

import (
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

type ClusterResponse struct {
	Data []Data
}

type Data struct {
	Name                 string       `json:"name"`
	CreatorId            string       `json:"creatorId"`
	Created              string       `json:"created"`
	State                string       `json:"state"`
	NodeCount            int          `json:"nodeCount"`
	Transitioning        string       `json:"transitioning"`
	TransitioningMessage string       `json:"transitioningMessage"`
	Conditions           []Conditions `json:"conditions"`
}

type Conditions struct {
	Status string `json:"status"`
	Type   string `json:"type"`
}

func MapGetClusters(c *gin.Context) string {

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

	bearerToken := claims["aud"]

	// bearerToken := os.Getenv("CATTLE_BEARER_TOKEN")
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
	var valuetok ClusterResponse
	json.Unmarshal(data, &valuetok)

	fmt.Println(valuetok)

	c.JSON(http.StatusOK, gin.H{
		"clusters": valuetok.Data,
	})
	return string("")

}
