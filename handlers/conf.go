package handlers

import (
	"findme/core"
	"findme/model"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)


type JWTClaims struct {
	UserID uint 
	Purpose string
	jwt.RegisteredClaims
}

var (
	JWTSecret = os.Getenv("JWTSECRET")
 	JWTExpiry = time.Hour * 24
	JWTRExpiry = time.Minute * 5
	HttpClient = &http.Client{Timeout: 10 * time.Second}
)


func GenerateJWT(userID uint, purpose string, expiry time.Duration) (string, error){
	claims := JWTClaims{
		UserID: userID,
		Purpose: purpose,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt: jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(JWTSecret))
}


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


func (c *Service) Authorization(username, password string) (string, error) {
	var user model.User

	err := c.DB.Where("username = ? OR email = ?", username, username).First(&user).Error
	if err != nil { return "", &core.CustomMessage{Code: 404, Message: "Invalid Credentials!"}}

	err = core.VerifyHashedPassword(password, user.Password)
	if err != nil { return "", &core.CustomMessage{Code: 404, Message: "Invalid Credentials!"}}

	jwtToken, err := GenerateJWT(user.ID, "login", JWTExpiry) 
	if err != nil { return "", &core.CustomMessage{Code: 500, Message: "Failed to generate jwt token"}}

	return jwtToken, nil
}


func (c *Service) Authentication() gin.HandlerFunc{
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

		ctx.Next()
	}
}


// Helper func for checking and updating skills
func (c *Service) CheckAndUpdateSkills(payload []string) ([]*model.Skill, error) {
	skills, err := c.RDB.RetrieveCachedSkills(payload)
	
	if err != nil { 																		// Falling back to the db if the cache fails 
		var existingSkills []*model.Skill

		if err := c.DB.Where("name IN ?", payload).Find(&existingSkills).Error; err != nil{
			return nil, err
		}

		existingSkillSet := make(map[string]bool)
		for _, name := range existingSkills {existingSkillSet[name.Name] = true}

		var newSkill []*model.Skill
		for _, skill := range payload {
			if _, exists := existingSkillSet[skill]; !exists {
				newSkill = append(newSkill, &model.Skill{Name: skill})
			}
		}

		if len(newSkill) > 0 {
			if err := c.DB.Create(&newSkill).Error; err != nil {
				return nil, err
			}
		}
		newSkill = append(newSkill, existingSkills...)
		return newSkill, nil
	}

	var newskills, allskills []*model.Skill
	for _, skill := range payload {
		if id, exists := skills[skill]; exists {
			allskills = append(allskills, &model.Skill{Name: skill, Model: gorm.Model{ID: id}})
			continue
		}
		newskills = append(newskills, &model.Skill{Name: skill})
	}

	if len(newskills) > 0 {
		if err := c.DB.Create(&newskills).Error; err != nil {
			return nil, err
		}
		c.RDB.AddNewSkillToCache(newskills)
	}
	allskills = append(allskills, newskills...)
	return allskills, nil 
} 
