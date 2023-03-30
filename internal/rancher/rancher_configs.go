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

func (r *Rancher) clusterOptionsToNodeTemplate(clusterOptions model.ClusterOptions) (*CustomNodeTemplate, error) {
	customNodeTemplate := CustomNodeTemplate{
		NodeTemplate: managementv3.NodeTemplate{
			Name: clusterOptions.Name,
		},
		OpenstackConfig: &openstackConfig{
			DomainID:    "06aa939bcd734f7aa85bef28e2412ec5",
			AuthURL:     "https://v2.dashboard.sto1.safedc.net:5000/v3/",
			FlavorName:  "l2.c2r4.100",
			ImageID:     "e18e0951-d077-44f7-8842-c03cdc126023",
			IPVersion:   "4",
			KeypairName: "rancher",
			NetID:       "21dfbb3d-a948-449b-b727-5fdda2026b45",
			SecGroups:   "a65b6c10-e350-4f8f-b931-eb3a73522fa9",
			SSHPort:     "22",
			SSHUser:     "ubuntu",
			// PrivateKeyFile: "",
		},
	}

	return &customNodeTemplate, nil
}

func (r *Rancher) GetSupportedVersions() []string {

	return []string{"v1.24.9-rancher1-1"}

}
