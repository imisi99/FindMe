package core

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	norm "math/rand"
	"time"

	"findme/model"

	"golang.org/x/crypto/bcrypt"
)

const (
	charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()-+_?,."
	otpset  = "1234567890"
)

type CustomMessage struct {
	Code    int
	Message string
}

func (cm *CustomMessage) Error() string {
	return fmt.Sprintf(" [CM] An error occured -> %s", cm.Message)
}

// HashPassword -> hash the password provided
func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(hashedPassword), err
}

// VerifyHashedPassword -> Verifies a password with it's hash
func VerifyHashedPassword(password, hashedPassword string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err
}

// GenerateState -> Generate state (Github signup)
func GenerateState() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// GenerateOTP -> Generate OTP
func GenerateOTP() string {
	b := make([]byte, 6)
	for i := range b {
		b[i] = otpset[norm.Intn(len(otpset))]
	}
	return string(b)
}

// GenerateUsername -> Generates a new username from existing one
func GenerateUsername(username string) string {
	b := make([]byte, 9)
	for i := range b {
		b[i] = charset[norm.Intn(len(charset))]
	}
	return username + "_" + string(b)
}

func CheckSubscription(user *model.User) bool {
	premium := false
	if len(user.Subscriptions) > 0 {
		for _, sub := range user.Subscriptions { // TODO: This is probably not the best way to do this (not efficient) but works for now
			if time.Now().Before(sub.EndDate) {
				premium = true
			}
		}
	}
	return premium
}
