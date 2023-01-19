package rancher

import (
	"Backend/api/v1/model"
	"bytes"
	"crypto/tls"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func ChangeRancherPWD(c *gin.Context, user model.User) string {

	RandomPwd := model.NewPassword{NewPassword: String(34)}
	fmt.Println(RandomPwd)

	reqBody, err := json.Marshal(RandomPwd)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Not valid JSON! Failed to marshal Body",
		})
		return ""
	}

	bearerToken := os.Getenv("CATTLE_BEARER_TOKEN")
	rancherURL := os.Getenv("CATTLE_URL")
	// Do external request
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, _ := http.NewRequest("POST", rancherURL+"/v3/users/"+user.RancherID+"?action=setpassword", bytes.NewBuffer(reqBody))
	req.Header.Set(
		"Authorization", "Basic "+b64.StdEncoding.EncodeToString([]byte(bearerToken)),
	)
	// Response from the external request
	resp, extErr := client.Do(req)
	if extErr != nil {
		c.String(http.StatusBadGateway, fmt.Sprintf("There was an external error: %s", extErr.Error()))
		return ""
	}

	respErr := resp.Body.Close()
	if respErr != nil {
		return ""
	}
	// time.Sleep(10 * time.Second)
	return RandomPwd.NewPassword
}
