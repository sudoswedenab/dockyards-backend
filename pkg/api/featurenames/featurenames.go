// Copyright 2024 Sudo Sweden AB
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	// Deprecated: superseded by CredentialTemplate type
	FeatureCloudDirectorAPITokens = "cloud-director-api-tokens"
)

const (
	FeatureWorkloads = "workloads"
)

const (
	FeatureImmutableStorageResources = "immutable-storage-resources"
)
