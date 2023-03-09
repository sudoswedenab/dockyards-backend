package internal

import (
	"bytes"
	"crypto/tls"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

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

func CreateClusterRole() {

	bearerToken := CattleBearerToken
	rancherURL := CattleUrl

	roles, err := GetRoles()
	if err != nil {
		log.Println(err.Error())
	}

	create := true
	for _, value := range roles.Data {
		if value.Name == "dockyard-role" {
			fmt.Println(time.Now().Format(time.RFC822), " User role verified")
			create = false
		}
	}
	if create {
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
				{
					ApiGroups: []string{"management.cattle.io"},
					Resources: []string{"clusters"},
					Type:      "/v3/schemas/policyRule",
					Verbs:     []string{"create", "list"},
				},
				{
					ApiGroups: []string{"provisioning.cattle.io"},
					Resources: []string{"clusters"},
					Type:      "/v3/schemas/policyRule",
					Verbs:     []string{"create"},
				},
				{
					ApiGroups: []string{"management.cattle.io"},
					Resources: []string{"kontainerdrivers"},
					Type:      "/v3/schemas/policyRule",
					Verbs:     []string{"get", "list", "watch"},
				},
			},
		}

		reqBody, _ := json.Marshal(body)

		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client := &http.Client{Transport: tr}

		req, err := http.NewRequest("POST", rancherURL+"/v3/globalroles", bytes.NewBuffer(reqBody))
		if err != nil {
			log.Fatal(err)
		}

		//Setting the header
		req.Header = http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Basic " + b64.StdEncoding.EncodeToString([]byte(bearerToken))},
			"Accept":        {"application/json"},
			"Origin":        {CattleUrl},
			"Connection":    {"keep-alive"},
			"TE":            {"trailers"},
		}
		_, err = client.Do(req)
		if err != nil {
			log.Fatal(err)
		}
	}
}
