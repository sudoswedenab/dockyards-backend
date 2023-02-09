package model

// Used in Create cluster, line 49
type ClusterData struct {
	DockerRootDir             string  `json:"dockerRootDir"`
	Type                      string  `json:"type"`
	Name                      string  `json:"name"`
	ClusterTemplateRevisionId string  `json:"clusterTemplateRevisionId"`
	ClusterTemplateId         string  `json:"clusterTemplateId"`
	Rke2Config                *string `json:"rke2Config,omitempty"`
}
