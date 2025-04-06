package models

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Courses struct {
	gorm.Model
	ID     int 	  
	Name   string `gorm:"type:text"`
}

func CoursesAll(ctx *gin.Context) *[]Courses {
	var course []Courses
	DB.Where("deleted_at is NULL").Find(&course)
	return &course
}

func CoursesFind(id uint64) *Courses {
	var course Courses
	DB.Where("id = ?", id).First(&course)
	return &course
}

func CreateCourse(ID int, Name string){
	course := Courses{
		ID: ID,
		Name: Name,
	}

	DB.Create(course)
}