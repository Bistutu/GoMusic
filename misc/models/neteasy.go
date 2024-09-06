package models

import "fmt"

type SongList struct {
	Name       string   `json:"name"` // 歌单名称
	Songs      []string `json:"songs"`
	SongsCount int      `json:"songs_count"`
}

type SongId struct {
	Id uint `json:"id"`
}

func (r *SongId) String() string {
	if r == nil {
		return "nil"
	}
	return fmt.Sprintf("{\"id\":%v}", r.Id)
}

type NetEasySongId struct {
	Code     int `json:"code"`
	Playlist struct {
		Id         int64      `json:"id"`
		Name       string     `json:"name"`
		TrackIds   []*TrackId `json:"trackIds"`
		TrackCount int        `json:"trackCount"`
	} `json:"playlist"`
}

type TrackId struct {
	Id uint `json:"id"`
}

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
