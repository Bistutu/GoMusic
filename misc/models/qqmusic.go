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

func NewQQMusicReq(disstid int, platform string) *QQMusicReq {
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
				SongBegin:  0,
				SongNum:    1024,
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

func GetQQMusicReqString(disstid int, platform string) string {
	param := NewQQMusicReq(disstid, platform)
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
