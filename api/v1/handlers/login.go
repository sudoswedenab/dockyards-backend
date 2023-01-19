package handlers

import (
	"Backend/api/v1/handlers/rancher"
	"Backend/api/v1/model"
	"Backend/internal"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/bcrypt"
)

// Login godoc
//
//	@Summary		Login to app
//	@Tags				Login
//	@Accept			application/json
//	@Produce		text/plain
//	@Param			id		path	int	true	"Account ID"
//	@Success		200
//	@Router			/login [post]
func Login(c *gin.Context) {

	fmt.Println("Login hit")

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
	var user model.User

	internal.DB.First(&user, "email = ?", body.Email)

	if user.ID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid email or password",
		})
		return
	}

	bearertoken := rancher.RancherCheck(user)
	fmt.Println(bearertoken)
	//Compare sent in pass with saved user pass hash
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(body.Password))

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid email or password",
		})
		return
	}
	// claims["aud"] = RancherBearerToken
	//Generate a jwt token
	token := jwt.New(jwt.SigningMethodHS256)
	fmt.Println("TOKENSTRING", token)
	claims := token.Claims.(jwt.MapClaims)
	claims["aud"] = bearertoken
	claims["sub"] = user.ID
	claims["name"] = user.Name
	claims["admin"] = false
	claims["exp"] = time.Now().Add(time.Minute * 15).Unix()

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
	rtClaims["exp"] = time.Now().Add(time.Hour * 1).Unix()

	rt, rerr := refreshToken.SignedString([]byte(os.Getenv("RefSECERET")))

	if rerr != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "failed to create Token",
		})
		return
	}

	// Send it back as a Cookie
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("access_token", tokenString, 900, "", "", false, true)
	c.SetCookie("refresh_token", rt, 3600*1, "", "", false, true)

	c.String(http.StatusOK, "Success.\n")
}
