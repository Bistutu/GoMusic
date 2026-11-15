package logic

import (
	"strings"
	"testing"
)

func TestExtractPlaylistID(t *testing.T) {
	tests := []struct {
		name string
		link string
		want int
	}{
		{
			name: "ryqq_v2 with query parameters",
			link: "https://y.qq.com/n/ryqq_v2/playlist/9280950805?ADTAG=h5_share_playlist&mnst=0.82",
			want: 9280950805,
		},
		{
			name: "standard playlist",
			link: "https://y.qq.com/n/ryqq/playlist/7364061065",
			want: 7364061065,
		},
		{
			name: "details page id parameter",
			link: "https://i.y.qq.com/n2/m/share/details/taoge.html?id=9094523921",
			want: 9094523921,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractPlaylistID(tt.link)
			if err != nil {
				t.Fatalf("extractPlaylistID() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("extractPlaylistID() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestParseLegacyPlaylist(t *testing.T) {
	body := `{
		"code": 0,
		"subcode": 0,
		"cdlist": [{
			"dissname": "测试歌单",
			"songnum": 2,
			"songlist": [
				{"songname": "第一首 (Live)", "singer": [{"name": "歌手甲"}, {"name": "歌手乙"}]},
				{"songname": "第二首", "singer": [{"name": "歌手丙"}]}
			]
		}]
	}`

	songList, err := parseLegacyPlaylist(strings.NewReader(body), true)
	if err != nil {
		t.Fatalf("parseLegacyPlaylist() error = %v", err)
	}
	if songList.Name != "测试歌单" {
		t.Fatalf("name = %q, want %q", songList.Name, "测试歌单")
	}
	if songList.SongsCount != 2 || len(songList.Songs) != 2 {
		t.Fatalf("count = %d, len(songs) = %d, want 2", songList.SongsCount, len(songList.Songs))
	}
	want := "第一首 (Live) - 歌手甲 / 歌手乙"
	if songList.Songs[0] != want {
		t.Fatalf("songs[0] = %q, want %q", songList.Songs[0], want)
	}
}

func TestParseLegacyPlaylistBusinessError(t *testing.T) {
	body := `{"code":0,"subcode":4000,"msg":"歌单不存在"}`
	_, err := parseLegacyPlaylist(strings.NewReader(body), false)
	if err == nil {
		t.Fatal("parseLegacyPlaylist() expected an error")
	}
	if !strings.Contains(err.Error(), "subcode:4000") {
		t.Fatalf("error = %q, want subcode detail", err)
	}
}
