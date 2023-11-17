package db

import (
	"context"

	//_ "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"GoMusic/common/models"
	"GoMusic/initialize/log"
)

var db *gorm.DB

func init() {
	dsn := "root:12345678@tcp(127.0.0.1:3306)/go_music?charset=utf8mb4&parseTime=True&loc=Local"
	open, err := gorm.Open(mysql.Open(dsn))
	if err != nil {
		log.Errorf("数据库连接失败：%v", err)
		panic(err)
	}
	db = open
	// 自动创建表
	db.AutoMigrate(&models.NetEasySong{})
}

func BatchGetSongById(ctx context.Context, ids []int) ([]*models.NetEasySong, error) {
	var netEasySongs []*models.NetEasySong
	err := db.Where("id in ?", ids).Find(&netEasySongs).Error
	if err != nil {
		log.Errorf("查询数据库失败：%v", err)
		return nil, err
	}
	return netEasySongs, nil
}

func BatchInsertSong(ctx context.Context, netEasySongs []*models.NetEasySong) error {
	// 如果 Duplicate primary key 则执行 update 操作
	err := db.Clauses(clause.OnConflict{
		UpdateAll: true,
	}).CreateInBatches(netEasySongs, 500).Error
	if err != nil {
		log.Errorf("数据库插入失败：%v", err)
	}
	return err
}

func BatchDelSong(ctx context.Context, ids []int) error {
	err := db.Delete(&models.NetEasySong{}, ids).Error
	if err != nil {
		log.Errorf("数据库删除数据失败：%v", err)
	}
	return err
}
