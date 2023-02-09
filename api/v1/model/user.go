package model

import (
	"time"
)

// User.go line 42
// User example
type User struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Idn       int    `json:"idn"`
	Name      string `json:"name"`
	Email     string `json:"email" gorm:"unique"`
	Password  string `json:"password"`
	RancherID string `json:"rancherID"`
}
