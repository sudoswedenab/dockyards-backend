package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type App struct {
	ID             uuid.UUID `json:"-"`
	Name           string    `json:"name"`
	ContainerImage string    `json:"container_image"`
	Organization   string    `json:"organization"`
	Cluster        string    `json:"cluster"`
	Port           int       `json:"port"`
}

func (a *App) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}

	return nil
}
