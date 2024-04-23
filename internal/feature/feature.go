package feature

import (
	"bitbucket.org/sudosweden/dockyards-backend/pkg/api/featurenames"
)

var (
	enabledFeatures = map[featurenames.FeatureName]bool{
		featurenames.FeatureLoadBalancerRole:       false,
		featurenames.FeatureStorageRole:            false,
		featurenames.FeatureOrganizationAutoAssign: false,
	}
)

func Enable(featureName featurenames.FeatureName) {
	enabledFeatures[featureName] = true
}

func Disable(featureName featurenames.FeatureName) {
	enabledFeatures[featureName] = false
}

func IsEnabled(featureName featurenames.FeatureName) bool {
	return enabledFeatures[featureName]
}
