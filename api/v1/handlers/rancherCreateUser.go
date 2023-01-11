package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"go/types"
	"net/http"
	"os"
)

// RancherCreateUser godoc
//
//	@Summary		Create rancher user
//	@Tags			Rancher
//	@Produce		text/plain
//	@Success		200
//	@Router			/create-user [post]
func RancherCreateUser(c *gin.Context) {
	var body struct {
		enabled            bool
		mustChangePassword bool
		name               string
		password           string
		principalIds       types.Array
		username           string
	}

	if c.Bind(&body) != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to read Body",
		})
		return
	}

	reqBody, err := json.Marshal(body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to marshal Body",
		})
		return
	}

	accessKey := os.Getenv("CATTLE_ACCESS_KEY")
	secretKey := os.Getenv("CATTLE_SECRET_KEY")
	rancherUrl := os.Getenv("CATTLE_URL")

	client := &http.Client{}
	req, _ := http.NewRequest("POST", rancherUrl+"/v3/users", bytes.NewBuffer(reqBody))
	req.Header.Add("Authorization", fmt.Sprintf("%s:%s", accessKey, secretKey)) //base64?
	resp, _ := client.Do(req)

	if resp.Status == "201" {
		fmt.Println("we happy noe.")
	}

}
