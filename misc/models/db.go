package models

import "time"

// NetEasySong NetEasy song, the table will be created automatically if successful run the program
type NetEasySong struct {
	Id        uint      `gorm:"column:id;primarykey"`
	Name      string    `gorm:"column:name;type:varchar(512);unique:true"`
	CreatedAt time.Time `gorm:"column:created_at"`
}
