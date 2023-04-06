package model

import "time"

type Cluster struct {
	Name      string    `json:"name"`
	ID        string    `json:"-"`
	State     string    `json:"state"`
	NodeCount int       `json:"node_count"`
	CreatedAt time.Time `json:"created_at"`
}
