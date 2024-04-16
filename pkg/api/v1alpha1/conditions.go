package v1alpha1

const (
	ReadyCondition            string = "Ready"
	VerifiedCondition         string = "Verified"
	ProvisionedCondition      string = "Provisioned"
	CloudConfigReadyCondition string = "CloudConfigReady"
)

const (
	CloudProjectAssignedReason string = "CloudProjectAssigned"
	UserVerifiedReason         string = "UserVerified"
	ClusterReadyReason         string = "ClusterReady"
	NodeReadyReason            string = "NodeReady"
	DeploymentReadyReason      string = "DeploymentReady"
	NodeProvisionedReason      string = "NodeProvisioned"
	ProvisioningFailedReason   string = "ProvisioningFailed"
	NodePoolProvisionedReason  string = "NodePoolProvisioned"
	ClusterProvisionedReason   string = "ClusterProvisioned"
)
