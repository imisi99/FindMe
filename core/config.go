package core

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"findme/model"
	"fmt"
	"log"
	norm "math/rand"
	"net/http"
	"strings"

	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)


var (
	JWTSecret = os.Getenv("JWTSECRET")
 	JWTExpiry = time.Hour * 24
	HttpClient = &http.Client{Timeout: 10 * time.Second,}
)

const (
	charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()-+_?,."
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


func GenerateState() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {return "", err}
	return base64.URLEncoding.EncodeToString(b), nil
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


func GenerateJWT(userID uint) (string, error){
	claims := JWTClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(JWTExpiry)),
			IssuedAt: jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(JWTSecret))
}


func ValidateJWT(tokenSting string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenSting, &JWTClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, &CustomMessage{Code: 400, Message: "Invalid Token!"}
		}
		return []byte(JWTSecret), nil
	})

	if err != nil {
		return nil, &CustomMessage{Code: 400, Message: "Expired Token!", Detail: err.Error()}
	}

	payload, ok := token.Claims.(*JWTClaims)

	if ok && token.Valid {
		return payload, nil
	}

	return nil, &CustomMessage{Code: 400, Message: "Invalid Token!"}
}


func Authorization(db *gorm.DB, username, password string) (string, error) {
	var user model.User

	err := db.Where("username = ? OR email = ?", username, username).First(&user).Error
	if err != nil { return "", &CustomMessage{Code: 404, Message: "Invalid Credentials!", Detail: err.Error()}}

	err = VerifyHashedPassword(password, user.Password)
	if err != nil {return "", &CustomMessage{Code: 404, Message: "Invalid Credentials!", Detail: err.Error()}}

	jwtToken, err := GenerateJWT(user.ID) 
	if err != nil { return "", &CustomMessage{Code: 500, Message: "Failed to generate jwt token", Detail: err.Error()}}

	return jwtToken, nil
}


func Authentication() gin.HandlerFunc{
	return func(ctx *gin.Context) {
		authHeader := ctx.GetHeader("Authorization")

		if authHeader == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Authorization header missing!"})
			return 
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Bearer token not found in the authorization header!"})
			return 
		}

		tokenString := parts[1]

		payload, err := ValidateJWT(tokenString)
		if err != nil {
			cm := err.(*CustomMessage)
			ctx.AbortWithStatusJSON(cm.Code, gin.H{"message": cm.Message, "detail": cm.Detail})
			return 
		}

		ctx.Set("userID", payload.UserID)

		ctx.Next()
	}
}


func CacheSkills(db *gorm.DB, rdb *redis.Client) {
	var skills []model.Skill
	if err := db.Find(&skills).Error; err != nil {
		log.Fatalf("An error occured while fetching skills from db -> %v", err)
	}

	skillName := make(map[string]uint, 0)

	for _, skill := range skills {
		skillName[skill.Name] = skill.ID
	}

	data, _ := json.Marshal(skillName)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := rdb.Set(ctx, "skills", data, 0).Result(); err != nil {
		log.Fatalf("An error occured while trying to set skills in redis -> %v", err)
	}
}	


func RetrieveCachedSkills(rdb *redis.Client) map[string]uint {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	val, err := rdb.Get(ctx, "skills").Result()
	if err != nil {
		log.Printf("Error retrieving cached skills: %v", err)
		return nil
	}

	var skills map[string]uint
	if err := json.Unmarshal([]byte(val), &skills); err != nil {
		log.Printf("Error unmarshalling cached skills: %v", err)
		return nil
	}

	return skills
}
