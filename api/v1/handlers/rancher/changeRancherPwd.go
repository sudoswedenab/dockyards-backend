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
)

func ChangeRancherPWD(user model.User) string {

	RandomPwd := model.NewPassword{NewPassword: String(34)}
	fmt.Println(RandomPwd)

	reqBody, err := json.Marshal(RandomPwd)
	if err != nil {
		fmt.Printf("%s", err)
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
		fmt.Printf("%s", extErr)
	}

	respErr := resp.Body.Close()
	if respErr != nil {
		return ""
	}

	return RandomPwd.NewPassword
}
