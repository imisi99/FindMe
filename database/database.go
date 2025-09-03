package database

import (
	"findme/model"
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)


var dbClient *gorm.DB

// Connection to database
func Connect() {

	// Get database connection details from environment variables
	dsn := fmt.Sprintf(
		"host=localhost user=%s password=%s dbname=%s port=5432 sslmode=disable",
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_DB"),
	)


	// Connect to the database
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil{
		log.Fatalf("[ERROR] Failed to establish database connection -> %s", err.Error())
	}


	err = db.AutoMigrate(&model.User{}, &model.Skill{}, &model.Post{}, &model.PostSkill{}, &model.UserSkill{})
	if err != nil {
		log.Fatalf("[ERROR] Failed to create tables -> %s", err.Error())
	}

	err = db.SetupJoinTable(&model.Post{}, "Tags", &model.PostSkill{})
	if err != nil {
		log.Fatalf("[ERROR] Failed to create join table on post and skills -> %s", err.Error())
	}

	err = db.SetupJoinTable(&model.User{}, "Skills", &model.UserSkill{})
	if err != nil {
		log.Fatalf("[ERROR] Failed to create join table on user and skills -> %s", err.Error())
	}

	dbClient = db

	log.Println("[INFO] Connected to the database successfully.")

}

// Returns database connection session
func GetDB() *gorm.DB {
	return dbClient
}


// Set database connection for test
func SetDB(client *gorm.DB) {
	dbClient = client
}
