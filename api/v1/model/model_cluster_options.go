package model

type ClusterOptions struct {
	Name            string `json:"name"`
	Version         string `json:"version"`
	IngressProvider string `json:"ingress_provider"`
}
