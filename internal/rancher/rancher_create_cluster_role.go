package rancher

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
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

func (r *Rancher) CreateClusterRole() error {
	init_roles, err := r.GetRoles()
	if err != nil {
		return err
	}

	create := true
	for _, value := range init_roles.Data {
		fmt.Printf("checking role '%s'\n", value.Name)
		if value.Name == "dockyard-role" {
			fmt.Println(time.Now().Format(time.RFC822), " User role verified")
			create = false
		}
	}
	fmt.Printf("role 'dockyard-role' needs to be created: %t\n", create)
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
		fmt.Printf("role '%s' prepared with %d rules\n", body.Name, len(body.Rules))

		reqBody, err := json.Marshal(body)
		if err != nil {
			return err
		}
		fmt.Printf("reqBody json: %s\n", string(reqBody))

		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client := &http.Client{Transport: tr}

		req, err := http.NewRequest(http.MethodPost, r.Url+"/v3/globalroles", bytes.NewBuffer(reqBody))
		if err != nil {
			return err
		}
		fmt.Printf("req: %#v\n", req)

		//Setting the header
		req.Header = http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Basic " + base64.StdEncoding.EncodeToString([]byte(r.BearerToken))},
			"Accept":        {"application/json"},
			"Origin":        {r.Url},
			"Connection":    {"keep-alive"},
			"TE":            {"trailers"},
		}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		if resp.StatusCode != http.StatusCreated {
			return fmt.Errorf("unxepected status code %d creating global role, data: %s", resp.StatusCode, body)
		}

		err = resp.Body.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
