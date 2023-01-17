package rancher

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
)

type RancherResponseToken struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	UserId      string `json:"userId"`
	Bearertoken string `json:"token"`
}

// CreateRancherToken godoc
//
// @Summary		Create a Rancher Token
// @Tags		RancherUser
// @Accept     	application/json
// @Produce		application/json
// @Success		200
// @Router			/ranchertoken [get]
func CreateRancherToken(c *gin.Context, rancherToken model.RRtoken) (string, string) {
	reqBody, err := json.Marshal(rancherToken)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Not valid JSON! Failed to marshal Body",
		})
		return "", ""
	}

	bearerToken := os.Getenv("CATTLE_BEARER_TOKEN")
	rancherURL := os.Getenv("CATTLE_URL")
	// Do external request
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, _ := http.NewRequest("POST", rancherURL+"/v3/tokens", bytes.NewBuffer(reqBody))
	req.Header.Set(
		"Authorization", "Basic "+b64.StdEncoding.EncodeToString([]byte(bearerToken)),
	)
	// Response from the external request
	resp, extErr := client.Do(req)
	if extErr != nil {
		c.String(http.StatusBadGateway, fmt.Sprintf("There was an external error: %s", extErr.Error()))
		return "", ""
	}
	data, _ := ioutil.ReadAll(resp.Body)
	respErr := resp.Body.Close()
	if respErr != nil {
		return "", ""
	}
	var valuetok RancherResponseToken
	json.Unmarshal(data, &valuetok)

	fmt.Println(valuetok)
	fmt.Printf("%T\n", valuetok.Id)
	c.JSON(http.StatusOK, gin.H{
		"UserStatus":  "user logged in",
		"UserToken":   "Created a token",
		"token":       valuetok,
		"TokenID":     valuetok.Id,
		"Bearertoken": valuetok.Bearertoken,
	})
	return valuetok.Bearertoken, valuetok.Id
}
