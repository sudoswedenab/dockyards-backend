package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID            uuid.UUID      `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	Name          string         `json:"name"`
	Email         string         `json:"email" gorm:"unique"`
	Password      string         `json:"password,omitempty"`
	Organizations []Organization `json:"orgs,omitempty" gorm:"many2many:organization_user"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}
