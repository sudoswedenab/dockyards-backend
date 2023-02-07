package model

// {"controlPlane": "true",
// "deleteNotReadyAfterSecs": 0,
// "drainBeforeDelete": "false",
// "etcd": "true",
// "quantity": 3,
// "worker": "true",
//  "type": "nodePool",
//  "clusterId": ID ,
//  "hostnamePrefix": NAME + "-node-",
//  "nodeTemplateId": "cattle-global-nt:nt-zd2tl"}

type ClusterTwo struct {
	ControlPlane            bool   `json:"controlPlane"`
	DeleteNotReadyAfterSecs int    `json:"deleteNotReadyAfterSecs"`
	DrainBeforeDelete       bool   `json:"drainBeforeDelete"`
	Etcd                    bool   `json:"etcd"`
	Quantity                int    `json:"quantity"`
	Worker                  bool   `json:"worker"`
	Type                    string `json:"nodePool"`
	ClusterId               string `json:"ID"`
	HostnamePrefix          string `json:"NAME" "+""-node-"`
	NodeTemplateId          string `json:"nodeTemplateId"`
}
