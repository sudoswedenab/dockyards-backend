package internal

import (
	"bitbucket.org/sudosweden/backend/api/v1/model"
	"gorm.io/gorm"
)

func SyncDataBase(db *gorm.DB) {
	err := db.AutoMigrate(&model.User{})
	if err != nil {
		return
	}
}
