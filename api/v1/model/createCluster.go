package model

type ClusterData struct {
	DockerRootDir             string `json:"dockerRootDir"`
	Type                      string `json:"type"`
	Name                      string `json:"name"`
	ClusterTemplateRevisionId string `json:"clusterTemplateRevisionId"`
	ClusterTemplateId         string `json:"clusterTemplateId"`
	// ClusterSecrets            []clusterSecrets `json:"clusterSecrets"`
}

// type clusterSecrets struct {
// 	Type    string    `json:"type"`
// 	Answers []answers `json:"answers"`
// }

// type answers struct {
// 	Values map[string]string `json:"values"`
// }
