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

package v1alpha3

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ReleaseType string

const (
	ReleaseKind = "Release"
)

const (
	ReleaseTypeKubernetes     ReleaseType = "KubernetesReleases"
	ReleaseTypeTalosInstaller ReleaseType = "TalosInstaller"
)

// ReleaseSpec defines the desired release query configuration.
type ReleaseSpec struct {
	// Type selects which release catalog is queried.
	Type   ReleaseType `json:"type"`
	// Ranges filters versions using semver ranges.
	Ranges []string    `json:"ranges,omitempty"`
}

// ReleaseStatus defines the observed release catalog state.
type ReleaseStatus struct {
	// LatestVersion is the newest version matching the requested criteria.
	LatestVersion string   `json:"latestVersion,omitempty"`
	// Versions lists all discovered versions matching the requested criteria.
	Versions      []string `json:"versions,omitempty"`
	// LatestURL is an optional download URL for the latest version.
	LatestURL     *string  `json:"latestURL,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Latest",type=string,JSONPath=".status.latestVersion"
// +kubebuilder:storageversion
// Release represents version metadata for a release channel.
type Release struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired release query configuration.
	Spec   ReleaseSpec   `json:"spec,omitempty"`
	// Status represents the observed release catalog state.
	Status ReleaseStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// ReleaseList contains a list of Release resources.
type ReleaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items contains the list of Release resources.
	Items []Release `json:"items,omitempty"`
}

func init() {
	SchemeBuilder.Register(&Release{}, &ReleaseList{})
}
