package jwt

import (
	"Backend/api/v1/handlers/rancher"
	"Backend/api/v1/model"
	"Backend/internal"
	"time"

	"github.com/golang-jwt/jwt"
)

func GenerateTokenPair(user model.User) (map[string]string, error) {
	// Create token
	token := jwt.New(jwt.SigningMethodHS256)

	//Checking agianst Racnher if user exist in rancher
	bearertoken, err := rancher.RancherLogin(user)
	if err != nil {
		return nil, err
	}

	// Set claims
	claims := token.Claims.(jwt.MapClaims)
	claims["aud"] = bearertoken
	claims["sub"] = user.ID
	claims["name"] = user.Name
	claims["admin"] = false
	claims["exp"] = time.Now().Add(time.Minute * 15).Unix()

	// Generate encoded token and send it as response.
	t, err := token.SignedString([]byte(internal.Secret))
	if err != nil {
		return nil, err
	}

	refreshToken := jwt.New(jwt.SigningMethodHS256)
	rtClaims := refreshToken.Claims.(jwt.MapClaims)
	rtClaims["sub"] = user.ID
	rtClaims["exp"] = time.Now().Add(time.Hour * 1).Unix()

	rt, err := refreshToken.SignedString([]byte(internal.RefSecret))
	if err != nil {
		return nil, err
	}

	return map[string]string{
		internal.AccessTokenName:  t,
		internal.RefreshTokenName: rt,
	}, nil
}
