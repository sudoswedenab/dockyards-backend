package model

type KubeConf struct {
	BaseType string `yaml:"baseType"`
	Config   string `yaml:"config"`
	Type     string `yaml:"type"`
}
