package rancher

import (
	"bitbucket.org/sudosweden/backend/api/v1/model"
	"bitbucket.org/sudosweden/backend/internal"
	"bytes"
	"crypto/tls"
	b64 "encoding/base64"
	"encoding/json"
	"io"
	"net/http"
)

type NodePool struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

func (r *Rancher) RancherCreateCluster(clusterData model.ClusterData, bearerToken string) (NodePool, error) {

	reqBody, err := json.Marshal(clusterData)
	if err != nil {
		return NodePool{}, err
	}

	//Do external request
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, _ := http.NewRequest("POST", r.Url+"/v3/clusters", bytes.NewBuffer(reqBody))

	req.Header = http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {"Basic " + b64.StdEncoding.EncodeToString([]byte(bearerToken))},
		"Accept":        {"application/json"},
		"Origin":        {internal.CattleUrl},
		"Connection":    {"keep-alive"},
		// "Referer":       {"https://ss-di-rancher.sudobash.io/g/clusters/add/launch/openstack?clusterTemplateRevision=cattle-global-data%3Actr-7xnpl"},
		"TE": {"trailers"},
	}

	// Response from the external request
	resp, err := client.Do(req)
	if err != nil {
		return NodePool{}, err
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return NodePool{}, err
	}

	err = resp.Body.Close()
	if err != nil {
		return NodePool{}, err
	}

	var responseBody NodePool
	err = json.Unmarshal(data, &responseBody)
	if err != nil {
		return NodePool{}, err
	}

	return responseBody, nil
}
