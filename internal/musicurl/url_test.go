package musicurl

import (
	"strings"
	"testing"
)

func TestParseMusicLinks(t *testing.T) {
	tests := []struct{ input, provider, kind, id string }{
		{"分享歌单 https://music.163.com/#/playlist?id=8725919816 (@网易云音乐)", NetEase, "playlist", "8725919816"},
		{"https://music.163.com/playlist/2275447155/434174568/?userid=440609461", NetEase, "playlist", "2275447155"},
		{"https://y.music.163.com/m/playlist?app_version=8.10.81&id=8725919816&userid=341246998", NetEase, "playlist", "8725919816"},
		{"https://y.qq.com/n/ryqq_v2/playlist/9280950805?ADTAG=h5", QQ, "playlist", "9280950805"},
		{"https://i.y.qq.com/n2/m/share/details/taoge.html?id=9094523921", QQ, "playlist", "9094523921"},
		{"歌单｜日语经典 https://qishui.douyin.com/s/iAA9MFsL/ @汽水音乐", Qishui, "playlist", ""},
		{"https://music.douyin.com/qishui/share/playlist?playlist_id=7434803282197954611", Qishui, "playlist", "7434803282197954611"},
		{"汽水歌单 https://music.douyin.com/qishui/share/playlist?auto_play_bgm=1&playlist_id=7203983736064622592&hybrid_sdk_version=2，", Qishui, "playlist", "7203983736064622592"},
		{"https://beta-luna.douyin.com/luna/playlist/detail?playlist_id=7434803282197954611", Qishui, "playlist", "7434803282197954611"},
		{"//www.kuwo.cn/playlist_detail/236289985", Kuwo, "playlist", "236289985"},
		{"https://www.kuwo.cn/playlist_detail/3527541442", Kuwo, "playlist", "3527541442"},
		{"http://www.kuwo.cn/playlist/index?pid=3527541442", Kuwo, "playlist", "3527541442"},
		{"酷我歌单 https://m.kuwo.cn/newh5app/playlist_detail/3527541442?t=plantform&from=ar", Kuwo, "playlist", "3527541442"},
		{"https://t.kuwo.cn/AbC123", Kuwo, "playlist", ""},
		{"https://t.kuwo.cn/s/AbC123?from=share", Kuwo, "playlist", ""},
		{"分享歌单 https://www.kuwo.cn/s/AbC123，", Kuwo, "playlist", ""},
		{"https%3A%2F%2Fwww.kuwo.cn%2Fplaylist_detail%2F236289985", Kuwo, "playlist", "236289985"},
		{"歌单有效率100% https%3A%2F%2Fwww.kuwo.cn%2Fplaylist_detail%2F236289985", Kuwo, "playlist", "236289985"},
		{"https%253A%252F%252Fmusic.163.com%252Fplaylist%253Fid%253D8725919816", NetEase, "playlist", "8725919816"},
		{"HTTP://music.migu.cn/v5/#/playlist?playlistId=231760782&playlistType=ordinary", Migu, "playlist", "231760782"},
		{"https://music.migu.cn/v3/music/playlist/12345", Migu, "playlist", "12345"},
		{"https://music.91q.com/songlist/295893", Qianqian, "playlist", "295893"},
		{"https://h5app.kuwo.cn/m/bodian/collection.html?uid=45780003&playlistId=86749638&source=5&owerId=", Bodian, "playlist", "86749638"},
		{"https://www.deezer.com/en/playlist/5339620562", Deezer, "playlist", "5339620562"},
		{"Deezer playlist https://link.deezer.com/s/32zD0iv1H70Dypq0V2AHe", Deezer, "playlist", ""},
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

func TestRejectMalformedKuwoTrackLinks(t *testing.T) {
	for _, input := range []string{
		"https://www.kuwo.cn/play_detail/not-a-number",
		"https://www.kuwo.cn/play_detail/0",
		"https://www.kuwo.cn/yinyue/not-a-number/",
		"https://www.kuwo.cn/yinyue/0/",
	} {
		if _, err := Parse(input); err == nil {
			t.Errorf("accepted %q", input)
		}
	}
}

func TestRejectMalformedKuwoWebRanklistLinks(t *testing.T) {
	for _, input := range []string{
		"https://www.kuwo.cn/bang/content",
		"https://www.kuwo.cn/bang/content?bangId=not-a-number",
		"https://m.kuwo.cn/bang/content?bangId=16",
	} {
		if _, err := Parse(input); err == nil {
			t.Errorf("accepted %q", input)
		}
	}
}

func TestRejectMalformedNetEaseTrackLinks(t *testing.T) {
	for _, input := range []string{
		"https://music.163.com/song",
		"https://music.163.com/#/song?id=not-a-number",
		"https://163cn.tv/song?id=186016",
	} {
		if _, err := Parse(input); err == nil {
			t.Errorf("accepted %q", input)
		}
	}
}

func TestRejectMalformedQQTrackLinks(t *testing.T) {
	for _, input := range []string{
		"https://y.qq.com/n/ryqq/songDetail/not-a-mid",
		"https://i.y.qq.com/n/ryqq/songDetail/0046YC6E3WRI52",
		"https://y.qq.com/n/yqq/song/short.html",
	} {
		if _, err := Parse(input); err == nil {
			t.Errorf("accepted %q", input)
		}
	}
}

func TestParseEscapedShareLinkInput(t *testing.T) {
	tests := []struct{ input, provider, kind, id string }{
		{"https://music.douyin.com/qishui/share/playlist?auto_play_bgm=1&amp;playlist_id=7203983736064622592&amp;hybrid_sdk_version=2", Qishui, "playlist", "7203983736064622592"},
		{"https://i.y.qq.com/n2/m/share/details/taoge.html?ADTAG=h5&amp;id=9094523921", QQ, "playlist", "9094523921"},
		{"https://music.migu.cn/v5/?from=share&amp;playlistId=231760782", Migu, "playlist", "231760782"},
		{"https://h5app.kuwo.cn/m/bodian/collection.html?uid=45780003&amp;playlistId=86749638&amp;source=5", Bodian, "playlist", "86749638"},
		{"https://music.douyin.com/qishui/share/playlist?auto_play_bgm=1&#38;playlist_id=7203983736064622592", Qishui, "playlist", "7203983736064622592"},
		{"https://music.douyin.com/qishui/share/playlist?auto_play_bgm=1&#x26;playlist_id=7203983736064622592", Qishui, "playlist", "7203983736064622592"},
		{"https://music.douyin.com/qishui/share/playlist?auto_play_bgm=1%26amp%3Bplaylist_id=7203983736064622592", Qishui, "playlist", "7203983736064622592"},
		{`{"url":"https:\/\/music.douyin.com\/qishui\/share\/playlist?auto_play_bgm=1\u0026playlist_id=7203983736064622592"}`, Qishui, "playlist", "7203983736064622592"},
		{`{"url":"https:\/\/i.y.qq.com\/n2\/m\/share\/details\/taoge.html?ADTAG=h5\u0026id=9094523921"}`, QQ, "playlist", "9094523921"},
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
	for _, input := range []string{
		"",
		"https://qq.evil.test/playlist/1",
		"https://y.qq.com.evil.test/?id=1",
		"https://y.music.163.com.evil.test/m/playlist?id=1",
		"https://evil.test/?url=https%3A%2F%2Fy.qq.com",
		"https://y.qq.com@127.0.0.1/?id=1",
		"https://y.qq.com:9999/?id=1",
		"http://127.0.0.1/qq/163",
		"file:///music.163.com",
		"https://music.163.com/song?id=not-a-number",
		"https://music.163.com/user/home?id=123",
		"https://i.y.qq.com/v8/playsong.html?id=123",
		"https://music.163.com/playlist?id=-1",
		"https://music.douyin.com/user/home?playlist_id=1",
		"https://beta-luna.douyin.com/untrusted?playlist_id=1",
		"https://qishui.douyin.com/qishui/share/playlist?playlist_id=1",
		"https://music.apple.com/cn/playlist/example/pl.f4d106fed2bd41149aaacabb233eb5eb",
		"https://music.migu.cn/v3/music/song/123",
		"https://music.migu.cn/v3/music/album/not-an-id",
		"https://t.kuwo.cn/",
		"https://t.kuwo.cn/a/b/",
		"https://t.kuwo.cn.evil.test/AbC123",
		"https://kbangserver.kuwo.cn/ksong.s?id=16",
		"https://deezer.com.evil.test/playlist/5339620562",
		"https://api.deezer.com/playlist/5339620562",
		"https://www.deezer.com/en/artist/not-a-playlist",
		"https://www.deezer.com/track/not-a-number",
		"https://link.deezer.com.evil.test/s/32zD0iv1H70Dypq0V2AHe",
		"https://link.deezer.com/track/3135556",
		"https://link.deezer.com/playlist/123",
		"https://link.deezer.com/s/",
		"https%3A%2F%2Fevil.test%2Fplaylist%2F1",
	} {
		if _, err := Parse(input); err == nil {
			t.Errorf("accepted %q", input)
		}
	}
}

func TestRetiredDeezerShareLinksExplainRecovery(t *testing.T) {
	for _, input := range []string{
		"https://deezer.page.link/kHcRMSzZEddVF4dd9",
		"https://dzr.page.link/abc123",
	} {
		_, err := Parse(input)
		if err == nil || !strings.Contains(err.Error(), "已停止跳转") || !strings.Contains(err.Error(), "link.deezer.com/s/") {
			t.Errorf("expected a recovery hint for retired Deezer link %q, got %v", input, err)
		}
	}
}
