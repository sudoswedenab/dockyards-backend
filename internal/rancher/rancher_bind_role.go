package rancher

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
)

type RoleBinding struct {
	GlobalRoleId string `json:"globalRoleId"`
	Type         string `json:"type"`
	UserId       string `json:"userId"`
}

func (r *Rancher) BindRole(userid string, roles RoleResponse) error {
	var roleId string
	var roleName string

	for _, value := range roles.Data {
		if value.Name == "dockyard-role" {
			roleId = value.Id
			roleName = value.Name
		}
	}
	if roleName != "dockyard-role" {
		return errors.New("no role named 'dockyard-role' found")
	}

	body := RoleBinding{
		GlobalRoleId: roleId,
		Type:         "globalRoleBinding",
		UserId:       userid,
	}

	reqBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	//Do external request
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, _ := http.NewRequest("POST", r.Url+"/v3/globalrolebindings", bytes.NewBuffer(reqBody))
	req.Header.Set(
		"Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(r.BearerToken)),
	)
	// Response from the external request
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusCreated {
		if resp.StatusCode == http.StatusInternalServerError {
			return errors.New("unexpected status code 500, data: test")
		}
	}

	err = resp.Body.Close()
	if err != nil {
		return err
	}

	return nil
}
