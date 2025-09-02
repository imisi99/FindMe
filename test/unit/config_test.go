package unit

import (
	"findme/core"
	"testing"

	"github.com/stretchr/testify/assert"
)

var token string
var id uint = 12
var hashPassword string

func TestGenerateJWT(t *testing.T) {
	token, _ = core.GenerateJWT(id, "login", core.JWTExpiry)
}


func TestValidateJWT(t *testing.T) {
	claims, _ := core.ValidateJWT(token)
	assert.Equal(t, claims.UserID, id)
}


func TestAuthorization(t *testing.T) {
	db := getTestDB()
	_, err := core.Authorization(db, "Imisioluwa23", "Password")

	assert.Nil(t, err)
}


func TestFailedAuthorization(t *testing.T) {
	db := getTestDB()
	_, err := core.Authorization(db, "Imisioluwa", "Password..")

	assert.Contains(t, err.Error(), "record not found")
}


func TestHashPassword(t *testing.T) {
	hashpassword, err := core.HashPassword("Imisioluwa")
	hashPassword = hashpassword
	assert.Nil(t, err)
}


func TestVerifyPassword(t *testing.T) {
	err := core.VerifyHashedPassword("Imisioluwa", hashPassword)
	assert.Nil(t, err)
}
