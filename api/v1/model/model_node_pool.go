package model

type NodePool struct {
	Name                       string `json:"name"`
	ControlPlane               bool   `json:"control_plane"`
	Etcd                       bool   `json:"etcd"`
	LoadBalancer               bool   `json:"load_balancer"`
	Quantity                   int    `json:"quantity"`
	ControlPlaneComponentsOnly bool   `json:"control_plane_components_only"`
	CPUCount                   int    `json:"cpu_count"`
	RAMSizeMB                  int    `json:"ram_size_mb"`
	DiskSizeGB                 int    `json:"disk_size_gb"`
}
