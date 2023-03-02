package rancher

import (
	"bytes"
	"crypto/tls"
	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"bitbucket.org/sudosweden/backend/api/v1/model"
	"bitbucket.org/sudosweden/backend/internal"
)

type RancherResponseToken struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	UserId      string `json:"userId"`
	Bearertoken string `json:"token"`
}

func CreateRancherToken(rancherToken model.RRtoken) (string, string, error) {
	reqBody, err := json.Marshal(rancherToken)
	if err != nil {
		err := errors.New("not valid json,failed to marshal body")
		return "", "", err
	}

	bearerToken := internal.CattleBearerToken
	rancherURL := internal.CattleUrl
	// Do external request
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, _ := http.NewRequest("POST", rancherURL+"/v3-public/localProviders/local?action=login", bytes.NewBuffer(reqBody))
	req.Header.Set(
		"Authorization", "Basic "+b64.StdEncoding.EncodeToString([]byte(bearerToken)),
	)
	// Response from the external request
	resp, extErr := client.Do(req)
	if extErr != nil {
		errormsg := fmt.Sprintf("There was an external error: %s", extErr.Error())
		err := errors.New(errormsg)
		return "", "", err
	}
	data, _ := ioutil.ReadAll(resp.Body)
	respErr := resp.Body.Close()
	if respErr != nil {
		return "", "", respErr
	}
	var valuetok RancherResponseToken
	json.Unmarshal(data, &valuetok)

	return valuetok.Bearertoken, valuetok.UserId, nil
}
