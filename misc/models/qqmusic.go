package models

import "encoding/json"

type QQMusicReq struct {
	Req0 struct {
		Module string `json:"module"`
		Method string `json:"method"`
		Param  struct {
			Disstid    int    `json:"disstid"`
			EncHostUin string `json:"enc_host_uin"`
			Tag        int    `json:"tag"`
			Userinfo   int    `json:"userinfo"`
			SongBegin  int    `json:"song_begin"`
			SongNum    int    `json:"song_num"`
		} `json:"param"`
	} `json:"req_0"`
	Comm struct {
		GTk      int    `json:"g_tk"`
		Uin      int    `json:"uin"`
		Format   string `json:"format"`
		Platform string `json:"platform"`
	} `json:"comm"`
}

func NewQQMusicReq(disstid int, platform string, songBegin, songNum int) *QQMusicReq {
	return &QQMusicReq{
		Req0: struct {
			Module string `json:"module"`
			Method string `json:"method"`
			Param  struct {
				Disstid    int    `json:"disstid"`
				EncHostUin string `json:"enc_host_uin"`
				Tag        int    `json:"tag"`
				Userinfo   int    `json:"userinfo"`
				SongBegin  int    `json:"song_begin"`
				SongNum    int    `json:"song_num"`
			} `json:"param"`
		}{
			Module: "music.srfDissInfo.aiDissInfo",
			Method: "uniform_get_Dissinfo",
			Param: struct {
				Disstid    int    `json:"disstid"`
				EncHostUin string `json:"enc_host_uin"`
				Tag        int    `json:"tag"`
				Userinfo   int    `json:"userinfo"`
				SongBegin  int    `json:"song_begin"`
				SongNum    int    `json:"song_num"`
			}{
				Disstid:    disstid,
				EncHostUin: "",
				Tag:        1,
				Userinfo:   1,
				SongBegin:  songBegin,
				SongNum:    songNum,
			},
		},
		Comm: struct {
			GTk      int    `json:"g_tk"`
			Uin      int    `json:"uin"`
			Format   string `json:"format"`
			Platform string `json:"platform"`
		}{
			GTk:      5381,
			Uin:      0,
			Format:   "json",
			Platform: platform,
		},
	}
}

// GetQQMusicReqStringWithPagination 获取带分页参数的请求字符串
func GetQQMusicReqStringWithPagination(disstid int, platform string, songBegin, songNum int) string {
	param := NewQQMusicReq(disstid, platform, songBegin, songNum)
	marshal, _ := json.Marshal(param)
	return string(marshal)
}

type QQMusicResp struct {
	Code int `json:"code"`
	Req0 struct {
		Code int `json:"code"`
		Data struct {
			Dirinfo struct {
				Title   string `json:"title"`
				Songnum int    `json:"songnum"`
			} `json:"dirinfo"`
			Songlist []struct {
				Name   string `json:"name"`
				Singer []struct {
					Name string `json:"name"`
				} `json:"singer"`
			} `json:"songlist"`
		} `json:"data"`
	} `json:"req_0"`
}

// QQMusicLegacyResp 是旧版公开歌单详情接口的响应。
// 部分新歌单在新版 musicu 接口返回 code=2000 时，可由该接口获取完整数据。
type QQMusicLegacyResp struct {
	Code    int    `json:"code"`
	Subcode int    `json:"subcode"`
	Message string `json:"msg"`
	Cdlist  []struct {
		Dissname string `json:"dissname"`
		Songnum  int    `json:"songnum"`
		Songlist []struct {
			Songname string `json:"songname"`
			Singer   []struct {
				Name string `json:"name"`
			} `json:"singer"`
		} `json:"songlist"`
	} `json:"cdlist"`
}
