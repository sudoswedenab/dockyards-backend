package model

type ClusterOptions struct {
	Name            string            `json:"name"`
	Version         string            `json:"version"`
	IngressProvider string            `json:"ingress_provider"`
	NodePoolOptions []NodePoolOptions `json:"node_pool_options"`
	SingleNode      bool              `json:"single_node"`
}
