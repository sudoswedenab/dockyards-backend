package rancher

import (
	"bytes"
	"crypto/tls"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"bitbucket.org/sudosweden/backend/api/v1/model"
	"bitbucket.org/sudosweden/backend/internal"
)

type RancherUserResponse struct {
	Id string `json:"id"`
}

var roleId string

func (r *Rancher) RancherCreateUser(user model.RancherUser) (string, error) {
	reqBody, err := json.Marshal(user)
	if err != nil {
		return "", err
	}

	// Do external request
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, _ := http.NewRequest("POST", r.Url+"/v3/users", bytes.NewBuffer(reqBody))
	req.Header.Set(
		"Authorization", "Basic "+b64.StdEncoding.EncodeToString([]byte(r.BearerToken)),
	)
	// Response from the external request
	resp, extErr := client.Do(req)
	if extErr != nil {
		return "", extErr
	}
	data, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode >= http.StatusBadRequest {
		return "", fmt.Errorf("unexpected status code %d, data: %s", resp.StatusCode, data)
	}

	respErr := resp.Body.Close()
	if respErr != nil {
		return "", respErr
	}
	var rancherUserResponse RancherUserResponse
	json.Unmarshal(data, &rancherUserResponse)
	// fmt.Printf("%T\n", rancherUserResponse.Id)

	roles, err := internal.GetRoles()
	if err != nil {
		log.Println(err.Error())
	}

	for _, value := range roles.Data {
		if value.Name == "dockyard-role" {
			roleId = value.Id
		}
	}

	internal.BindRole(rancherUserResponse.Id, roleId)

	if resp.Status == "201" {
		return "", nil
	}
	return rancherUserResponse.Id, nil
}
