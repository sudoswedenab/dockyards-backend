package v1alpha1

import (
	"k8s.io/apimachinery/pkg/types"
)

type CloudReference struct {
	APIVersion string `json:"apiVersion,omitempty"`
	Kind       string `json:"kind,omitempty"`
	Name       string `json:"name"`
	Namespace  string `json:"namespace,omitempty"`
	SecretRef  string `json:"secretRef,omitempty"`
}

type UserReference struct {
	Name string    `json:"name"`
	UID  types.UID `json:"uid"`
}

type DeploymentReference struct {
	APIVersion string    `json:"apiVersion,omitempty"`
	Kind       string    `json:"kind,omitempty"`
	Name       string    `json:"name"`
	UID        types.UID `json:"uid"`
}
