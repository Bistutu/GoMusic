package models

import "time"

// SongMetadata is a versioned raw metadata cache. Legacy net_easy_songs remains untouched.
// Distinct tracks can share a title; only (provider, id) is unique.
type SongMetadata struct {
	Provider  string    `gorm:"primaryKey;size:16"`
	ID        string    `gorm:"primaryKey;size:32"`
	Title     string    `gorm:"type:text;not null"`
	Artists   string    `gorm:"type:text;not null"`
	UpdatedAt time.Time `gorm:"index"`
}
