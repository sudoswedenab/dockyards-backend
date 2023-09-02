package rancher

import (
	"errors"
	"slices"
	"strings"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/util"
	managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"
)

func (r *rancher) clusterOptionsToRKEConfig(clusterOptions *v1.ClusterOptions) (*managementv3.RancherKubernetesEngineConfig, error) {
	supportedVersions, err := r.GetSupportedVersions()
	if err != nil {
		return nil, err
	}

	version := supportedVersions[0]
	if clusterOptions.Version != nil {
		versionSupported := false
		for _, supportedVersion := range supportedVersions {
			if *clusterOptions.Version == supportedVersion {
				versionSupported = true
				break
			}
		}

		if !versionSupported {
			return nil, errors.New("unsupported version")
		}
	}

	rancherKubernetesEngineConfig := managementv3.RancherKubernetesEngineConfig{
		Version:         version,
		AddonJobTimeout: 45,
		Authentication: &managementv3.AuthnConfig{
			Strategy: "x509",
		},
		IgnoreDockerVersion: util.Ptr(true),
		Ingress: &managementv3.IngressConfig{
			Provider: "none",
		},
		Monitoring: &managementv3.MonitoringConfig{
			Provider: "metrics-server",
			Replicas: util.Ptr(int64(1)),
		},
		Network: &managementv3.NetworkConfig{
			Options: map[string]string{
				"flannel_backend_type": "vxlan",
			},
			Plugin: "flannel",
		},
		Services: &managementv3.RKEConfigServices{
			Etcd: &managementv3.ETCDService{
				BackupConfig: &managementv3.BackupConfig{
					Enabled:       util.Ptr(true),
					IntervalHours: 12,
					Retention:     6,
					Timeout:       300,
				},
				Creation: "12h",
				ExtraArgs: map[string]string{
					"election-timeout":   "5000",
					"heartbeat-interval": "500",
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
				IgnoreDaemonSets: util.Ptr(true),
				Timeout:          120,
			},
		},
	}

	return &rancherKubernetesEngineConfig, nil
}

func (r *rancher) clusterOptionsToNodeTemplate(clusterOptions *v1.ClusterOptions, config *openstackConfig) (*CustomNodeTemplate, error) {
	customNodeTemplate := CustomNodeTemplate{
		NodeTemplate: managementv3.NodeTemplate{
			Name: clusterOptions.Name,
		},
		OpenstackConfig: config,
	}

	return &customNodeTemplate, nil
}

func (r *rancher) GetSupportedVersions() ([]string, error) {
	setting, err := r.managementClient.Setting.ByID("k8s-versions-current")
	if err != nil {
		return []string{}, err
	}

	versions := strings.Split(setting.Value, ",")

	slices.Sort(versions)
	slices.Reverse(versions)

	return versions, nil
}
