package model

type ClusterResponse struct {
	Data []Datan
}

type Datan struct {
	Name                 string          `json:"name"`
	CreatorId            string          `json:"creatorId"`
	Created              string          `json:"created"`
	State                string          `json:"state"`
	NodeCount            int             `json:"nodeCount"`
	Transitioning        string          `json:"transitioning"`
	TransitioningMessage string          `json:"transitioningMessage"`
	Conditions           []ConditionsOne `json:"conditions"`
}

type ConditionsOne struct {
	Status string `json:"status"`
	Type   string `json:"type"`
}
