package models

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Courses struct {
	gorm.Model
	Title   string `gorm:"size:255"`
	Content string `gorm:"type:text"`
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

func CreateCourse(title, content string) (*Courses, error) {
	course := Courses{
		Title:   title,
		Content: content,
	}
	if err := DB.Create(&course).Error; err != nil {
		return nil, err
	}

	return &course, nil
}