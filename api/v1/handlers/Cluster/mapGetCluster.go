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

type ClusterResponse struct {
	Data []Data
}

type Data struct {
	Name string `json:"name"`
}

func MapGetClusters(c *gin.Context, cluster model.Cluster) string {
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
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
		return []byte(os.Getenv("SECERET")), nil
	})
	claims := token.Claims.(jwt.MapClaims)
	fmt.Println(claims)
	bearerToken := os.Getenv("CATTLE_BEARER_TOKEN")
	rancherURL := os.Getenv("CATTLE_URL")

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

	fmt.Println("EASY FIND", string(data))
	var valuetok ClusterResponse
	json.Unmarshal(data, &valuetok)

	fmt.Println(valuetok)

	c.JSON(http.StatusOK, gin.H{
		"Clusters": valuetok.Data,
	})
	return string("")

}
