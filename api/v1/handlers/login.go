package handlers

import (
	"Backend/api/v1/models"
	"Backend/internal"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/bcrypt"
)

func Login(c *gin.Context) {
	println("Login hit")

	// Get email and pass off req body
	var body struct {
		Email    string
		Password string
	}

	if c.Bind(&body) != nil {

		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to read Body",
		})
		return
	}
	//Look up requested User
	var user models.User

	internal.DB.First(&user, "email = ?", body.Email)

	if user.ID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid email or password",
		})
		return
	}

	//Compare sent in pass with saved user pass hash
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(body.Password))

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid email or password",
		})
		return
	}
	//Generate a jwt token
	token := jwt.New(jwt.SigningMethodHS256)

	claims := token.Claims.(jwt.MapClaims)
	claims["sub"] = user.ID
	claims["name"] = user.Name
	claims["admin"] = false
	claims["exp"] = time.Now().Add(time.Minute * 15).Unix()

	// token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
	// 	"sub": user.ID,
	// 	"exp": time.Now().Add(time.Hour * 24 * 30).Unix(),
	// })

	// Sign and get the complete encoded token as a string using the secret
	tokenString, err := token.SignedString([]byte(os.Getenv("SECERET")))

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "failed to create Token",
		})
		return
	}

	refreshToken := jwt.New(jwt.SigningMethodHS256)

	rtClaims := refreshToken.Claims.(jwt.MapClaims)
	rtClaims["sub"] = user.ID
	rtClaims["exp"] = time.Now().Add(time.Hour * 24).Unix()

	rt, rerr := refreshToken.SignedString([]byte("secret"))

	if rerr != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "failed to create Token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":         tokenString,
		"refresh_token": rt,
		"Welcomme user": user.Name,
	})

	// // Send it back as a Cookie
	// c.SetSameSite(http.SameSiteLaxMode)
	// c.SetCookie("Authorization", tokenString, 3600*24*30, "", "", false, true)
	// c.SetCookie("refresh_token", rt, 3600*24*30, "", "", false, true)
}
