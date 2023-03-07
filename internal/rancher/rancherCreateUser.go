package rancher

import (
	"bytes"
	"crypto/tls"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"bitbucket.org/sudosweden/backend/api/v1/model"
	"bitbucket.org/sudosweden/backend/internal"
	"github.com/gin-gonic/gin"
)

type RancherUserResponse struct {
	Id string `json:"id"`
}

func RancherCreateUser(c *gin.Context, user model.RancherUser) string {

	reqBody, err := json.Marshal(user)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Not valid JSON! Failed to marshal Body",
		})
		return ""
	}

	bearerToken := internal.CattleBearerToken
	rancherURL := internal.CattleUrl
	// Do external request
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, _ := http.NewRequest("POST", rancherURL+"/v3/users", bytes.NewBuffer(reqBody))
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
	var rancherUserResponse RancherUserResponse
	json.Unmarshal(data, &rancherUserResponse)
	fmt.Printf("%T\n", rancherUserResponse.Id)

	if resp.Status == "201" {
		c.String(http.StatusCreated, fmt.Sprintf("User has been created:\n%s", reqBody))
		return ""
	}
	return rancherUserResponse.Id
}
