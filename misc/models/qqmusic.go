package models

import "encoding/json"

type QQMusicParam struct {
	Disstid    int    `json:"disstid"`
	EncHostUin string `json:"enc_host_uin"`
	Tag        int    `json:"tag"`
	Userinfo   int    `json:"userinfo"`
	SongBegin  int    `json:"song_begin"`
	SongNum    int    `json:"song_num"`
}
type QQMusicReq struct {
	Req0 struct {
		Module string       `json:"module"`
		Method string       `json:"method"`
		Param  QQMusicParam `json:"param"`
	} `json:"req_0"`
	Comm struct {
		GTk      int    `json:"g_tk"`
		Uin      int    `json:"uin"`
		Format   string `json:"format"`
		Platform string `json:"platform"`
	} `json:"comm"`
}

func NewQQMusicReq(id int, platform string, begin, count int) *QQMusicReq {
	r := &QQMusicReq{}
	r.Req0.Module = "music.srfDissInfo.aiDissInfo"
	r.Req0.Method = "uniform_get_Dissinfo"
	r.Req0.Param = QQMusicParam{Disstid: id, Tag: 1, Userinfo: 1, SongBegin: begin, SongNum: count}
	r.Comm.GTk = 5381
	r.Comm.Format = "json"
	r.Comm.Platform = platform
	return r
}
func GetQQMusicReqStringWithPagination(id int, platform string, begin, count int) string {
	data, _ := json.Marshal(NewQQMusicReq(id, platform, begin, count))
	return string(data)
}

type QQSong struct {
	ID     json.Number `json:"id"`
	Mid    string      `json:"mid"`
	Name   string      `json:"name"`
	Singer []struct {
		Name string `json:"name"`
	} `json:"singer"`
}
type QQMusicResp struct {
	Code int `json:"code"`
	Req0 *struct {
		Code int `json:"code"`
		Data *struct {
			Dirinfo struct {
				Title   string `json:"title"`
				Songnum int    `json:"songnum"`
			} `json:"dirinfo"`
			Songlist []QQSong `json:"songlist"`
		} `json:"data"`
	} `json:"req_0"`
}
type QQMusicLegacyResp struct {
	Code    int                 `json:"code"`
	Subcode int                 `json:"subcode"`
	Message string              `json:"msg"`
	Cdlist  []QQMusicLegacyList `json:"cdlist"`
	Data    *struct {
		Cdlist []QQMusicLegacyList `json:"cdlist"`
	} `json:"data"`
}

type QQMusicLegacyList struct {
	Dissname string `json:"dissname"`
	Songnum  int    `json:"songnum"`
	Songlist []struct {
		Songid   json.Number `json:"songid"`
		Songmid  string      `json:"songmid"`
		Songname string      `json:"songname"`
		Songorig string      `json:"songorig"`
		Singer   []struct {
			Name string `json:"name"`
		} `json:"singer"`
	} `json:"songlist"`
}
