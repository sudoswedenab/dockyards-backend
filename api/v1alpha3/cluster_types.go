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
	"fmt"
	"net"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	apiserverv1 "k8s.io/apiserver/pkg/apis/apiserver/v1beta1"
)

const (
	ClusterKind                         = "Cluster"
	ResourceCluster corev1.ResourceName = "cluster"
)

// ClusterUpgrade describes a requested cluster version upgrade.
type ClusterUpgrade struct {
	// To is the Kubernetes version to upgrade the cluster to.
	To string `json:"to"`
}

// ClusterAPIEndpoint defines the external Kubernetes API endpoint for a cluster.
type ClusterAPIEndpoint struct {
	// Host is the DNS name or IP address of the API endpoint.
	Host string `json:"host"`
	// Port is the TCP port exposed by the API endpoint.
	Port int32  `json:"port"`
}

func (e *ClusterAPIEndpoint) IsValid() bool {
	return e.Host != "" && e.Port != 0
}

func (e *ClusterAPIEndpoint) String() string {
	port := fmt.Sprintf("%d", e.Port)

	return "https://" + net.JoinHostPort(e.Host, port)
}

type Patch = runtime.RawExtension

// TalosInstallImageOptions customizes how Talos installer images are sourced.
type TalosInstallImageOptions struct {
	// URL points to a non-image factory Talos installer archive.
	URL string `json:"url,omitempty"`

	// Size of the DataVolume to be allocated for this image.
	Size string `json:"size,omitempty"`
}

func (o *TalosInstallImageOptions) IsZero() bool {
	if o.URL != "" {
		return false
	}

	if o.Size != "" {
		return false
	}

	return true
}

// ClusterTalosOptions defines Talos-specific cluster options.
type ClusterTalosOptions struct {
	// Talos install image options.
	InstallImage TalosInstallImageOptions `json:"installImage,omitempty,omitzero"`

	// Name of the additional network interface
	ExternalNodeInterface string `json:"externalNodeInterface,omitempty"`

	// IPv4 subnet to be used for the additional network interface
	ExternalNodeIPv4Subnet string `json:"externalNodeIPv4Subnet,omitempty"`

	// Additional patches to apply to the talosconfig of all nodes.
	AdditionalSharedConfigPatches []Patch `json:"additionalSharedConfigPatches,omitempty"`

	// Additional patches to apply to the talosconfig of controlplane nodes.
	AdditionalControlPlaneConfigPatches []Patch `json:"additionalControlPlaneConfigPatches,omitempty"`

	// Additional patches to apply to the talosconfig of worker nodes.
	AdditionalWorkerConfigPatches []Patch `json:"additionalWorkerConfigPatches,omitempty"`
}

func (o *ClusterTalosOptions) IsZero() bool {
	if !o.InstallImage.IsZero() {
		return false
	}

	if o.ExternalNodeInterface != "" {
		return false
	}

	if o.ExternalNodeIPv4Subnet != "" {
		return false
	}

	if len(o.AdditionalSharedConfigPatches) != 0 {
		return false
	}

	if len(o.AdditionalControlPlaneConfigPatches) != 0 {
		return false
	}

	if len(o.AdditionalWorkerConfigPatches) != 0 {
		return false
	}

	return true
}

// ClusterKubevirtOptions defines kubevirt-specific cluster options.
type ClusterKubevirtOptions struct {
	// Overrides the installation-wide default StorageClass for kubevirt DataVolumes.
	DataVolumeStorageClassName string `json:"dataVolumeStorageClassName,omitempty"`

	// Options to apply to talos
	Talos ClusterTalosOptions `json:"talos,omitempty,omitzero"`
}

func (o *ClusterKubevirtOptions) IsZero() bool {
	if o.DataVolumeStorageClassName != "" {
		return false
	}

	if !o.Talos.IsZero() {
		return false
	}

	return true
}

// Options to override the installation-wide Gateway API configuration.
type ClusterGatewayOptions struct {
	// Gateway parentRef settings used for this specific DockyardsCluster.
	ParentRef ClusterGatewayParentRefOptions `json:"parentRef,omitempty,omitzero"`
}

func (o *ClusterGatewayOptions) IsZero() bool {
	if !o.ParentRef.IsZero() {
		return false
	}

	return true
}

// Gateway parentRef settings used for this specific DockyardsCluster.
type ClusterGatewayParentRefOptions struct {
	// Gateway name to use for this cluster.
	Name string `json:"name,omitempty"`

	// Gateway namespace to use for this cluster.
	Namespace string `json:"namespace,omitempty"`
}

func (o *ClusterGatewayParentRefOptions) IsZero() bool {
	if o.Name != "" {
		return false
	}

	if o.Namespace != "" {
		return false
	}

	return true
}

