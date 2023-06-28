package model

type NodePoolOptions struct {
	ControlPlane               bool   `json:"control_plane"`
	Etcd                       bool   `json:"etcd"`
	ControlPlaneComponentsOnly bool   `json:"control_plane_components_only"`
	Quantity                   int    `json:"quantity"`
	Name                       string `json:"name"`
	CPUCount                   int    `json:"cpu_count"`
	RAMSize                    int    `json:"ram_size"`
	DiskSize                   int    `json:"disk_sice"`
	LoadBalancer               bool   `json:"load_balancer"`
}
