// Package database -> Connection to database and redis
package database

import (
	"fmt"
	"log"
	"os"

	"findme/model"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Connect -> Connection to database
func Connect() *gorm.DB {
	dsn := fmt.Sprintf(
		"host=localhost user=%s password=%s dbname=%s port=5432 sslmode=disable",
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_DB"),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("[ERROR] [DB] Failed to establish database connection -> %s", err.Error())
	}

	err = db.AutoMigrate(
		&model.User{},
		&model.Skill{},
		&model.Post{},
		&model.PostSkill{},
		&model.UserSkill{},
		&model.UserFriend{},
		&model.FriendReq{},
		&model.UserMessage{},
	)
	if err != nil {
		log.Fatalf("[ERROR] [DB] Failed to create tables -> %s", err.Error())
	}

	err = db.SetupJoinTable(&model.Post{}, "Tags", &model.PostSkill{})
	if err != nil {
		log.Fatalf("[ERROR] [DB] Failed to create join table on post and skills -> %s", err.Error())
	}

	err = db.SetupJoinTable(&model.User{}, "Skills", &model.UserSkill{})
	if err != nil {
		log.Fatalf("[ERROR] [DB] Failed to create join table on user and skills -> %s", err.Error())
	}

	err = db.SetupJoinTable(&model.User{}, "Friends", &model.UserFriend{})
	if err != nil {
		log.Fatalf("[ERROR] [DB] Failed to create join table on user and friends -> %s", err.Error())
	}

	log.Println("[INFO] [DB] Connected to the database successfully.")
	return db
}
