package v1alpha1

const (
	ReadyCondition       string = "Ready"
	VerifiedCondition    string = "Verified"
	ProvisionedCondition string = "Provisioned"
)

const (
	CloudProjectAssignedReason string = "CloudProjectAssigned"
	UserVerifiedReason         string = "UserVerified"
	ClusterReadyReason         string = "ClusterReady"
	NodePoolReadyReason        string = "NodePoolReady"
	NodeReadyReason            string = "NodeReady"
	DeploymentReadyReason      string = "DeploymentReady"
	NodeProvisionedReason      string = "NodeProvisioned"
)
