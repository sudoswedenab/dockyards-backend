package rancher

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

type RoleBinding struct {
	GlobalRoleId string `json:"globalRoleId"`
	Type         string `json:"type"`
	UserId       string `json:"userId"`
}

func (r *Rancher) BindRole(userid string, roleid string) error {
	body := RoleBinding{
		GlobalRoleId: roleid,
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
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		err = resp.Body.Close()
		if err != nil {
			return err
		}
		return errors.New(fmt.Sprintf(time.Now().Format(time.RFC822), " %d %s", resp.StatusCode, string(body)))
	}

	err = resp.Body.Close()
	if err != nil {
		return err
	}

	return err
}
