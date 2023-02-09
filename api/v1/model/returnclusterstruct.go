package model

type ReturnClusterResponse struct {
	Data []Data
	Id   string `json:"id"`
	Name string `json:"name"`
}

type Data struct {
	Name                 string         `json:"name"`
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
