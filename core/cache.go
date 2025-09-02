package core

import (
	"context"
	"encoding/json"
	"findme/model"
	"log"
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


func AddNewSkillToCache(rdb *redis.Client, newskills []*model.Skill) {
	skills := RetrieveCachedSkills(rdb)

	for _, skill := range newskills {
		skills[strings.ToLower(skill.Name)] = skill.ID
	}

	data, _ := json.Marshal(skills)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := rdb.Set(ctx, "skills", data, 0).Result(); err != nil {
		log.Printf("An error occured while trying to set new skill in redis -> %v", err)
	}
}