// These are meant as an escape hatch for users who know it in their heart
// that on their cloud platform there is a configuration option that, when
// tuned, makes their cluster go wrooom, but since dockyards either hasn't
// or wont add explicit support for the feature, they cannot turn the knob.
//
// These options require an advanced knowledge of how the platform operates
// and which cloud environment is being actively used, so tread carefully
// when using these options.
type ClusterAdvancedOptions struct {
	// Options to override the installation-wide Gateway API configuration.
	Gateway ClusterGatewayOptions `json:"gateway,omitempty,omitzero"`

	// Options to apply to kubevirt in case we're running in a kubevirt environment.
	Kubevirt ClusterKubevirtOptions `json:"kubevirt,omitempty,omitzero"`
}

func (o *ClusterAdvancedOptions) IsZero() bool {
	if !o.Gateway.IsZero() {
		return false
	}

	if !o.Kubevirt.IsZero() {
		return false
	}

	return true
}

// ClusterSpec defines the desired configuration of a Dockyards cluster.
type ClusterSpec struct {
	// Version is the desired Kubernetes version for the cluster.
	Version                  string                                   `json:"version,omitempty"`
	// NoDefaultIngressProvider disables installation of the default ingress provider.
	NoDefaultIngressProvider bool                                     `json:"noDefaultIngressProvider,omitempty"`
	// Upgrades lists requested version upgrades for the cluster.
	Upgrades                 []ClusterUpgrade                         `json:"upgrades,omitempty"`
	// BlockDeletion prevents cluster deletion while set to true.
	BlockDeletion            bool                                     `json:"blockDeletion,omitempty"`
	// AllocateInternalIP requests allocation of an internal API endpoint IP.
	AllocateInternalIP       bool                                     `json:"allocateInternalIP,omitempty"`
	// IPPoolRef references the IP pool used for API endpoint allocation.
	IPPoolRef                *corev1.TypedLocalObjectReference        `json:"ipPoolRef,omitempty"`
	// Duration defines the cluster lifetime from creation time.
	Duration                 *metav1.Duration                         `json:"duration,omitempty"`
	// NoDefaultNetworkPlugin disables installation of the default network plugin.
	NoDefaultNetworkPlugin   bool                                     `json:"noDefaultNetworkPlugin,omitempty"`
	// PodSubnets configures CIDR ranges used for pod networking.
	PodSubnets               []string                                 `json:"podSubnets,omitempty"`
	// ServiceSubnets configures CIDR ranges used for service networking.
	ServiceSubnets           []string                                 `json:"serviceSubnets,omitempty"`
	// AuthenticationConfig provides custom Kubernetes API authentication settings.
	AuthenticationConfig     *apiserverv1.AuthenticationConfiguration `json:"authenticationConfig,omitempty"`
	// Advanced contains platform-specific override settings.
	Advanced                 ClusterAdvancedOptions                   `json:"advanced,omitempty,omitzero"`
}

// ClusterStatus defines the observed state of a Dockyards cluster.
type ClusterStatus struct {
	// Conditions represents the latest available observations of cluster state.
	Conditions          []metav1.Condition `json:"conditions,omitempty"`
	// ClusterServiceID identifies the corresponding cluster in backend services.
	ClusterServiceID    string             `json:"clusterServiceID,omitempty"`
	// Version is the currently observed Kubernetes version.
	Version             string             `json:"version,omitempty"`
	// DNSZones lists DNS zones assigned to the cluster.
	DNSZones            []string           `json:"dnsZones,omitempty"`
	// APIEndpoint is the observed external Kubernetes API endpoint.
	APIEndpoint         ClusterAPIEndpoint `json:"apiEndpoint,omitempty"`
	// ExpirationTimestamp is when the cluster is expected to expire.
	ExpirationTimestamp *metav1.Time       `json:"expirationTimestamp,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=".status.conditions[?(@.type==\"Ready\")].status"
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=".status.conditions[?(@.type==\"Ready\")].reason"
// +kubebuilder:printcolumn:name="Version",type=string,priority=1,JSONPath=".status.version"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Duration",type=string,priority=1,JSONPath=".spec.duration"
// Cluster represents a managed Kubernetes cluster in Dockyards.
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired cluster configuration.
	Spec   ClusterSpec   `json:"spec,omitempty"`
	// Status represents the observed cluster state.
	Status ClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// ClusterList contains a list of Cluster resources.
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items contains the list of Cluster resources.
	Items []Cluster `json:"items,omitempty"`
}

func (c *Cluster) GetConditions() []metav1.Condition {
	return c.Status.Conditions
}

func (c *Cluster) SetConditions(conditions []metav1.Condition) {
	c.Status.Conditions = conditions
}

func (c *Cluster) GetExpiration() *metav1.Time {
	if c.Spec.Duration == nil {
		return nil
	}

	expiration := c.CreationTimestamp.Add(c.Spec.Duration.Duration)

	return &metav1.Time{Time: expiration}
}

func init() {
	SchemeBuilder.Register(&Cluster{}, &ClusterList{})
}
