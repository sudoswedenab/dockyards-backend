package model

type Signup struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	Name      string `json:"name"`
	RancherID string
}
