package jwt

import (
	"Backend/api/v1/model"
	"Backend/internal"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

// Validate godoc
//
//	@Summary		Refresh
//	@Tags			Refresh Token
//	@Accept			application/json
//	@Produce		application/json
//	@Success		200
//	@Router			/refresh [post]
func RefreshTokenEndpoint(c *gin.Context) error {

	//Get the cookie
	refreshToken, err := c.Cookie("refresh_token")
	fmt.Println("im here")
	if err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
	}
	// Parse the token string and a function for looking for the key.

	token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		// hmacSampleSecret is a []byte containing your incl secret key
		return []byte(os.Getenv("RefSECERET")), nil
	})

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		// Get the user record from database or
		if float64(time.Now().Unix()) > claims["exp"].(float64) {
			c.AbortWithStatus(http.StatusUnauthorized)
		}

		//Find the user with token sub
		var user model.User

		First := internal.DB.First(&user, claims["sub"])

		// run through your business logic to verify if the user can log in
		if First.Error == nil {

			newTokenPair, err := GenerateTokenPair()
			if err != nil {
				return err
			}
			c.SetCookie("access_token", newTokenPair["access_token"], 900, "", "", false, true)
			c.SetCookie("refresh_token", newTokenPair["refresh_token"], 3600*1, "", "", false, true)
		}
	}
	return err
}
