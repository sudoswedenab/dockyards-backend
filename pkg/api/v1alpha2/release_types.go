package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ReleaseKind               = "Release"
	ReleaseTypeKubernetes     = "KubernetesReleases"
	ReleaseTypeTalosInstaller = "TalosInstaller"
)

const (
	ReleaseNameSupportedKubernetesVersions = "supported-kubernetes-versions"
	ReleaseNameCurrentTalosInstaller       = "current-talos-installer"
)

type ReleaseSpec struct {
	Type   string   `json:"type"`
	Ranges []string `json:"ranges,omitempty"`
}

type ReleaseStatus struct {
	LatestVersion string   `json:"latestVersion,omitempty"`
	Versions      []string `json:"versions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Latest",type=string,JSONPath=".status.latestVersion"
// +kubebuilder:deprecatedversion
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
