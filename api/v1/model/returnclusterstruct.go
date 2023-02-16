package model

type ReturnClusterResponse struct {
	Data []Data
}

type Data struct {
	Name                 string         `json:"name"`
	Id                   string         `json:"id"`
	CreatorId            string         `json:"creatorId"`
	Created              string         `json:"created"`
	State                string         `json:"state"`
	NodeCount            int            `json:"nodeCount"`
	Transitioning        string         `json:"transitioning"`
	TransitioningMessage string         `json:"transitioningMessage"`
	Conditions           []ConditionsOn `json:"conditions"`
}

type ConditionsOn struct {
	Status string `json:"status"`
	Type   string `json:"type"`
}
