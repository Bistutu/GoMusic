package db

import (
	"GoMusic/internal/usage"
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type usageDaily struct {
	Day            time.Time `gorm:"type:date;primaryKey"`
	Requests       int64     `gorm:"not null;default:0"`
	Successes      int64     `gorm:"not null;default:0"`
	Failures       int64     `gorm:"not null;default:0"`
	PartialResults int64     `gorm:"not null;default:0"`
	Songs          int64     `gorm:"not null;default:0"`
}

func (usageDaily) TableName() string { return "gomusic_usage_daily" }

type usageVisitor struct {
	Day         time.Time `gorm:"type:date;primaryKey"`
	VisitorHash string    `gorm:"type:char(64);primaryKey;index:idx_gomusic_visitor_hash"`
}

func (usageVisitor) TableName() string { return "gomusic_usage_visitors" }

func (s *Store) RecordUsage(ctx context.Context, event usage.Event) error {
	decoded, err := hex.DecodeString(event.VisitorHash)
	if err != nil || len(decoded) != 32 || event.At.IsZero() || event.Songs < 0 || (!event.Succeeded && (event.Songs != 0 || event.Partial)) {
		return fmt.Errorf("invalid usage event")
	}
	row := usageDaily{Day: usage.Day(event.At), Requests: 1, Songs: event.Songs}
	if event.Succeeded {
		row.Successes = 1
	} else {
		row.Failures = 1
	}
	if event.Partial {
		row.PartialResults = 1
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{
			"requests":        gorm.Expr("requests + ?", row.Requests),
			"successes":       gorm.Expr("successes + ?", row.Successes),
			"failures":        gorm.Expr("failures + ?", row.Failures),
			"partial_results": gorm.Expr("partial_results + ?", row.PartialResults),
			"songs":           gorm.Expr("songs + ?", row.Songs),
		}
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "day"}}, DoUpdates: clause.Assignments(updates)}).Create(&row).Error; err != nil {
			return err
		}
		visitor := usageVisitor{Day: row.Day, VisitorHash: event.VisitorHash}
		return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&visitor).Error
	})
}

func (s *Store) UsageStats(ctx context.Context, window usage.Range) (usage.Result, error) {
	result := usage.Result{Period: window.Period, EndDate: window.End.AddDate(0, 0, -1).Format("2006-01-02")}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		filter := func(table string) *gorm.DB {
			query := tx.Table(table).Where("day < ?", window.End)
			if !window.Start.IsZero() {
				query = query.Where("day >= ?", window.Start)
			}
			return query
		}
		var totals struct {
			FirstDay                                             *time.Time
			Requests, Successes, Failures, PartialResults, Songs int64
		}
		if err := filter("gomusic_usage_daily").Select("MIN(day) AS first_day, COALESCE(SUM(requests),0) AS requests, COALESCE(SUM(successes),0) AS successes, COALESCE(SUM(failures),0) AS failures, COALESCE(SUM(partial_results),0) AS partial_results, COALESCE(SUM(songs),0) AS songs").Scan(&totals).Error; err != nil {
			return err
		}
		result.Requests, result.Successes, result.Failures = totals.Requests, totals.Successes, totals.Failures
		result.PartialResults, result.Songs = totals.PartialResults, totals.Songs
		if !window.Start.IsZero() {
			result.StartDate = window.Start.Format("2006-01-02")
		} else if totals.FirstDay != nil {
			result.StartDate = totals.FirstDay.Format("2006-01-02")
		}
		return filter("gomusic_usage_visitors").Distinct("visitor_hash").Count(&result.UniqueVisitors).Error
	})
	return result, err
}
