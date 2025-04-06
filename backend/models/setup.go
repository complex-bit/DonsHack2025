package models

import (
	"os"
	"log"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB
func ConnectDatabase() {
	dsn := os.Getenv("DB_CONN_STR") 
	if dsn == "" {
		dsn = "root:password@tcp(127.0.0.1:3306)/usersdb" 
	}
	database, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	DB = database
}

func DBMigrate() {
	err := DB.AutoMigrate(&Courses{}, &User{})
	if err != nil {
		log.Fatalf("Error running migration: %v", err)
	}
}
