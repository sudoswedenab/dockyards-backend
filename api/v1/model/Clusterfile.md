

type NewClusterorius struct {
	AgentEnvVars                        []AgentEnvVars `json:"agentEnvVars"`
	AksConfig                           *string        `json:"aksConfig"`
	AmazonElasticContainerServiceConfig *string        `json:"amazonElasticContainerServiceConfig"`
	Answers                             *string        `json:"answers"`
	AzureKubernetesServiceConfig        *string        `json:"azureKubernetesServiceConfig"`
	ClusterTemplateRevisionId           string         `json:"clusterTemplateRevisionId"`
	DefaultClusterRoleForProjectMembers string         `json:"defaultClusterRoleForProjectMembers"`
	DefaultPodSecurityPolicyTemplateId  string         `json:"defaultPodSecurityPolicyTemplateId"`
	DockerRootDir                       string         `json:"dockerRootDir"`
	EksConfig                           *string        `json:"eksConfig"`
	EnableClusterAlerting               bool           `json:"enableClusterAlerting"`
	EnableClusterMonitoring             bool           `json:"enableClusterMonitoring"`
	GkeConfig                           *string        `json:"gkeConfig"`
	GoogleKubernetesEngineConfig        *string        `json:"googleKubernetesEngineConfig"`
	K3sConfig                           *string        `json:"k3sConfig"`
	LocalClusterAuthEndpoint            *string        `json:"localClusterAuthEndpoint"`
	Name                                string         `json:"name"`
	RancherKubernetesEngineConfig       *string        `json:"rancherKubernetesEngineConfig"`
	Rke2Config                          *string        `json:"rke2Config"`
	ScheduledClusterScan                *string        `json:"scheduledClusterScan"`
	WindowsPreferedCluster              bool           `json:"windowsPreferedCluster"`
}

type AgentEnvVars struct {
}

// {
// "agentEnvVars": [ ],
// "aksConfig": null,
// "amazonElasticContainerServiceConfig": null,
// "answers": null,
// "azureKubernetesServiceConfig": null,
// "clusterTemplateRevisionId": "cattle-global-data:ctr-7xnpl",
// "defaultClusterRoleForProjectMembers": "",
// "defaultPodSecurityPolicyTemplateId": "",
// "dockerRootDir": "/var/lib/docker",
// "eksConfig": null,
// "enableClusterAlerting": false,
// "enableClusterMonitoring": false,
// "gkeConfig": null,
// "googleKubernetesEngineConfig": null,
// "k3sConfig": null,
// "localClusterAuthEndpoint": null,
// "name": "adamcluster",
// "rancherKubernetesEngineConfig": null,
// "rke2Config": null,
// "scheduledClusterScan": null,
// "windowsPreferedCluster": false
// }
