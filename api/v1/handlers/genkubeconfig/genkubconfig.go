package genkubeconfig

import (
	"crypto/tls"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"bitbucket.org/sudosweden/backend/api/v1/model"
	"bitbucket.org/sudosweden/backend/internal"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

func GenKubeConfig(c *gin.Context) string {
	//Get the cookie
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

	bearerToken := claims["aud"]
	rancherURL := internal.CattleUrl
	clusterID := c.Param("id")

	//Do external request
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, _ := http.NewRequest("POST", rancherURL+"/v3/clusters/"+clusterID+"?action=generateKubeconfig", nil)
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

	var value model.KubeConf
	json.Unmarshal(data, &value)

	c.YAML(http.StatusOK, gin.H{
		"Kubeconfig": value,
	})
	return string("")

}
