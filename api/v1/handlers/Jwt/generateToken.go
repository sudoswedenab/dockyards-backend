package jwt

// package jwt

// import (
// 	"Backend/api/v1/models"
// 	"net/http"
// 	"os"
// 	"time"

// 	"github.com/gin-gonic/gin"
// 	"github.com/golang-jwt/jwt"
// )

// func GenerateToken(c *gin.Context) {

// 	//Look up requested User
// 	var user models.User

// 	//Generate a jwt token
// 	token := jwt.New(jwt.SigningMethodHS256)

// 	claims := token.Claims.(jwt.MapClaims)
// 	claims["sub"] = user.ID
// 	claims["name"] = user.Name
// 	claims["admin"] = false
// 	claims["exp"] = time.Now().Add(time.Minute * 15).Unix()

// 	// token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
// 	// 	"sub": user.ID,
// 	// 	"exp": time.Now().Add(time.Hour * 24 * 30).Unix(),
// 	// })

// 	// Sign and get the complete encoded token as a string using the secret
// 	tokenString, err := token.SignedString([]byte(os.Getenv("SECERET")))

// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"error": "failed to create Token",
// 		})
// 		return
// 	}

// 	refreshToken := jwt.New(jwt.SigningMethodHS256)

// 	rtClaims := refreshToken.Claims.(jwt.MapClaims)
// 	rtClaims["sub"] = user.ID
// 	rtClaims["exp"] = time.Now().Add(time.Hour * 24).Unix()

// 	rt, rerr := refreshToken.SignedString([]byte("secret"))

// 	if rerr != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{
// 			"error": "failed to create Token",
// 		})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"token":         tokenString,
// 		"refresh_token": rt,
// 		"Welcomme user": user.Name,
// 	})

// 	// Send it back as a Cookie
// 	c.SetSameSite(http.SameSiteLaxMode)
// 	c.SetCookie("Authorization", tokenString, 3600*24*30, "", "", false, true)
// 	c.SetCookie("refresh_token", rt, 3600*24*30, "", "", false, true)

// 	//"token": tokenString
// 	// 	//send it back as a token string example
// 	// 	c.JSON(http.StatusOK, gin.H{
// 	// 		"token": tokenString,
// 	// 	})
// }

// c.JSON(http.StatusOK, gin.H{
// 		"token":         tokenString,
// 		"refresh_token": rt,
// 		"Welcomme user": user.Name,
// 	})
