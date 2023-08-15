package model

type Options struct {
	Version         []string          `json:"version"`
	SingleNode      bool              `json:"single_node"`
	NodePoolOptions []NodePoolOptions `json:"node_pool_options"`
}
