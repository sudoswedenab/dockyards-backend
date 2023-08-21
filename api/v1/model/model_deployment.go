package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type HelmValues map[string]any

type Deployment struct {
	ID             uuid.UUID  `json:"id"`
	Name           string     `json:"name"`
	ContainerImage string     `json:"container_image,omitempty"`
	Organization   string     `json:"organization"`
	Cluster        string     `json:"cluster"`
	Port           int        `json:"port,omitempty"`
	HelmChart      string     `json:"helm_chart,omitempty"`
	HelmRepository string     `json:"helm_repository,omitempty"`
	HelmVersion    string     `json:"helm_version,omitempty"`
	HelmValues     HelmValues `json:"helm_values,omitempty"`
	Namespace      string     `json:"namespace,omitempty"`
	CredentialID   *uuid.UUID `json:"credential_id,omitempty"`
}

func (d *Deployment) BeforeCreate(tx *gorm.DB) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}

	return nil
}

func (v *HelmValues) Scan(source any) error {
	switch source := source.(type) {
	case []byte:
		err := json.Unmarshal(source, &v)
		if err != nil {
			return nil
		}
	default:
		return fmt.Errorf("cannot scan helm values of type %T", source)
	}

	return nil
}

func (v HelmValues) Value() (driver.Value, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	return b, nil
}
