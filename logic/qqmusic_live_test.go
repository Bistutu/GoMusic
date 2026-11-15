package logic

import "testing"

// TestQQMusicLivePlaylists 验证有效分享短链和用户的 ryqq_v2 歌单链接。
func TestQQMusicLivePlaylists(t *testing.T) {
	tests := []struct {
		name     string
		link     string
		minSongs int
	}{
		{
			name:     "share short link falls back to legacy API",
			link:     "https://c6.y.qq.com/base/fcgi-bin/u?__=GNSf48IjarL6",
			minSongs: 31,
		},
		{
			name:     "ryqq_v2 playlist with query parameters",
			link:     "https://y.qq.com/n/ryqq_v2/playlist/9280950805?ADTAG=h5_share_playlist&redirecttag=mn.redirect.custom&mnst=0.82",
			minSongs: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			songList, err := QQMusicDiscover(tt.link, true)
			if err != nil {
				t.Fatalf("QQMusicDiscover() error = %v", err)
			}
			if songList == nil {
				t.Fatal("QQMusicDiscover() returned nil song list")
			}
			t.Logf("name=%q, reported=%d, parsed=%d", songList.Name, songList.SongsCount, len(songList.Songs))
			if len(songList.Songs) < tt.minSongs {
				t.Fatalf("expected at least %d songs, got %d", tt.minSongs, len(songList.Songs))
			}
			if songList.SongsCount != len(songList.Songs) {
				t.Fatalf("reported count %d does not match parsed count %d", songList.SongsCount, len(songList.Songs))
			}
		})
	}
}
