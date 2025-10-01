package core

import (
	"crypto/rand"
	"encoding/base64"
	norm "math/rand"

	"fmt"
	"golang.org/x/crypto/bcrypt"
)


const (
	charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()-+_?,."
	otpset = "1234567890"
)



type CustomMessage struct {
	Code int
	Message string
}


func (cm *CustomMessage) Error() string{
	return fmt.Sprintf("An error occured -> %s", cm.Message)
}


func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(hashedPassword), err
}


func GenerateState() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {return "", err}
	return base64.URLEncoding.EncodeToString(b), nil
}


func GenerateOTP() string{
	b := make([]byte, 6)
	for i := range b {
		b[i] = otpset[norm.Intn(len(otpset))]
	}
	return string(b)
}


func GenerateUsername(username string) string {
	b := make([]byte, 9)
	for i := range b {
		b[i] = charset[norm.Intn(len(charset))]
	}
	return username+"_"+string(b)
}


func VerifyHashedPassword(password, hashedPassword string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err
}
