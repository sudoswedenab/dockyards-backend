package internal

import "Backend/api/v1/model"

func SyncDataBase() {
	DB.AutoMigrate(&model.User{})
}
