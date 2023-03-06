package model

type KubeConfig struct {
	Data []KubeConf
}
type KubeConf struct {
	BaseType string `json:"baseType"`
	Config   string `json:"config"`
	Type     string `json:"type"`
}
