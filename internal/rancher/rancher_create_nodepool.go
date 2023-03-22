package rancher

import (
	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
)

func (r *Rancher) RancherCreateNodePool(id, name string) (managementv3.NodePool, error) {
	nodePool := managementv3.NodePool{
		ClusterID:               id,
		ControlPlane:            true,
		DeleteNotReadyAfterSecs: 0,
		DrainBeforeDelete:       true,
		Etcd:                    true,
		HostnamePrefix:          name + "-node-",
		Name:                    "",
		NamespaceId:             "",
		NodeTaints:              []managementv3.Taint{},
		NodeTemplateID:          "cattle-global-nt:nt-dzqhk",
		Quantity:                3,
		Worker:                  true,
	}

	createdNodePool, err := r.ManagementClient.NodePool.Create(&nodePool)
	if err != nil {
		return managementv3.NodePool{}, err
	}

	return *createdNodePool, nil
}
