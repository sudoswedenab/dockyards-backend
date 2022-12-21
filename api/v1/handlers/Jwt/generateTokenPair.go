package jwt

import (
	"Backend/api/v1/models"
	"os"
	"time"

	"github.com/golang-jwt/jwt"
)

func GenerateTokenPair() (map[string]string, error) {
	// Create token
	token := jwt.New(jwt.SigningMethodHS256)

	//Look up requested User
	var user models.User
	// Set claims
	// This is the information which frontend can use
	// The backend can also decode the token and get admin etc.
	claims := token.Claims.(jwt.MapClaims)
	claims["sub"] = user.ID
	claims["name"] = user.Name
	claims["admin"] = false
	claims["exp"] = time.Now().Add(time.Minute * 15).Unix()

	// Generate encoded token and send it as response.
	// The signing string should be secret (a generated UUID works too)
	t, err := token.SignedString([]byte(os.Getenv("SECERET")))
	if err != nil {
		return nil, err
	}

	refreshToken := jwt.New(jwt.SigningMethodHS256)
	rtClaims := refreshToken.Claims.(jwt.MapClaims)
	rtClaims["sub"] = user.ID
	rtClaims["exp"] = time.Now().Add(time.Hour * 1).Unix()

	rt, err := refreshToken.SignedString([]byte(os.Getenv("RefSECERET")))
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"access_token":  t,
		"refresh_token": rt,
	}, nil

	// if err != nil {
	// 	c.JSON(http.StatusBadRequest, gin.H{
	// 		"error": err,
	// 	})
	// 	return
	// }

	// c.JSON(http.StatusOK, gin.H{
	// 	"token":         t,
	// 	"refresh_token": rt,
	// 	"Welcomme user": user.Name,
	// })
}
