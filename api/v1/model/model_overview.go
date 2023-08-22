package model

import (
	"github.com/google/uuid"
)

type UserOverview struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
}

type DeploymentOverview struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type ClusterOverview struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	Deployments []DeploymentOverview `json:"apps"`
}

type OrganizationOverview struct {
	ID       uuid.UUID         `json:"id"`
	Name     string            `json:"name"`
	Clusters []ClusterOverview `json:"clusters"`
	Users    []UserOverview    `json:"users"`
}

type Overview struct {
	Organizations []OrganizationOverview `json:"organizations"`
}
