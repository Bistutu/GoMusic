package models

import "gorm.io/gorm"

// NetEasySong NetEasy song, the table will be created automatically if successful run the program
type NetEasySong struct {
	gorm.Model
	Id    uint   `gorm:"column:id"`
	Name  string `gorm:"column:name;type:varchar(512);unique:true"`
	Exist byte   `gorm:"column:exist;default:1"`
}
