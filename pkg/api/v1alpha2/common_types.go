package v1alpha2

const (
	// Deprecated: use LabelOrganizationName
	OrganizationNameLabel = "dockyards.io/organization-name"

	// Deprecated: use LabelClusterName
	ClusterNameLabel = "dockyards.io/cluster-name"

	// Deprecated: use LabelNodePoolName
	NodePoolNameLabel = "dockyards.io/node-pool-name"

	// Deprecated: use LabelNodeName
	NodeNameLabel = "dockyards.io/node-name"

	// Deprecated: use LabelDeploymentName
	DeploymentNameLabel = "dockyards.io/deploymennt-name"
)

const (
	LabelOrganizationName       = "dockyards.io/organization-name"
	LabelClusterName            = "dockyards.io/cluster-name"
	LabelNodePoolName           = "dockyards.io/node-pool-name"
	LabelNodeName               = "dockyards.io/node-name"
	LabelDeploymentName         = "dockyards.io/deployment-name"
	LabelReleaseName            = "dockyards.io/release-name"
	LabelCredentialTemplateName = "dockyards.io/credential-template-name"
)

const (
	ProvenienceDockyards = "Dockyards"
	ProvenienceUser      = "User"
)

const (
	SecretTypeCredential = "dockyards.io/credential"
)

const (
	AnnotationDefaultTemplate = "dockyards.io/is-default-template"
	AnnotationVoucherCode     = "dockyards.io/voucher-code"
)
