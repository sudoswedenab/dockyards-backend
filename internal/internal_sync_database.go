package internal

import (
	"bitbucket.org/sudosweden/dockyards-backend/api/v1/model"
	"gorm.io/gorm"
)

func SyncDataBase(db *gorm.DB) error {
	err := db.AutoMigrate(&model.User{})
	if err != nil {
		return err
	}

	err = db.AutoMigrate(&model.Organization{})
	if err != nil {
		return err
	}

	err = db.AutoMigrate(&model.App{})
	if err != nil {
		return err
	}

	return nil
}
