package jwt

import (
	"Backend/api/v1/model"
	"Backend/internal"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

// RefreshTokenEndpoint godoc
//
//	@Summary		Refresh Token
//	@Tags			Login
//	@Accept			application/json
//	@Produce		application/json
//	@Success		200
//	@Failure		401
//	@Router			/refresh [post]
func RefreshTokenEndpoint(c *gin.Context) error {

	// Get the cookie
	refreshToken, err := c.Cookie(internal.RefreshTokenName)

	if err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
	}
	// Parse the token string and a function for looking for the key.

	token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// hmacSampleSecret is a []byte containing your incl secret key
		return []byte(internal.RefSecret), nil
	})

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		// Get the user record from database or
		if float64(time.Now().Unix()) > claims["exp"].(float64) {
			c.AbortWithStatus(http.StatusUnauthorized)
		}

		//Find the user with token sub
		var user model.User

		First := internal.DB.First(&user, claims["sub"])

		// replace with jwt response
		if First.Error == nil {
			newTokenPair, err := GenerateTokenPair(user)
			if err != nil {
				return err
			}
			c.JSON(http.StatusOK, gin.H{
				"Login":                   "Success",
				internal.AccessTokenName:  newTokenPair[internal.AccessTokenName],
				internal.RefreshTokenName: newTokenPair[internal.RefreshTokenName],
			})
		}
	}
	return err
}
