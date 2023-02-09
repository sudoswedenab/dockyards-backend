package rancher

import (
	"Backend/api/v1/model"
	"bytes"
	"crypto/tls"
	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
)

func ChangeRancherPWD(user model.User) (string, error) {

	RandomPwd := model.NewPassword{NewPassword: String(34)}

	reqBody, err := json.Marshal(RandomPwd)
	if err != nil {
		err := errors.New("not valid json,failed to marshal body")
		return "", err
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
		errormsg := fmt.Sprintf("There was an external error: %s", extErr.Error())
		err := errors.New(errormsg)
		fmt.Println(err)
		return "", err
	}

	respErr := resp.Body.Close()
	if respErr != nil {
		return "", respErr
	}
	// time.Sleep(10 * time.Second)
	return RandomPwd.NewPassword, nil
}
