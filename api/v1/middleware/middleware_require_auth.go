package middleware

import (
	"fmt"
	"net/http"
	"time"

	"bitbucket.org/sudosweden/backend/api/v1/model"
	"bitbucket.org/sudosweden/backend/internal"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

func (h *Handler) RequireAuth(c *gin.Context) {
	// Get the cookie
	tokenString, err := c.Cookie(internal.AccessTokenName)
	if err != nil {
		h.Logger.Error("error fetching access token", "access_token_name", internal.AccessTokenName, "err", err)
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	// Parse takes the token string and a function for looking up the key.
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
		return []byte(internal.Secret), nil
	})

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		//Check the exp
		if float64(time.Now().Unix()) > claims["exp"].(float64) {
			h.Logger.Debug("jwt token expired", "exp", claims["exp"])
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		//Find the user with token sub
		var user model.User
		h.DB.First(&user, claims["sub"])

		if user.ID == 0 {
			h.Logger.Debug("no user found", "sub", claims["sub"])
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		//Attach to req
		c.Set("user", user)
		//Continue
		c.Next()
	} else {
		h.Logger.Debug("invalid token", "token", token)
		c.AbortWithStatus(http.StatusUnauthorized)
	}
}
