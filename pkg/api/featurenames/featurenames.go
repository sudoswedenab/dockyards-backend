package featurenames

type FeatureName string

const (
	FeatureLoadBalancerRole FeatureName = "load-balancer-role"
)

const (
	FeatureStorageRole FeatureName = "storage-role"
)

const (
	FeatureTalosconfig FeatureName = "talosconfig"
)

const (
	FeatureMetrics FeatureName = "metrics"
)

const (
	FeatureSafespringElasticNetworks = "safespring-elastic-networks"
)

const (
	FeatureOrganizationAutoAssign = "organization-auto-assign"
)

const (
	FeatureCostEstimates = "cost-estimates"
)

const (
	// Deprecated: use FeatureAPIEquivalents
	FeatureAPIExplanations = "api-explanations"
)

const (
	FeatureDNSZones = "dns-zones"
)

const (
	FeatureAPIEquivalents = "api-equivalents"
)

const (
	FeatureClusterUpgrades = "cluster-upgrades"
)

const (
	FeatureImmutableResources = "immutable-resources"
)

const (
	FeatureCloudDirectorAPITokens = "cloud-director-api-tokens"
)
