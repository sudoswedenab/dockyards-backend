package v1alpha3

const (
	LabelOrganizationName       = "dockyards.io/organization-name"
	LabelClusterName            = "dockyards.io/cluster-name"
	LabelNodePoolName           = "dockyards.io/node-pool-name"
	LabelNodeName               = "dockyards.io/node-name"
	LabelDeploymentName         = "dockyards.io/deployment-name"
	LabelReleaseName            = "dockyards.io/release-name"
	LabelCredentialTemplateName = "dockyards.io/credential-template-name"
	LabelWorkloadTemplateName   = "dockyards.io/workload-template-name"
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
	AnnotationDefaultRelease  = "dockyards.io/is-default-release"
)
