package core

import (
	"findme/model"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)


var (
	JWTSecret = os.Getenv("JWTSECRET")
	JWTExpiry = time.Hour * 24
)


type JWTClaims struct {
	UserID uint 
	jwt.RegisteredClaims
}


type CustomMessage struct {
	Code int
	Message string
	Detail string
}


func (cm *CustomMessage) Error() string{
	return fmt.Sprintf("An error occured -> %s", cm.Detail)
}


func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(hashedPassword), err
}


func VerifyHashedPassword(password, hashedPassword string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err
}


func GenerateJWT(userID uint) (string, error){
	claims := JWTClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(JWTExpiry)),
			IssuedAt: jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JWTSecret)
}


func Authorization(db *gorm.DB, username, password string) (string, error) {
	var user model.User

	err := db.Where("username = ? OR email = ?", username, username).Find(&user).Error
	if err != nil { return "", &CustomMessage{Code: 404, Message: "Invalid Credentials", Detail: err.Error()}}

	err = VerifyHashedPassword(password, user.Password)
	if err != nil {return "", &CustomMessage{Code: 404, Message: "Invalid Credentials", Detail: err.Error()}}

	jwtToken, err := GenerateJWT(user.ID) 
	if err != nil { return "", &CustomMessage{Code: 500, Message: "An error occured while generating jwt token", Detail: err.Error()}}

	return jwtToken, nil
}
