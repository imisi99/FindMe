// Package handlers -> Endpoints for the app
package handlers

import (
	"net/http"
	"os"
	"strings"
	"time"

	"findme/core"
	"findme/model"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

type JWTClaims struct {
	UserID  string
	Purpose string
	Premium bool
	jwt.RegisteredClaims
}

var (
	JWTSecret  = os.Getenv("JWTSECRET")
	JWTExpiry  = time.Hour * 24
	JWTRExpiry = time.Minute * 5
	HTTPClient = &http.Client{Timeout: 10 * time.Second}
	upgrade    = websocket.Upgrader{
		ReadBufferSize:  2048,
		WriteBufferSize: 2048,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

// GenerateJWT -> Generates JWT token
func GenerateJWT(userID string, purpose string, premium bool, expiry time.Duration) (string, error) {
	claims := JWTClaims{
		UserID:  userID,
		Purpose: purpose,
		Premium: premium,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(JWTSecret))
}

// ValidateJWT -> Validates JWT token for authentication
func ValidateJWT(tokenSting string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenSting, &JWTClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, &core.CustomMessage{Code: 400, Message: "Invalid Token!"}
		}
		return []byte(JWTSecret), nil
	})
	if err != nil {
		return nil, &core.CustomMessage{Code: 400, Message: "Expired Token!"}
	}

	payload, ok := token.Claims.(*JWTClaims)

	if ok && token.Valid {
		return payload, nil
	}

	return nil, &core.CustomMessage{Code: 400, Message: "Invalid Token!"}
}

// Authorization -> Authorize user
func Authorization(user *model.User, password string) (string, error) {
	err := core.VerifyHashedPassword(password, user.Password)
	if err != nil {
		return "", &core.CustomMessage{Code: 404, Message: "Invalid Credentials!"}
	}

	premium := CheckSubscription(user)

	jwtToken, err := GenerateJWT(user.ID, "login", premium, JWTExpiry)
	if err != nil {
		return "", &core.CustomMessage{Code: 500, Message: "Failed to generate jwt token"}
	}

	return jwtToken, nil
}

// CheckSubscription -> Checks if a user has a current subscription
func CheckSubscription(user *model.User) bool {
	premium := false

	if time.Now().Before(user.FreeTrial) {
		return true
	}

	if len(user.Subscriptions) > 0 {
		for _, sub := range user.Subscriptions { // TODO: This is probably not the best way to do this (not efficient) but works for now
			if time.Now().Before(sub.EndDate) {
				premium = true
				break
			}
		}
	}
	return premium
}

// Authentication -> Authenticate user
func (s *Service) Authentication() gin.HandlerFunc {
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
			cm := err.(*core.CustomMessage)
			ctx.AbortWithStatusJSON(cm.Code, gin.H{"message": cm.Message})
			return
		}

		ctx.Set("userID", payload.UserID)
		ctx.Set("purpose", payload.Purpose)
		ctx.Set("premium", payload.Premium)

		ctx.Next()
	}
}

// CheckAndUpdateSkills -> Helper func for checking and updating skills
func (s *Service) CheckAndUpdateSkills(payload []string) ([]*model.Skill, error) {
	skills, err := s.RDB.RetrieveCachedSkills(payload)
	if err != nil { // Falling back to the db if the cache fails
		var existingSkills []*model.Skill

		if err := s.DB.FindExistingSkills(&existingSkills, payload); err != nil {
			return nil, err
		}

		existingSkillSet := make(map[string]bool)
		for _, name := range existingSkills {
			existingSkillSet[name.Name] = true
		}

		var newSkill []*model.Skill
		for _, skill := range payload {
			if _, exists := existingSkillSet[skill]; !exists {
				newSkill = append(newSkill, &model.Skill{Name: skill})
			}
		}

		if len(newSkill) > 0 {
			if err := s.DB.AddSkills(&newSkill); err != nil {
				return nil, err
			}
		}
		newSkill = append(newSkill, existingSkills...)
		return newSkill, nil
	}

	var newskills, allskills []*model.Skill
	for _, skill := range payload {
		if id, exists := skills[skill]; exists {
			allskills = append(allskills, &model.Skill{Name: skill, GormModel: model.GormModel{ID: id}})
			continue
		}
		newskills = append(newskills, &model.Skill{Name: skill})
	}

	if len(newskills) > 0 {
		if err := s.DB.AddSkills(&newskills); err != nil {
			return nil, err
		}
		s.RDB.AddNewSkillToCache(newskills)
	}
	allskills = append(allskills, newskills...)
	return allskills, nil
}
