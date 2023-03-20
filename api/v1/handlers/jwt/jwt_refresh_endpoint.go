package jwt

import (
	"fmt"
	"net/http"
	"time"

	"bitbucket.org/sudosweden/backend/api/v1/model"
	"bitbucket.org/sudosweden/backend/internal"

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
func (h *handler) refreshTokenEndpoint(c *gin.Context) error {
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

		First := h.db.First(&user, claims["sub"])

		// replace with jwt response
		if First.Error == nil {
			newTokenPair, err := h.generateTokenPair(user)
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
