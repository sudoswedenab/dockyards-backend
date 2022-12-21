package jwt

import (
	"Backend/api/v1/models"
	"Backend/internal"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

// This is the api to refresh tokens
// Most of the code is taken from the jwt-go package's sample codes
// https://godoc.org/github.com/dgrijalva/jwt-go#example-Parse--Hmac

func RefreshTokenEndpoint(c *gin.Context) error {

	//Get the cookie
	refreshToken, err := c.Cookie("refresh_token")
	fmt.Println("im here")
	if err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
	}
	// Parse takes the token string and a function for looking up the key.
	// The latter is especially useful if you use multiple keys for your application.
	// The standard is to use 'kid' in the head of the token to identify
	// which key to use, but the parsed token (head and claims) is provided
	// to the callback, providing flexibility.

	token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
		return []byte(os.Getenv("RefSECERET")), nil
	})
	// fmt.Println(token)

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		// Get the user record from database or
		if float64(time.Now().Unix()) > claims["exp"].(float64) {
			c.AbortWithStatus(http.StatusUnauthorized)
		}
		// fmt.Println(token)

		//Find the user with token sub
		var user models.User

		First := internal.DB.First(&user, claims["sub"])

		// fmt.FPrintln(*First)
		fmt.Printf("%s\n", First.Error)
		// fmt.Println(&user)

		// fmt.Println(int(claims["sub"].(float64)))

		// run through your business logic to verify if the user can log in
		if First.Error == nil {

			newTokenPair, err := GenerateTokenPair()
			if err != nil {
				return err

			}
			c.SetCookie("access_token", newTokenPair["access_token"], 3600*24*30, "", "", false, true)
			c.SetCookie("refresh_token", newTokenPair["refresh_token"], 3600*24*30, "", "", false, true)
			// fmt.Println(newTokenPair)
			c.JSON(http.StatusOK, gin.H{
				"new": newTokenPair,
			})
		}
	}
	return err
}
