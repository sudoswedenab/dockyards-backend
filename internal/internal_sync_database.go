package internal

import (
	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"gorm.io/gorm"
)

func SyncDataBase(db *gorm.DB) error {
	err := db.AutoMigrate(&v1.Credential{})
	if err != nil {
		return err
	}

	err = db.AutoMigrate(&v1.App{})
	if err != nil {
		return err
	}

	err = db.AutoMigrate(&v1.AppStep{})
	if err != nil {
		return err
	}

	err = db.AutoMigrate(&v1.StepOption{})
	if err != nil {
		return err
	}

	return nil
}
