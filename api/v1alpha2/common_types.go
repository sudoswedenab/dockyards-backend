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
