package rancher

import (
	"errors"

	"bitbucket.org/sudosweden/backend/api/v1/model"
	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
)

func (r *Rancher) clusterOptionsToRKEConfig(clusterOptions model.ClusterOptions) (*managementv3.RancherKubernetesEngineConfig, error) {

	version := "v1.24.9-rancher1-1"
	ingressProvider := "nginx"

	if clusterOptions.Version != "" && clusterOptions.Version != "v1.24.9-rancher1-1" {
		return nil, errors.New("unsupported version")
	}

	if clusterOptions.IngressProvider != "" && clusterOptions.IngressProvider != "nginx" {
		return nil, errors.New("unsupported ingress provider")
	}

	//STRUCT for Config Rancher
	rancherKubernetesEngineConfig := managementv3.RancherKubernetesEngineConfig{
		Version:         version,
		AddonJobTimeout: 45,
		Authentication: &managementv3.AuthnConfig{
			Strategy: "x509",
		},
		IgnoreDockerVersion: boolPtr(true),
		Ingress: &managementv3.IngressConfig{
			DefaultIngressClass: boolPtr(true),
			Provider:            ingressProvider,
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
					"Election-timeout":   "5000",
					"Heartbeat-interval": "500",
				},
				Retention: "72h",
			},
			KubeAPI: &managementv3.KubeAPIService{
				ServiceNodePortRange: "30000-32767",
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
	return &rancherKubernetesEngineConfig, nil
}
