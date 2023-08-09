package model

import "time"

type Cluster struct {
	Name         string     `json:"name"`
	ID           string     `json:"id"`
	State        string     `json:"state"`
	NodeCount    int        `json:"node_count"`
	CreatedAt    time.Time  `json:"created_at"`
	Organization string     `json:"org,omitempty"`
	NodePools    []NodePool `json:"node_pools,omitempty"`
	Version      string     `json:"version"`
}
