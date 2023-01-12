package handlers

import (
	"Backend/api/v1/model"
	"bytes"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
)

// RancherCreateUser godoc
//
//	@Summary		Create rancher user
//	@Tags			RancherUser
//	@Produce		text/plain
//	@Param			request	body	model.RancherUser	true "RancherUser model"
//	@Success		200
//	@Router			/create-user [post]
func RancherCreateUser(c *gin.Context) {
	user := model.RancherUser{}
	if c.Bind(&user) != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to read Body",
		})
		return
	}

	reqBody, err := json.Marshal(user)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Not valid JSON! Failed to marshal Body",
		})
		return
	}

	bearerToken := os.Getenv("CATTLE_BEARER_TOKEN")
	rancherURL := os.Getenv("CATTLE_URL")

	// Do external request
	client := &http.Client{}
	req, _ := http.NewRequest("POST", rancherURL+"/v3/users", bytes.NewBuffer(reqBody))
	req.Header.Set(
		"Authorization", "Basic "+b64.StdEncoding.EncodeToString([]byte(bearerToken)),
	)
	// Response from the external request
	resp, extErr := client.Do(req)
	if extErr != nil {
		c.String(http.StatusBadGateway, fmt.Sprintf("There was an external error: %s", extErr.Error()))
		return
	}

	if resp.Status == "201" {
		c.String(http.StatusCreated, fmt.Sprintf("User has been created:\n%s", reqBody))
		return
	}
}
