package rancher

import (
	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
)

func (r *Rancher) RancherCreateCluster(dockerRootDir, name, ctrId, ctId string) (managementv3.Cluster, error) {

	clusterTemplate := managementv3.ClusterTemplate{
		Name: "testar",
	}

	createdClusterTemplate, err := r.ManagementClient.ClusterTemplate.Create(&clusterTemplate)
	if err != nil {
		return managementv3.Cluster{}, err
	}

	//STRUCT for Config Rancher
	rancherKubernetesEngineConfig := managementv3.RancherKubernetesEngineConfig{
		Version:         "v1.24.9-rancher1-1",
		AddonJobTimeout: 45,
		Authentication: &managementv3.AuthnConfig{
			Strategy: "x509",
		},
		IgnoreDockerVersion: boolPtr(true),
		Ingress: &managementv3.IngressConfig{
			DefaultIngressClass: boolPtr(true),
			Provider:            "nginx",
		},
		Monitoring: &managementv3.MonitoringConfig{
			Provider: "metrics-server",
			Replicas: int64Ptr(1),
		},
		Network: &managementv3.NetworkConfig{
			Options: map[string]string{
				"flannel_backend_type": "vxlan",
			},
			Plugin: "canal",
		},

		Services: &managementv3.RKEConfigServices{
			Etcd: &managementv3.ETCDService{
				BackupConfig: &managementv3.BackupConfig{
					Enabled:       boolPtr(true),
					IntervalHours: 12,
					Retention:     6,
					Timeout:       300,
				},
				Creation: "12h",
				ExtraArgs: map[string]string{
					"Election - timeout":   "5000",
					"Heartbeat - interval": "500",
				},
				Retention: "72h",
			},
			KubeAPI: &managementv3.KubeAPIService{
				ServiceNodePortRange: "30000 - 32767",
			},
		},

		UpgradeStrategy: &managementv3.NodeUpgradeStrategy{
			MaxUnavailableControlplane: "1",
			MaxUnavailableWorker:       "10%",
			DrainInput: &managementv3.NodeDrainInput{
				GracePeriod:      -1,
				IgnoreDaemonSets: boolPtr(true),
				Timeout:          120,
			},
		},
	}

	clusterConfig := managementv3.ClusterSpecBase{
		RancherKubernetesEngineConfig: &rancherKubernetesEngineConfig,
	}

	clusterTemplateRevision := managementv3.ClusterTemplateRevision{
		Name:              name,
		ClusterTemplateID: createdClusterTemplate.ID,
		ClusterConfig:     &clusterConfig,
	}

	createdClusterTemplateRevision, err := r.ManagementClient.ClusterTemplateRevision.Create(&clusterTemplateRevision)
	if err != nil {
		return managementv3.Cluster{}, err
	}

	cluster := managementv3.Cluster{
		DockerRootDir:             dockerRootDir,
		Name:                      name,
		ClusterTemplateRevisionID: createdClusterTemplateRevision.ID,
		ClusterTemplateID:         createdClusterTemplate.ID,
	}

	createdCluster, err := r.ManagementClient.Cluster.Create(&cluster)
	if err != nil {
		return managementv3.Cluster{}, err
	}

	return *createdCluster, nil
}
