package internal

import (
	"bytes"
	"crypto/tls"
	b64 "encoding/base64"
	"encoding/json"
	"net/http"
)

type RoleBinding struct {
	GlobalRoleId string `json:"globalRoleId"`
	Type         string `json:"type"`
	UserId       string `json:"userId"`
}

func BindRole(userid string, roleid string) error {

	bearerToken := CattleBearerToken
	rancherURL := CattleUrl

	body := RoleBinding{
		GlobalRoleId: roleid,
		Type:         "globalRoleBinding",
		UserId:       userid,
	}

	reqBody, _ := json.Marshal(body)

	//Do external request
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, _ := http.NewRequest("POST", rancherURL+"/v3/globalrolebindings", bytes.NewBuffer(reqBody))
	req.Header.Set(
		"Authorization", "Basic "+b64.StdEncoding.EncodeToString([]byte(bearerToken)),
	)
	// Response from the external request
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	err = resp.Body.Close()
	if err != nil {
		return err
	}

	return err
}
