package v1alpha2

import (
	"k8s.io/apimachinery/pkg/types"
)

type MemberRole string

const (
	MemberRoleSuperUser MemberRole = "SuperUser"
	MemberRoleUser      MemberRole = "User"
	MemberRoleReader    MemberRole = "Reader"
)

type MemberReference struct {
	Group string `json:"group,omitempty"`
	Kind  string `json:"kind,omitempty"`
	Name  string `json:"name"`

	// +kubebuilder:validation:Enum=SuperUser;User;Reader
	Role MemberRole `json:"role"`

	UID types.UID `json:"uid"`
}

type NamespacedObjectReference struct {
	APIVersion string `json:"apiVersion,omitempty"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Namespace  string `json:"namespace"`
}

type NamespacedSecretReference struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}
