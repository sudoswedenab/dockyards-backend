package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type HelmValues map[string]any

type App struct {
	ID             uuid.UUID  `json:"id"`
	Name           string     `json:"name"`
	ContainerImage string     `json:"container_image"`
	Organization   string     `json:"organization"`
	Cluster        string     `json:"cluster"`
	Port           int        `json:"port"`
	HelmChart      string     `json:"helm_chart,omitempty"`
	HelmRepository string     `json:"helm_repository,omitempty"`
	HelmVersion    string     `json:"helm_version,omitempty"`
	HelmValues     HelmValues `json:"helm_values,omitempty"`
	Namespace      string     `json:"namespace,omitempty"`
}

func (a *App) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}

	return nil
}
