package rancher

import managementv3 "github.com/rancher/rancher/pkg/client/generated/management/v3"

const (
	TaintNodeRoleLoadBalancer = "node-role.dockyards.io/load-balancer"
	LabelNodeRoleLoadBalancer = "node-role.dockyards.io/load-balancer"
)

type CustomNodeTemplate struct {
	managementv3.NodeTemplate
	OpenstackConfig *openstackConfig `json:"openstackConfig,omitempty" yaml:"openstackConfig,omitempty"`
}

type openstackConfig struct {
	ActiveTimeout               string `json:"activeTimeout,omitempty" yaml:"activeTimeout,omitempty"`
	AuthURL                     string `json:"authUrl,omitempty" yaml:"authUrl,omitempty"`
	AvailabilityZone            string `json:"availabilityZone,omitempty" yaml:"availabilityZone,omitempty"`
	CaCert                      string `json:"cacert,omitempty" yaml:"cacert,omitempty"`
	ConfigDrive                 bool   `json:"configDrive,omitempty" yaml:"configDrive,omitempty"`
	DomainID                    string `json:"domainId,omitempty" yaml:"domainId,omitempty"`
	DomainName                  string `json:"domainName,omitempty" yaml:"domainName,omitempty"`
	EndpointType                string `json:"endpointType,omitempty" yaml:"endpointType,omitempty"`
	FlavorID                    string `json:"flavorId,omitempty" yaml:"flavorId,omitempty"`
	FlavorName                  string `json:"flavorName,omitempty" yaml:"flavorName,omitempty"`
	FloatingIPPool              string `json:"floatingipPool,omitempty" yaml:"floatingipPool,omitempty"`
	ImageID                     string `json:"imageId,omitempty" yaml:"imageId,omitempty"`
	ImageName                   string `json:"imageName,omitempty" yaml:"imageName,omitempty"`
	Insecure                    bool   `json:"insecure,omitempty" yaml:"insecure,omitempty"`
	IPVersion                   string `json:"ipVersion,omitempty" yaml:"ipVersion,omitempty"`
	KeypairName                 string `json:"keypairName,omitempty" yaml:"keypairName,omitempty"`
	NetID                       string `json:"netId,omitempty" yaml:"netId,omitempty"`
	NetName                     string `json:"netName,omitempty" yaml:"netName,omitempty"`
	NovaNetwork                 bool   `json:"novaNetwork,omitempty" yaml:"novaNetwork,omitempty"`
	Password                    string `json:"password,omitempty" yaml:"password,omitempty"`
	PrivateKeyFile              string `json:"privateKeyFile,omitempty" yaml:"privateKeyFile,omitempty"`
	Region                      string `json:"region,omitempty" yaml:"region,omitempty"`
	SecGroups                   string `json:"secGroups,omitempty" yaml:"secGroups,omitempty"`
	SSHPort                     string `json:"sshPort,omitempty" yaml:"sshPort,omitempty"`
	SSHUser                     string `json:"sshUser,omitempty" yaml:"sshUser,omitempty"`
	TenantID                    string `json:"tenantId,omitempty" yaml:"tenantId,omitempty"`
	TenantName                  string `json:"tenantName,omitempty" yaml:"tenantName,omitempty"`
	UserDataFile                string `json:"userDataFile,omitempty" yaml:"userDataFile,omitempty"`
	Username                    string `json:"username,omitempty" yaml:"username,omitempty"`
	ApplicationCredentialID     string `json:"applicationCredentialId,omitempty" yaml:"applicationCredentialId,omitempty"`
	ApplicationCredentialName   string `json:"applicationCredentialName,omitempty" yaml:"applicationCredentialName,omitempty"`
	ApplicationCredentialSecret string `json:"applicationCredentialSecret,omitempty" yaml:"applicationCredentialSecret,omitempty"`
	BootFromVolume              bool   `json:"bootFromVolume,omitempty" yaml:"bootFromVolume,omitempty"`
	VolumeType                  string `json:"volumeType,omitempty" yaml:"volumeType,omitempty"`
	VolumeSize                  string `json:"volumeSize,omitempty" yaml:"volumeSize,omitempty"`
	VolumeID                    string `json:"volumeId,omitempty" yaml:"volumeId,omitempty"`
	VolumeName                  string `json:"volumeName,omitempty" yaml:"volumeName,omitempty"`
	VolumeDevicePath            string `json:"volumeDevicePath,omitempty" yaml:"volumeDevicePath,omitempty"`
}
