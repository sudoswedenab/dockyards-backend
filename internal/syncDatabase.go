package internal

import "Backend/api/v1/models"

func SyncDataBase() {
	DB.AutoMigrate(&models.User{})
}
