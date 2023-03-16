package rancher

import (
	"bytes"
	"crypto/tls"
	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"bitbucket.org/sudosweden/backend/api/v1/model"
)

func (r *Rancher) changeRancherPWD(user model.User) (string, error) {
	RandomPwd := model.NewPassword{NewPassword: String(34)}

	reqBody, err := json.Marshal(RandomPwd)
	if err != nil {
		err := errors.New("not valid json,failed to marshal body")
		return "", err
	}

	// Do external request
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, _ := http.NewRequest("POST", r.Url+"/v3/users/"+user.RancherID+"?action=setpassword", bytes.NewBuffer(reqBody))
	req.Header.Set(
		"Authorization", "Basic "+b64.StdEncoding.EncodeToString([]byte(r.BearerToken)),
	)
	// Response from the external request
	resp, extErr := client.Do(req)
	if extErr != nil {
		errormsg := fmt.Sprintf("There was an external error: %s", extErr.Error())
		err := errors.New(errormsg)
		fmt.Println(err)
		return "", err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	respErr := resp.Body.Close()
	if respErr != nil {
		return "", respErr
	}

	fmt.Printf("status code from set password action: %d, body: %s\n", resp.StatusCode, body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code %d when setting rancher password", resp.StatusCode)
	}
	// time.Sleep(10 * time.Second)
	return RandomPwd.NewPassword, nil
}
