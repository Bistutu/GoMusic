package db

import (
	_ "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"GoMusic/misc/log"
	"GoMusic/misc/models"
)

var db *gorm.DB

func init() {
	dsn := "go_music:12345678@tcp(127.0.0.1:3306)/go_music?charset=utf8mb4&parseTime=True&loc=Local"
	open, err := gorm.Open(mysql.Open(dsn))
	if err != nil {
		log.Errorf("数据库连接失败：%v", err)
		panic(err)
	}
	db = open
	// 自动创建表
	db.AutoMigrate(&models.NetEasySong{})

	// 调用自定义迁移函数修改表结构
	if err := MigrateNameField(db); err != nil {
		log.Errorf("failed to migrate database: %v", err)
	}
}

func MigrateNameField(db *gorm.DB) error {
	// 使用原生 SQL 来修改字段长度
	return db.Exec("ALTER TABLE net_easy_songs MODIFY name VARCHAR(512);").Error
}

func BatchGetSongById(ids []uint) (map[uint]string, error) {
	var netEasySongs []*models.NetEasySong
	err := db.Where("id in ?", ids).Find(&netEasySongs).Error
	if err != nil {
		log.Errorf("查询数据库失败：%v", err)
		return nil, err
	}
	// 歌曲id:歌曲信息
	netEasySongMap := make(map[uint]string)
	for _, v := range netEasySongs {
		netEasySongMap[v.Id] = v.Name
	}
	return netEasySongMap, nil
}

func BatchInsertSong(netEasySongs []*models.NetEasySong) error {
	// 如果 Duplicate primary key 则执行 update 操作
	err := db.Clauses(clause.OnConflict{
		UpdateAll: true,
	}).CreateInBatches(netEasySongs, 500).Error
	if err != nil {
		log.Errorf("数据库插入失败：%v", err)
	}
	return err
}

func BatchDelSong(ids []int) error {
	err := db.Delete(&models.NetEasySong{}, ids).Error
	if err != nil {
		log.Errorf("数据库删除数据失败：%v", err)
	}
	return err
}
