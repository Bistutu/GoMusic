package models

import "fmt"

// SongList represents a song list entity.
type SongList struct {
	Name       string   `json:"name"` // song list name
	Songs      []string `json:"songs"`
	SongsCount int      `json:"songs_count"`
}

// SongId represents a song ID entity.
type SongId struct {
	Id uint `json:"id"`
}

// String returns the string representation of the SongId.
func (r *SongId) String() string {
	if r == nil {
		return "nil"
	}
	return fmt.Sprintf("{\"id\":%v}", r.Id)
}

// NetEasySongId represents a NetEasy song ID entity.
type NetEasySongId struct {
	Code     int `json:"code"`
	Playlist struct {
		Id         int64      `json:"id"`
		Name       string     `json:"name"`
		TrackIds   []*TrackId `json:"trackIds"`
		TrackCount int        `json:"trackCount"`
	} `json:"playlist"`
}

// TrackId represents a track ID entity.
type TrackId struct {
	Id uint `json:"id"`
}

// Songs represents a songs entity.
type Songs struct {
	Songs []struct {
		Id   uint   `json:"id"`
		Name string `json:"name"`
		Ar   []struct {
			Id   int64  `json:"id"`
			Name string `json:"name"`
		} `json:"ar"`
	} `json:"songs"`
}
