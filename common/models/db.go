package models

import "gorm.io/gorm"

type NetEasySong struct {
	gorm.Model
	Id    int    `gorm:"column:id"`
	Name  string `gorm:"column:name;type:varchar(255);unique:true"`
	Exist byte   `gorm:"column:exist;default:1"`
}
