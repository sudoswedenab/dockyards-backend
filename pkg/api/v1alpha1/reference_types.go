package v1alpha1

type CloudReference struct {
	APIVersion string `json:"apiVersion,omitempty"`
	Kind       string `json:"kind,omitempty"`
	Name       string `json:"name"`
	SecretRef  string `json:"secretRef,omitempty"`
}

type UserReference struct {
	Kind string `json:"kind,omitempty"`
	Name string `json:"name,omitempty"`
}
