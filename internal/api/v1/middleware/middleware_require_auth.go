package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func (h *Handler) RequireAuth(c *gin.Context) {
	authorizationHeader := c.GetHeader("Authorization")
	if authorizationHeader == "" {
		h.Logger.Debug("empty or missing authorization header")
		c.AbortWithStatus(http.StatusUnauthorized)

		return
	}

	bearerToken := strings.TrimPrefix(authorizationHeader, "Bearer ")

	// Parse takes the token string and a function for looking up the key.
	token, err := jwt.Parse(bearerToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
		return h.AccessPublicKey, nil
	})

	if err != nil {
		h.Logger.Debug("error parsing bearer token", "err", err)
		c.AbortWithStatus(http.StatusUnauthorized)

		return
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		//Check the exp
		if float64(time.Now().Unix()) > claims["exp"].(float64) {
			h.Logger.Debug("jwt token expired", "exp", claims["exp"])
			c.AbortWithStatus(http.StatusUnauthorized)

			return
		}

		//Attach to req
		c.Set("sub", claims["sub"])
		//Continue
		c.Next()
	} else {
		h.Logger.Debug("invalid token", "token", token)

		c.AbortWithStatus(http.StatusUnauthorized)
	}
}
