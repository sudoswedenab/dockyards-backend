package internal

import (
	"bytes"
	"crypto/tls"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

func CreateClusterRole() {

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
	resp, extErr := client.Do(req)
	if extErr != nil {
		fmt.Printf("There was an external error: %s", extErr.Error())
		return
	}
	data, _ := ioutil.ReadAll(resp.Body)

	respErr := resp.Body.Close()
	if respErr != nil {
		return
	}

	type Data struct {
		Name string `json:"name"`
	}

	type RoleResponse struct {
		Data []Data `json:"data"`
	}

	type Rule struct {
		ApiGroups []string `json:"apiGroups"`
		Resources []string `json:"resources"`
		Type      string   `json:"type"`
		Verbs     []string `json:"verbs"`
	}

	type Role struct {
		Description    string `json:"description"`
		Name           string `json:"name"`
		NewUserDefault bool   `json:"newUserDefault"`
		Rules          []Rule `json:"rules"`
	}

	var roles RoleResponse
	json.Unmarshal(data, &roles)

	create := true
	for _, value := range roles.Data {
		if value.Name == "dockyard-role" {
			fmt.Println(time.Now().Format(time.RFC822), " User role verified")
			*&create = false
		}
	}
	if create == true {
		body := Role{
			Description:    "",
			Name:           "dockyard-role",
			NewUserDefault: true,
			Rules: []Rule{
				{
					ApiGroups: []string{"management.cattle.io"},
					Resources: []string{"nodetemplates"},
					Type:      "/v3/schemas/policyRule",
					Verbs:     []string{"get"},
				},
				{
					ApiGroups: []string{"management.cattle.io"},
					Resources: []string{"clustertemplaterevisions"},
					Type:      "/v3/schemas/policyRule",
					Verbs:     []string{"get"},
				},
				{
					ApiGroups: []string{"management.cattle.io"},
					Resources: []string{"nodepools"},
					Type:      "/v3/schemas/policyRule",
					Verbs:     []string{"update"},
				},
			},
		}

		reqBody, _ := json.Marshal(body)

		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client := &http.Client{Transport: tr}

		req, _ := http.NewRequest("POST", rancherURL+"/v3/globalroles", bytes.NewBuffer(reqBody))

		//Setting the header
		req.Header = http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Basic " + b64.StdEncoding.EncodeToString([]byte(bearerToken))},
			"Accept":        {"application/json"},
			"Origin":        {CattleUrl},
			"Connection":    {"keep-alive"},
			"TE":            {"trailers"},
		}
		_, err := client.Do(req)
		if err != nil {

		}
	}
}
