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

func NewQQMusicReq(disstid int) *QQMusicReq {
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
			Platform: "h5",
		},
	}
}

func GetQQMusicReqString(disstid int) string {
	param := NewQQMusicReq(disstid)
	marshal, _ := json.Marshal(param)
	return string(marshal)
}

type QQMusicResp struct {
	Code    int    `json:"code"`
	Ts      int64  `json:"ts"`
	StartTs int64  `json:"start_ts"`
	Traceid string `json:"traceid"`
	Req0    struct {
		Code int `json:"code"`
		Data struct {
			Code               int    `json:"code"`
			Subcode            int    `json:"subcode"`
			Msg                string `json:"msg"`
			FromGedanPlaza     int    `json:"from_gedan_plaza"`
			AccessedPlazaCache int    `json:"accessed_plaza_cache"`
			AccessedByfav      int    `json:"accessed_byfav"`
			Optype             int    `json:"optype"`
			FilterSongNum      int    `json:"filter_song_num"`
			Dirinfo            struct {
				Id             int64         `json:"id"`
				HostUin        int           `json:"host_uin"`
				Dirid          int           `json:"dirid"`
				Title          string        `json:"title"`
				Picurl         string        `json:"picurl"`
				Picid          int           `json:"picid"`
				Desc           string        `json:"desc"`
				VecTagid       []interface{} `json:"vec_tagid"`
				VecTagname     []interface{} `json:"vec_tagname"`
				Ctime          int           `json:"ctime"`
				Mtime          int           `json:"mtime"`
				Listennum      int           `json:"listennum"`
				Ordernum       int           `json:"ordernum"`
				Picmid         string        `json:"picmid"`
				Dirtype        int           `json:"dirtype"`
				HostNick       string        `json:"host_nick"`
				Songnum        int           `json:"songnum"`
				Ordertime      int           `json:"ordertime"`
				Show           int           `json:"show"`
				Picurl2        string        `json:"picurl2"`
				SongUpdateTime int           `json:"song_update_time"`
				SongUpdateNum  int           `json:"song_update_num"`
				Disstype       int           `json:"disstype"`
				AiUin          int           `json:"ai_uin"`
				Dv2            int           `json:"dv2"`
				DirShow        int           `json:"dir_show"`
				EncryptUin     string        `json:"encrypt_uin"`
				EncryptAiUin   string        `json:"encrypt_ai_uin"`
				Owndir         int           `json:"owndir"`
				Headurl        string        `json:"headurl"`
				Tag            []interface{} `json:"tag"`
				Creator        struct {
					Musicid      int         `json:"musicid"`
					Type         int         `json:"type"`
					Singerid     int         `json:"singerid"`
					Nick         string      `json:"nick"`
					Headurl      string      `json:"headurl"`
					Ifpicurl     string      `json:"ifpicurl"`
					EncryptUin   string      `json:"encrypt_uin"`
					IsVip        int         `json:"isVip"`
					AiUin        int         `json:"ai_uin"`
					EncryptAiUin string      `json:"encrypt_ai_uin"`
					Ext          interface{} `json:"ext"`
				} `json:"creator"`
				Status      int    `json:"status"`
				EdgeMark    string `json:"edge_mark"`
				LayerUrl    string `json:"layer_url"`
				Ext1        string `json:"ext1"`
				Ext2        string `json:"ext2"`
				OriginTitle string `json:"origin_title"`
				AdTag       bool   `json:"ad_tag"`
				AiToast     string `json:"aiToast"`
			} `json:"dirinfo"`
			Songlist []struct {
				Id         int    `json:"id"`
				Type       int    `json:"type"`
				Songtype   int    `json:"songtype"`
				Version    int    `json:"version"`
				Trace      string `json:"trace"`
				Mid        string `json:"mid"`
				Name       string `json:"name"`
				Label      string `json:"label"`
				Title      string `json:"title"`
				Subtitle   string `json:"subtitle"`
				Interval   int    `json:"interval"`
				Isonly     int    `json:"isonly"`
				Language   int    `json:"language"`
				Genre      int    `json:"genre"`
				IndexCd    int    `json:"index_cd"`
				IndexAlbum int    `json:"index_album"`
				Status     int    `json:"status"`
				Fnote      int    `json:"fnote"`
				Url        string `json:"url"`
				TimePublic string `json:"time_public"`
				Singer     []struct {
					Id    int    `json:"id"`
					Mid   string `json:"mid"`
					Name  string `json:"name"`
					Title string `json:"title"`
				} `json:"singer"`
				Album struct {
					Id    int    `json:"id"`
					Mid   string `json:"mid"`
					Name  string `json:"name"`
					Title string `json:"title"`
					Pmid  string `json:"pmid"`
				} `json:"album"`
				Mv struct {
					Id  int    `json:"id"`
					Vid string `json:"vid"`
					Vt  int    `json:"vt"`
				} `json:"mv"`
				Ksong struct {
					Id  int    `json:"id"`
					Mid string `json:"mid"`
				} `json:"ksong"`
				File struct {
					MediaMid      string        `json:"media_mid"`
					SizeTry       int           `json:"size_try"`
					TryBegin      int           `json:"try_begin"`
					TryEnd        int           `json:"try_end"`
					Size24Aac     int           `json:"size_24aac"`
					Size48Aac     int           `json:"size_48aac"`
					Size96Aac     int           `json:"size_96aac"`
					Size128Mp3    int           `json:"size_128mp3"`
					Size192Ogg    int           `json:"size_192ogg"`
					Size192Aac    int           `json:"size_192aac"`
					Size320Mp3    int           `json:"size_320mp3"`
					SizeFlac      int           `json:"size_flac"`
					SizeApe       int           `json:"size_ape"`
					SizeDts       int           `json:"size_dts"`
					SizeHires     int           `json:"size_hires"`
					HiresSample   int           `json:"hires_sample"`
					HiresBitdepth int           `json:"hires_bitdepth"`
					B30S          int           `json:"b_30s"`
					E30S          int           `json:"e_30s"`
					Size96Ogg     int           `json:"size_96ogg"`
					Size360Ra     []interface{} `json:"size_360ra"`
					SizeDolby     int           `json:"size_dolby"`
					SizeNew       []int         `json:"size_new"`
				} `json:"file"`
				Volume struct {
					Gain float64 `json:"gain"`
					Peak float64 `json:"peak"`
					Lra  float64 `json:"lra"`
				} `json:"volume"`
				Pay struct {
					PayMonth   int `json:"pay_month"`
					PriceTrack int `json:"price_track"`
					PriceAlbum int `json:"price_album"`
					PayPlay    int `json:"pay_play"`
					PayDown    int `json:"pay_down"`
					PayStatus  int `json:"pay_status"`
					TimeFree   int `json:"time_free"`
				} `json:"pay"`
				Action struct {
					Switch   int `json:"switch"`
					Alert    int `json:"alert"`
					Msgshare int `json:"msgshare"`
					Msgfav   int `json:"msgfav"`
					Msgid    int `json:"msgid"`
					Msgdown  int `json:"msgdown"`
					Icons    int `json:"icons"`
					Msgpay   int `json:"msgpay"`
					Switch2  int `json:"switch2"`
					Icon2    int `json:"icon2"`
				} `json:"action"`
				UiAction int      `json:"uiAction"`
				NewIcon  int      `json:"new_icon"`
				Tid      int      `json:"tid"`
				Ov       int      `json:"ov"`
				Tf       string   `json:"tf"`
				Sa       int      `json:"sa"`
				Es       string   `json:"es"`
				Abt      string   `json:"abt"`
				Pingpong string   `json:"pingpong"`
				DataType int      `json:"data_type"`
				Ppurl    string   `json:"ppurl"`
				Vs       []string `json:"vs"`
				Bpm      int      `json:"bpm"`
				Ktag     string   `json:"ktag"`
				Team     string   `json:"team"`
			} `json:"songlist"`
			LoginUin     int           `json:"login_uin"`
			InvalidSong  []interface{} `json:"invalid_song"`
			FilteredSong []interface{} `json:"filtered_song"`
			AdList       []interface{} `json:"ad_list"`
			TotalSongNum int           `json:"total_song_num"`
			EncryptLogin string        `json:"encrypt_login"`
			Ct           int           `json:"ct"`
			Cv           int           `json:"cv"`
			Ip           string        `json:"ip"`
			Orderlist    []interface{} `json:"orderlist"`
			CmtURLBykey  struct {
				UrlKey    string `json:"url_key"`
				UrlParams string `json:"url_params"`
			} `json:"cmtURL_bykey"`
			SrfIp           string        `json:"srf_ip"`
			Referer         string        `json:"referer"`
			Namedflag       int           `json:"namedflag"`
			IsAd            int           `json:"isAd"`
			AdTitle         string        `json:"adTitle"`
			AdUrl           string        `json:"adUrl"`
			IsForbidComment int           `json:"isForbidComment"`
			Songtag         []interface{} `json:"songtag"`
			ToplistSong     []interface{} `json:"toplist_song"`
			ToplistNolimit  bool          `json:"toplist_nolimit"`
			SacForbid       []interface{} `json:"sac_forbid"`
			Birthday        []interface{} `json:"birthday"`
			AiExt           struct {
				Couple          []interface{} `json:"couple"`
				Recommdays      int           `json:"recommdays"`
				NextLink        string        `json:"nextLink"`
				AiSongExt       []interface{} `json:"aiSongExt"`
				AllListening    int           `json:"allListening"`
				StrAllListening string        `json:"strAllListening"`
				ListeningIcon   string        `json:"listeningIcon"`
			} `json:"aiExt"`
			VecSongidNewtime []interface{} `json:"vec_songid_newtime"`
			VecSongidType    []interface{} `json:"vec_songid_type"`
			VecAiExtern      []interface{} `json:"vec_ai_extern"`
			RecomUgcValid    int           `json:"recomUgcValid"`
			QuickListenVid   []interface{} `json:"quickListenVid"`
			Bitflag          int           `json:"bitflag"`
		} `json:"data"`
	} `json:"req_0"`
}
