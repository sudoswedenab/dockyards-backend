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

// Line 52 in createNodePool.go
type NodePoolbody struct {
	ClusterId               string   `json:"clusterId"`
	ControlPlane            bool     `json:"controlPlane"`
	DeleteNotReadyAfterSecs int      `json:"deleteNotReadyAfterSecs"`
	DrainBeforeDelete       bool     `json:"drainBeforeDelete"`
	Etcd                    bool     `json:"etcd"`
	HostnamePrefix          string   `json:"hostnamePrefix"`
	Name                    string   `json:"name"`
	NamespaceId             string   `json:"namespaceId"`
	NodeTaints              []string `json:"nodeTaints"`
	NodeTemplateId          string   `json:"nodeTemplateId"`
	Quantity                int      `json:"quantity"`
	Worker                  bool     `json:"worker"`
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
