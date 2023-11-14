package db

import (
	//_ "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"GoMusic/common/models"
	"GoMusic/initialize/log"
)

var db *gorm.DB

func init() {
	dsn := "root:12345678@tcp(127.0.0.1:3306)/music?charset=utf8mb4&parseTime=True&loc=Local"
	open, err := gorm.Open(mysql.Open(dsn))
	if err != nil {
		log.Errorf("数据库连接失败：", err)
		panic(err)
	}
	db = open
	// 自动创建表
	db.AutoMigrate(&models.NetEasySongs{})
}
