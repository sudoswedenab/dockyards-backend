package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User.go line 42
// User example
type User struct {
	ID        uuid.UUID `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Name      string `json:"name"`
	Email     string `json:"email" gorm:"unique"`
	Password  string `json:"password"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}
