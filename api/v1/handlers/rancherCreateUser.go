package handlers

import (
	"bytes"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
)

type RancherUser struct {
	Enabled            bool   `json:"enabled"`
	MustChangePassword bool   `json:"mustChangePassword"`
	Name               string `json:"name"`
	Password           string `json:"password"`
	Username           string `json:"username"`
}

// RancherCreateUser godoc
//
//	@Summary		Create rancher user
//	@Tags			Rancher
//	@Produce		text/plain
//	@Success		200
//	@Router			/create-user [post]
func RancherCreateUser(c *gin.Context) {
	user := RancherUser{}
	if c.Bind(&user) != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to read Body",
		})
		return
	}

	reqBody, err := json.Marshal(user)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to marshal Body",
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
	resp, _ := client.Do(req)

	if resp.Status == "201" {
		fmt.Println("be happy.")
	}
}
