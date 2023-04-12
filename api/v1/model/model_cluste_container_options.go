package model

type ContainerOptions struct {
	Options []Options `json:"options"`
}

type Options struct {
	Name            string            `json:"name"`
	Version         []string          `json:"version"`
	SingleNode      bool              `json:"single_node"`
	NodePoolOptions []NodePoolOptions `json:"node_pool_options"`
}
