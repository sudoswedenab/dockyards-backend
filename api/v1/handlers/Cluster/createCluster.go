package cluster

import (
	"Backend/api/v1/model"
	"crypto/tls"
	"encoding/json"
	b64 "encoding/json"
	"io/ioutil"

	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

type RancherResponse struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	UserId string `json:"userId"`
}

func CreateCluster(c *gin.Context, cluster model.Cluster) string {
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
