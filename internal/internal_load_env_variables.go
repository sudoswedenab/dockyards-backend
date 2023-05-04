package internal

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

var (
	RefreshTokenName   string
	AccessTokenName    string
	Secret             string
	RefSecret          string
	CattleUrl          string
	CattleBearerToken  string
	FlagUseCors        = false
	FlagServerCookie   = false
	OpenstackAuthURL   string
	OpenstackAppID     string
	OpenstackAppSecret string
)

func LoadEnvVariables() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Cloud not load .env file")
	}

	FlagUseCors, err = strconv.ParseBool(os.Getenv("FLAG_USE_CORS"))
	if err != nil {
		fmt.Printf("error parsing: %s", err)
	}
	FlagServerCookie, err = strconv.ParseBool(os.Getenv("FLAG_SET_SERVER_COOKIE"))
	if err != nil {
		fmt.Printf("error parsing: %s", err)
	}
	AccessTokenName = os.Getenv("ACCESS_TOKEN_NAME")
	RefreshTokenName = os.Getenv("REFRESH_TOKEN_NAME")
	Secret = os.Getenv("SECRET")
	RefSecret = os.Getenv("REF_SECRET")
	CattleUrl = os.Getenv("CATTLE_URL")
	CattleBearerToken = os.Getenv("CATTLE_BEARER_TOKEN")
	OpenstackAuthURL = os.Getenv("OPENSTACK_AUTH_URL")
	OpenstackAppID = os.Getenv("OPENSTACK_APP_ID")
	OpenstackAppSecret = os.Getenv("OPENSTACK_APP_SECRET")
}
