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

type ReleaseSpec struct {
	Type   ReleaseType `json:"type"`
	Ranges []string    `json:"ranges,omitempty"`
}

type ReleaseStatus struct {
	LatestVersion string   `json:"latestVersion,omitempty"`
	Versions      []string `json:"versions,omitempty"`
	LatestURL     *string  `json:"latestURL,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Latest",type=string,JSONPath=".status.latestVersion"
// +kubebuilder:storageversion
type Release struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ReleaseSpec   `json:"spec,omitempty"`
	Status ReleaseStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type ReleaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Release `json:"items,omitempty"`
}

func init() {
	SchemeBuilder.Register(&Release{}, &ReleaseList{})
}
