// Package core -> Core Functionalities of the app
package core

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"findme/model"

	"github.com/redis/go-redis/v9"
)

type Cache interface {
	CheckHealth() error
	CacheSkills(skills []model.Skill)
	RetrieveCachedSkills(skills []string) (map[string]string, error)
	AddNewSkillToCache(skill []*model.Skill)
	SetOTP(otp string, uid string) error
	GetOTP(otp string) (string, error)
}

type RDB struct {
	Cache *redis.Client
}

func NewRDB(rdb *redis.Client) *RDB {
	return &RDB{Cache: rdb}
}

func (c *RDB) CheckHealth() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := c.Cache.Ping(ctx).Result()
	return err
}

// CacheSkills -> Cache skills in rdb at app startup
func (c *RDB) CacheSkills(skills []model.Skill) {
	skillName := make(map[string]any, 0)

	for _, skill := range skills {
		skillName[skill.Name] = skill.ID
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := c.Cache.HSet(ctx, "skills", skillName).Result(); err != nil {
		log.Printf("[ERROR] [RDB] An error occured while trying to set skills in redis -> %v", err)
	}
}

// RetrieveCachedSkills -> Retrieve cached skills from rdb if possible
func (c *RDB) RetrieveCachedSkills(skills []string) (map[string]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	skill, err := c.Cache.HMGet(ctx, "skills", skills...).Result()
	if err != nil {
		log.Printf("[ERROR] [RDB] Failed to receive skills from redis -> %s", err)
		return nil, err
	}

	foundskills := make(map[string]string, 0)

	for i, val := range skill {
		if val == nil {
			continue
		}
		id := val.(string)
		foundskills[skills[i]] = id
	}
	return foundskills, nil
}

// AddNewSkillToCache -> Add new skills to rdb
func (c *RDB) AddNewSkillToCache(newskills []*model.Skill) {
	skills := make(map[string]any, 0)

	for _, skill := range newskills {
		skills[skill.Name] = skill.ID
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := c.Cache.HSet(ctx, "skills", skills).Result(); err != nil {
		log.Printf("An error occured while trying to set new skill in redis -> %v", err)
	}
}

// SetOTP -> Set OTP for password reset temporary in rdb
func (c *RDB) SetOTP(otp string, userID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := c.Cache.HGet(ctx, "otps", otp).Result(); err != redis.Nil {
		return &CustomMessage{http.StatusConflict, "Token already exists."}
	}

	otps := &redis.HSetEXOptions{
		ExpirationType: redis.HSetEXExpirationEX,
		ExpirationVal:  600,
	}

	if _, err := c.Cache.HSetEXWithArgs(ctx, "otps", otps, otp, userID).Result(); err != nil {
		log.Printf("An error occured while trying to set otp in redis -> %v", err)
		return &CustomMessage{http.StatusInternalServerError, "Failed to set otp."}
	}
	return nil
}

// GetOTP -> Verify if OTP provided exists in rdb and returns the userID
func (c *RDB) GetOTP(otp string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	userID, err := c.Cache.HGetDel(ctx, "otps", otp).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", &CustomMessage{http.StatusNotFound, "Invalid otp."}
		} else {
			return "", &CustomMessage{http.StatusInternalServerError, "Failed to verify otp."}
		}
	}
	if len(userID) != 1 || userID[0] == "" {
		return "", &CustomMessage{http.StatusNotFound, "Invalid otp."}
	}
	return userID[0], nil
}
