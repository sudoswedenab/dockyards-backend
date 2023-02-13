package internal

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

var (
	RefreshTokenName string
	AccessTokenName  string
)

func LoadEnvVariables() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	AccessTokenName = os.Getenv("ACCESS_TOKEN_NAME")
	RefreshTokenName = os.Getenv("REFRESH_TOKEN_NAME")
}
