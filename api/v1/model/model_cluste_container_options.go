package model

type ContainerOptions struct {
	Options []Options `json:"options"`
}

type Options struct {
	Name            string            `json:"name"`
	KubeVersion     []string          `json:"kube_version"`
	SingleNode      bool              `json:"single_node"`
	NodePoolOptions []NodePoolOptions `json:"node_pool_options"`
}

type NodePoolOptions struct {
	ControlPlane              bool   `json:"control_plane"`
	Etcd                      bool   `json:"etcd"`
	ControlPlaneComponetsOnly bool   `json:"control_plane_componets_only"`
	Quantity                  int    `json:"quantity"`
	Name                      string `json:"name"`
	CPUCount                  int    `json:"cpu_count"`
	RAMSize                   int    `json:"ram_size"`
	DiskSize                  int    `json:"disk_sice"`
}
