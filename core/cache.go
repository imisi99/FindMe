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


type Cache interface {
	CacheSkills()
	RetrieveCachedSkills(skills []string) (map[string]uint, error)
	AddNewSkillToCache(skill []*model.Skill)
	SetOTP(otp string, uid uint) error
	GetOTP(otp string, otpInfo *schema.OTPInfo) error 
}


type RDB struct {
	Cache	*redis.Client
	DB		*gorm.DB
}


func NewRDB(rdb *redis.Client, db *gorm.DB) *RDB {
	return &RDB{Cache: rdb, DB: db}
}

func (c *RDB) CacheSkills() {
	var skills []model.Skill
	if err := c.DB.Find(&skills).Error; err != nil {
		log.Fatalf("An error occured while fetching skills from db -> %v", err)
	}

	skillName := make(map[string]string, 0)

	for _, skill := range skills {
		skillName[skill.Name] = fmt.Sprintf("%d", skill.ID)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := c.Cache.HSet(ctx, "skills", skillName).Result(); err != nil {
		log.Printf("An error occured while trying to set skills in redis -> %v", err)
	}
}	


func (c *RDB) RetrieveCachedSkills(skills []string) (map[string]uint, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	skill, err := c.Cache.HMGet(ctx, "skills", skills...).Result()
	if err != nil {
		log.Printf("Failed to receive skills from redis-> %s", err)
		return nil, err
	}

	foundskills := make(map[string]uint, 0)

	for i, val := range skill {
		if val == nil {continue}
		id, _ := strconv.ParseUint(val.(string), 10, 64)
		foundskills[skills[i]] = uint(id)
	}
	return foundskills, nil
}


func (c *RDB) AddNewSkillToCache(newskills []*model.Skill) {
	skills := make(map[string]string, 0)
	for _, skill := range newskills {
		skills[strings.ToLower(skill.Name)] = fmt.Sprintf("%d", skill.ID)
	}
 
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := c.Cache.HSet(ctx, "skills", skills).Result(); err != nil {
		log.Printf("An error occured while trying to set new skill in redis -> %v", err)
	}
}


func (c *RDB) SetOTP(otp string, userID uint) error {
	otpInfo := schema.OTPInfo{UserID: userID}
	data, _ := json.Marshal(otpInfo)

	ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
	defer cancel()

	if _, err := c.Cache.Get(ctx, otp).Result(); err != redis.Nil {
		return &CustomMessage{Message: "Token already exists."}
	}
	if _, err := c.Cache.Set(ctx, otp, data, 10*time.Minute).Result(); err != nil {
		log.Printf("An error occured while trying to set otp in redis -> %v", err)
		return err
	}
	return nil
}


func (c *RDB) GetOTP(otp string, otpInfo *schema.OTPInfo) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
	defer cancel()

	val, err := c.Cache.Get(ctx, otp).Result()
	if err != nil {return err}

	json.Unmarshal([]byte(val), otpInfo)

	return nil
}
