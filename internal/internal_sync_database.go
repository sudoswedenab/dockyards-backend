package internal

import (
	"bitbucket.org/sudosweden/backend/api/v1/model"
	"gorm.io/gorm"
)

func SyncDataBase(db *gorm.DB) error {
	return db.AutoMigrate(&model.User{})
}
