package rancher

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"bitbucket.org/sudosweden/backend/api/v1/model"
)

type rancherResponseToken struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	UserId      string `json:"userId"`
	Bearertoken string `json:"token"`
}

func (r *Rancher) createRancherToken(rancherToken model.RRtoken) (string, string, error) {
	reqBody, err := json.Marshal(rancherToken)
	if err != nil {
		err := errors.New("not valid json,failed to marshal body")
		return "", "", err
	}

	// Do external request
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, _ := http.NewRequest("POST", r.url+"/v3-public/localProviders/local?action=login", bytes.NewBuffer(reqBody))
	req.Header.Set(
		"Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(r.bearerToken)),
	)
	// Response from the external request
	resp, extErr := client.Do(req)
	if extErr != nil {
		errormsg := fmt.Sprintf("There was an external error: %s", extErr.Error())
		err := errors.New(errormsg)
		return "", "", err
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	r.Logger.Debug("response status code from login", "status-code", resp.StatusCode, "data", data)
	if resp.StatusCode != http.StatusCreated {
		return "", "", fmt.Errorf("unexpected status code %d when doing user login", resp.StatusCode)
	}

	respErr := resp.Body.Close()
	if respErr != nil {
		return "", "", respErr
	}
	var valuetok rancherResponseToken
	json.Unmarshal(data, &valuetok)

	return valuetok.Bearertoken, valuetok.UserId, nil
}
