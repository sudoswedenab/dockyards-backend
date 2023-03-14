package rancher

import (
	"bitbucket.org/sudosweden/backend/api/v1/model"
	"bytes"
	"crypto/tls"
	b64 "encoding/base64"
	"encoding/json"
	"io"
	"net/http"
)

type ClusterTwos struct {
	Name string `json:"name"`
	Id   string `json:"id"`
}

func (r *Rancher) RancherCreateNodePool(bearerToken, id, name string) (ClusterTwos, error) {
	body := model.NodePoolbody{
		ClusterId:               id,
		ControlPlane:            true,
		DeleteNotReadyAfterSecs: 0,
		DrainBeforeDelete:       true,
		Etcd:                    true,
		HostnamePrefix:          name + "-node-",
		Name:                    "",
		NamespaceId:             "",
		NodeTaints:              make([]string, 0),
		NodeTemplateId:          "cattle-global-nt:nt-5hxd5",
		Quantity:                3,
		Worker:                  true,
	}

	reqBody, err := json.Marshal(body)
	if err != nil {
		return ClusterTwos{}, err
	}

	//Do external request
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, _ := http.NewRequest("POST", r.Url+"/v3/nodepools", bytes.NewBuffer(reqBody))

	//Setting the header
	req.Header = http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {"Basic " + b64.StdEncoding.EncodeToString([]byte(bearerToken))},
		"Accept":        {"application/json"},
		"Origin":        {r.Url},
		"Connection":    {"keep-alive"},
		"TE":            {"trailers"},
	}

	// Response from the external request
	resp, err := client.Do(req)
	if err != nil {
		return ClusterTwos{}, err
	}
	// The Response from the header are we Captureing here

	data, _ := io.ReadAll(resp.Body)

	err = resp.Body.Close()
	if err != nil {
		return ClusterTwos{}, err
	}

	var clusterTwos ClusterTwos
	// Converting the data from ones and zeros and binding it to the struct Responsbody
	err = json.Unmarshal(data, &clusterTwos)
	if err != nil {
		return ClusterTwos{}, err
	}

	return clusterTwos, err
}
