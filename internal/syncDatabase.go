package internal

import "bitbucket.org/sudosweden/backend/api/v1/model"

func SyncDataBase() {
	DB.AutoMigrate(&model.User{})
}
