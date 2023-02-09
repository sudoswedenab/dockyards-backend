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

type NodePoolbody struct {
	ControlPlane            bool     `json:"controlPlane"`
	DeleteNotReadyAfterSecs int      `json:"deleteNotReadyAfterSecs"`
	DrainBeforeDelete       bool     `json:"drainBeforeDelete"`
	Etcd                    bool     `json:"etcd"`
	Quantity                int      `json:"quantity"`
	Worker                  bool     `json:"worker"`
	NamespaceId             string   `json:"namespaceId"`
	Name                    string   `json:"name"`
	Type                    string   `json:"nodePool"`
	ClusterId               string   `json:"clusterId"`
	HostnamePrefix          string   `json:"hostnamePrefix"`
	NodeTemplateId          string   `json:"nodeTemplateId"`
	NodeTaints              []string `json:"nodeTaints"`
}

// clusterId": "c-tvrfj",
// "controlPlane": true,
// "deleteNotReadyAfterSecs": 0,
// "drainBeforeDelete": true,
// "etcd": true,
// "hostnamePrefix": "kappastwo-node-",
// "name": "",
// "namespaceId": "",
// "nodeTaints": [ ],
// "nodeTemplateId": "cattle-global-nt:nt-zd2tl",
// "quantity": 3,
// "worker": true
