package model

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
