package database

import (
	"findme/model"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)


var DB *gorm.DB

// Connection to database
func Connect() {

	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("[WARNING] Error loading .env file ->", err, "Ignore if in production")
	}

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


	err = db.AutoMigrate(&model.User{}, &model.Skill{})
	if err != nil {
		log.Fatalf("[ERROR] Failed to create tables -> %s", err.Error())
	}

	DB = db

	log.Println("[INFO] Connected to the database successfully.")

}

// Returns database connection session
func GetDB() *gorm.DB {
	return DB
}
