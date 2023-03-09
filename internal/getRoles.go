package internal

import (
	"crypto/tls"
	b64 "encoding/base64"
	"encoding/json"
	"io"
	"net/http"
)

type Data struct {
	Name string `json:"name"`
	Id   string `json:"id"`
}

type RoleResponse struct {
	Data []Data `json:"data"`
}

func GetRoles() (RoleResponse, error) {

	bearerToken := CattleBearerToken
	rancherURL := CattleUrl

	//Do external request
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, _ := http.NewRequest("GET", rancherURL+"/v3/globalRoles", nil)
	req.Header.Set(
		"Authorization", "Basic "+b64.StdEncoding.EncodeToString([]byte(bearerToken)),
	)
	// Response from the external request
	resp, err := client.Do(req)
	if err != nil {
		return RoleResponse{}, err
	}
	data, _ := io.ReadAll(resp.Body)

	err = resp.Body.Close()
	if err != nil {
		return RoleResponse{}, err
	}

	var roles RoleResponse
	json.Unmarshal(data, &roles)

	return roles, err
}
