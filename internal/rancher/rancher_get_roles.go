package rancher

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
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

func (r *Rancher) GetRoles() (RoleResponse, error) {
	var roles RoleResponse

	//Do external request
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, err := http.NewRequest("GET", r.Url+"/v3/globalRoles", nil)
	if err != nil {
		return RoleResponse{}, err
	}
	req.Header.Set(
		"Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(r.BearerToken)),
	)
	// Response from the external request
	resp, err := client.Do(req)
	if err != nil {
		return RoleResponse{}, err
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return RoleResponse{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return RoleResponse{}, errors.New(fmt.Sprintf("%s %s", resp.Status, string(data)))
	}
	err = resp.Body.Close()
	if err != nil {
		return RoleResponse{}, err
	}
	err = json.Unmarshal(data, &roles)
	if err != nil {
		return RoleResponse{}, err
	}
	return roles, nil
}
