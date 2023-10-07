package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"

	"GoMusic/httputil"
	"GoMusic/models"
)

func main() {
	id := "8725919816"
	res, err := httputil.Post("https://music.163.com/api/v6/playlist/detail", strings.NewReader("id="+id))
	if err != nil {
		log.Fatalf("fail to post: %v", err)
		return
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatalf("fail to read res body: %v", err)
		return
	}
	netEasySongId := &models.NetEasySongId{}
	json.Unmarshal(body, &netEasySongId)
	SongsName := netEasySongId.Playlist.Name            // 歌单名称
	trackIds := netEasySongId.Playlist.TrackIds         // 歌单歌曲 ID 列表
	songsId := make([]*models.SongId, 0, len(trackIds)) // 歌曲 ID To []Uint
	for _, v := range trackIds {
		songsId = append(songsId, &models.SongId{Id: v.Id})
	}
	fmt.Println(SongsName)
	//fmt.Println(songsId)
	marshal, _ := json.Marshal(songsId)

	reader := strings.NewReader("c=" + string(marshal))
	post, err := httputil.Post("https://music.163.com/api/v3/song/detail", reader)
	if err != nil {
		log.Fatalf("fail to post: %v", err)
		return
	}
	defer post.Body.Close()
	bytes, _ := io.ReadAll(post.Body)
	songs := &models.Songs{}
	err = json.Unmarshal(bytes, &songs)
	if err != nil {
		log.Fatalf("fail to unmarshal: %v", err)
		return
	}
	songsString := make([]string, 0, len(songs.Songs))
	for _, v := range songs.Songs {
		builder := strings.Builder{}
		builder.WriteString(v.Name)
		builder.WriteString(" - ")

		authors := make([]string, 0, len(v.Ar))
		for _, v := range v.Ar {
			authors = append(authors, v.Name)
		}
		authorsString := strings.Join(authors, " / ")
		builder.WriteString(authorsString)
		songsString = append(songsString, builder.String())
	}
	for k, v := range songsString {
		fmt.Println(k, v)
	}
}
