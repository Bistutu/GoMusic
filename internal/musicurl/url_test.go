package musicurl

import "testing"

func TestParseMusicLinks(t *testing.T) {
	tests := []struct{ input, provider, kind, id string }{
		{"分享歌单 https://music.163.com/#/playlist?id=8725919816 (@网易云音乐)", NetEase, "playlist", "8725919816"},
		{"https://music.163.com/playlist/2275447155/434174568/?userid=440609461", NetEase, "playlist", "2275447155"},
		{"https://music.163.com/album?id=35150843", NetEase, "album", "35150843"},
		{"https://y.qq.com/n/ryqq_v2/playlist/9280950805?ADTAG=h5", QQ, "playlist", "9280950805"},
		{"https://i.y.qq.com/n2/m/share/details/taoge.html?id=9094523921", QQ, "playlist", "9094523921"},
		{"歌单｜日语经典 https://qishui.douyin.com/s/iAA9MFsL/ @汽水音乐", Qishui, "playlist", ""},
		{"https://music.douyin.com/qishui/share/playlist?playlist_id=7434803282197954611", Qishui, "playlist", "7434803282197954611"},
		{"https://www.kugou.com/songlist/gcid_3z17seve8z2z051/", Kugou, "gcid", "gcid_3z17seve8z2z051"},
		{"https://www.kugou.com/yy/special/single/546903.html", Kugou, "special", "546903"},
		{"https://activity.kugou.com/share/index.html?global_specialid=collection_3_2248574973_2_0", Kugou, "collection", "collection_3_2248574973_2_0"},
		{"https://activity.kugou.com/page/index.html?qrcode=https%3A%2F%2Factivity.kugou.com%2Fshare%2Findex.html%3Fglobal_specialid%3Dcollection_3_2248574973_2_0", Kugou, "collection", "collection_3_2248574973_2_0"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := Parse(tt.input)
			if err != nil {
				t.Fatal(err)
			}
			if got.Provider != tt.provider || got.Kind != tt.kind || got.ID != tt.id {
				t.Fatalf("got %#v", got)
			}
		})
	}
}
func TestRejectUnofficialHosts(t *testing.T) {
	for _, input := range []string{"", "https://qq.evil.test/playlist/1", "https://y.qq.com.evil.test/?id=1", "https://evil.test/?url=https%3A%2F%2Fy.qq.com", "https://y.qq.com@127.0.0.1/?id=1", "https://y.qq.com:9999/?id=1", "http://127.0.0.1/qq/163", "file:///music.163.com", "https://music.163.com/song?id=123", "https://music.163.com/user/home?id=123", "https://i.y.qq.com/v8/playsong.html?id=123", "https://music.163.com/playlist?id=-1"} {
		if _, err := Parse(input); err == nil {
			t.Errorf("accepted %q", input)
		}
	}
}
