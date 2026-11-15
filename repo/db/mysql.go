package db

import (
	"GoMusic/misc/models"
	"context"
	"encoding/json"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"time"
)

type Store struct {
	db  *gorm.DB
	ttl time.Duration
}

func Open(ctx context.Context, dsn string, ttl time.Duration) (*Store, error) {
	handle, err := gorm.Open(mysql.New(mysql.Config{DSN: dsn, SkipInitializeWithVersion: true}), &gorm.Config{DisableAutomaticPing: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return nil, err
	}
	sqlDB, err := handle.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	if err = sqlDB.PingContext(ctx); err != nil {
		sqlDB.Close()
		return nil, err
	}
	if err = handle.WithContext(ctx).AutoMigrate(&models.SongMetadata{}); err != nil {
		sqlDB.Close()
		return nil, err
	}
	return &Store{db: handle, ttl: ttl}, nil
}
func (s *Store) Get(ctx context.Context, provider string, ids []string) (map[string]models.Track, error) {
	found := make(map[string]models.Track, len(ids))
	if len(ids) == 0 {
		return found, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	var rows []models.SongMetadata
	query := s.db.WithContext(ctx).Where("provider = ? AND id IN ?", provider, ids)
	if s.ttl > 0 {
		query = query.Where("updated_at >= ?", time.Now().Add(-s.ttl))
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		track := models.Track{ID: row.ID, Title: row.Title}
		if json.Unmarshal([]byte(row.Artists), &track.Artists) == nil && track.Title != "" {
			found[row.ID] = track
		}
	}
	return found, nil
}
func (s *Store) Put(ctx context.Context, provider string, tracks []models.Track) error {
	if len(tracks) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	rows := make([]models.SongMetadata, 0, len(tracks))
	for _, track := range tracks {
		artists, err := json.Marshal(track.Artists)
		if err != nil {
			return err
		}
		rows = append(rows, models.SongMetadata{Provider: provider, ID: track.ID, Title: track.Title, Artists: string(artists), UpdatedAt: time.Now()})
	}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "provider"}, {Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"title", "artists", "updated_at"})}).CreateInBatches(rows, 400).Error
}
func (s *Store) Ping(ctx context.Context) error {
	handle, err := s.db.DB()
	if err != nil {
		return err
	}
	return handle.PingContext(ctx)
}
func (s *Store) Close() error {
	handle, err := s.db.DB()
	if err != nil {
		return err
	}
	return handle.Close()
}
