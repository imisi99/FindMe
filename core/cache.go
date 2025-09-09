package core

import (
	"context"
	"encoding/json"
	"findme/model"
	"findme/schema"
	"fmt"
	"log"
	"strconv"
	"strings"

	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func CacheSkills(db *gorm.DB, rdb *redis.Client) {
	var skills []model.Skill
	if err := db.Find(&skills).Error; err != nil {
		log.Fatalf("An error occured while fetching skills from db -> %v", err)
	}

	skillName := make(map[string]string, 0)

	for _, skill := range skills {
		skillName[skill.Name] = fmt.Sprintf("%d", skill.ID)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := rdb.HSet(ctx, "skills", skillName).Result(); err != nil {
		log.Printf("An error occured while trying to set skills in redis -> %v", err)
	}
}	


func RetrieveCachedSkills(rdb *redis.Client, skills []string) (map[string]uint, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	skill, err := rdb.HMGet(ctx, "skills", skills...).Result()
	if err != nil {
		log.Printf("Failed to receive skills from redis-> %s", err)
		return nil, err
	}

	foundskills := make(map[string]uint, 0)

	for i, val := range skill {
		if val == nil {continue}
		idstr := val.(string)
		id, _ := strconv.ParseUint(idstr, 10, 64)
		foundskills[skills[i]] = uint(id)
	}
	return foundskills, nil
}


func AddNewSkillToCache(rdb *redis.Client, newskills []*model.Skill) {
	skills := make(map[string]string, 0)
	for _, skill := range newskills {
		skills[strings.ToLower(skill.Name)] = fmt.Sprintf("%d", skill.ID)
	}
 
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := rdb.HSet(ctx, "skills", skills).Result(); err != nil {
		log.Printf("An error occured while trying to set new skill in redis -> %v", err)
	}
}


func SetOTP(rdb *redis.Client, otp string, userID uint) error {

	otpInfo := schema.OTPInfo{UserID: userID}
	data, _ := json.Marshal(otpInfo)

	ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
	defer cancel()

	if _, err := rdb.Get(ctx, otp).Result(); err != redis.Nil {
		return &CustomMessage{Detail: "Token already exists."}
	}
	if _, err := rdb.Set(ctx, otp, data, 10*time.Minute).Result(); err != nil {
		log.Printf("An error occured while trying to set otp in redis -> %v", err)
		return err
	}
	return nil
}


func GetOTP(rdb *redis.Client, otp string) (*schema.OTPInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
	defer cancel()

	val, err := rdb.Get(ctx, otp).Result()
	if err != nil {return nil, redis.Nil}

	var otpInfo schema.OTPInfo
	if err := json.Unmarshal([]byte(val), &otpInfo); err != nil {
		return nil, err
	}

	return &otpInfo, nil
}
