package model

type RancherUser struct {
	Enabled            bool   `json:"enabled"`
	MustChangePassword bool   `json:"mustChangePassword,omitempty"`
	Name               string `json:"name,omitempty"`
	Password           string `json:"password"`
	Username           string `json:"username"`
}
