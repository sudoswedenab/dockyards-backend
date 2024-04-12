package feature

import (
	dockyardsv1 "bitbucket.org/sudosweden/dockyards-backend/pkg/api/v1alpha1"
)

var (
	enabledFeatures = map[dockyardsv1.FeatureName]bool{
		dockyardsv1.LoadBalancerRoleFeature: false,
		dockyardsv1.StorageRoleFeature:      false,
	}
)

func Enable(featureName dockyardsv1.FeatureName) {
	enabledFeatures[featureName] = true
}

func Disable(featureName dockyardsv1.FeatureName) {
	enabledFeatures[featureName] = false
}

func IsEnabled(featureName dockyardsv1.FeatureName) bool {
	return enabledFeatures[featureName]
}
