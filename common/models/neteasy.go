package models

import "fmt"

type SongList struct {
	// 歌单名
	Name  string   `json:"name"`
	Songs []string `json:"songs"`
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
		Id       int64  `json:"id"`
		Name     string `json:"name"`
		TrackIds []struct {
			Id uint `json:"id"`
		} `json:"trackIds"`
	} `json:"playlist"`
}

type Songs struct {
	Songs []struct {
		Name string `json:"name"`
		Ar   []struct {
			Name string `json:"name"`
		} `json:"ar"`
	} `json:"songs"`
}
