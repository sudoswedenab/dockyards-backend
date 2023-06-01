package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CredentialData map[string]string

type Credential struct {
	ID           uuid.UUID      `json:"id"`
	Name         string         `json:"name"`
	Organization string         `json:"organization"`
	Data         CredentialData `json:"data,omitempty"`
}

func (c *Credential) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}

	return nil
}

func (d *CredentialData) Scan(value any) error {
	switch value := value.(type) {
	case []byte:
		err := json.Unmarshal(value, &d)
		if err != nil {
			return err
		}
	default:
		errors.New("unsupported value type")
	}

	return nil
}

func (d CredentialData) Value() (driver.Value, error) {
	b, err := json.Marshal(d)
	if err != nil {
		return nil, err
	}

	return b, nil
}
