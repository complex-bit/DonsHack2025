package models

import (

	"github.com/jinzhu/gorm"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	gorm.Model
	Email     string `gorm:"size:64, index"`
	Password  string `gorm:"size:255"`
	Canvas    string `gorm:"size:255"`
}

func CheckUser(email string) bool {
	var user User
	DB.Where("email = ?", email).First(&user)
	return user.ID == 0
}

func UserCreate(email string, password string, canvas string) *User {

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	if err != nil {
		return nil
	}

	entry := User{Email: email, Password: string(hashedPassword), Canvas: canvas}
	DB.Create(&entry)
	return &entry
}

func UserMatch(email string) (*User, string) {
	var user User
	var canvas string
	DB.Table("users").Select("canvas").Where("id = ?", 11).Scan(&canvas)
	return &user, canvas
}


func UserFromId(id uint) *User {
	var user User
	DB.Where("id = ?", id).First(&user)
	return &user
}

func UserMatchPassword(email string, password string) *User {
	var user User
	DB.Where("email = ?", email).First(&user)

	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return &User{}
	} else {
		return &user
	}
}
