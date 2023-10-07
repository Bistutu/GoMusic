package models

import "fmt"

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
	Code          int         `json:"code"`
	RelatedVideos interface{} `json:"relatedVideos"`
	Playlist      struct {
		Id                    int64         `json:"id"`
		Name                  string        `json:"name"`
		CoverImgId            int64         `json:"coverImgId"`
		CoverImgUrl           string        `json:"coverImgUrl"`
		CoverImgIdStr         string        `json:"coverImgId_str"`
		AdType                int           `json:"adType"`
		UserId                int           `json:"userId"`
		CreateTime            int64         `json:"createTime"`
		Status                int           `json:"status"`
		OpRecommend           bool          `json:"opRecommend"`
		HighQuality           bool          `json:"highQuality"`
		NewImported           bool          `json:"newImported"`
		UpdateTime            int64         `json:"updateTime"`
		TrackCount            int           `json:"trackCount"`
		SpecialType           int           `json:"specialType"`
		Privacy               int           `json:"privacy"`
		TrackUpdateTime       int64         `json:"trackUpdateTime"`
		CommentThreadId       string        `json:"commentThreadId"`
		PlayCount             int           `json:"playCount"`
		TrackNumberUpdateTime int64         `json:"trackNumberUpdateTime"`
		SubscribedCount       int           `json:"subscribedCount"`
		CloudTrackCount       int           `json:"cloudTrackCount"`
		Ordered               bool          `json:"ordered"`
		Description           interface{}   `json:"description"`
		Tags                  []interface{} `json:"tags"`
		UpdateFrequency       interface{}   `json:"updateFrequency"`
		BackgroundCoverId     int           `json:"backgroundCoverId"`
		BackgroundCoverUrl    interface{}   `json:"backgroundCoverUrl"`
		TitleImage            int           `json:"titleImage"`
		TitleImageUrl         interface{}   `json:"titleImageUrl"`
		EnglishTitle          interface{}   `json:"englishTitle"`
		OfficialPlaylistType  interface{}   `json:"officialPlaylistType"`
		Copied                bool          `json:"copied"`
		RelateResType         interface{}   `json:"relateResType"`
		Subscribers           []interface{} `json:"subscribers"`
		Subscribed            interface{}   `json:"subscribed"`
		Creator               struct {
			DefaultAvatar       bool        `json:"defaultAvatar"`
			Province            int         `json:"province"`
			AuthStatus          int         `json:"authStatus"`
			Followed            bool        `json:"followed"`
			AvatarUrl           string      `json:"avatarUrl"`
			AccountStatus       int         `json:"accountStatus"`
			Gender              int         `json:"gender"`
			City                int         `json:"city"`
			Birthday            int         `json:"birthday"`
			UserId              int         `json:"userId"`
			UserType            int         `json:"userType"`
			Nickname            string      `json:"nickname"`
			Signature           string      `json:"signature"`
			Description         string      `json:"description"`
			DetailDescription   string      `json:"detailDescription"`
			AvatarImgId         int64       `json:"avatarImgId"`
			BackgroundImgId     int64       `json:"backgroundImgId"`
			BackgroundUrl       string      `json:"backgroundUrl"`
			Authority           int         `json:"authority"`
			Mutual              bool        `json:"mutual"`
			ExpertTags          interface{} `json:"expertTags"`
			Experts             interface{} `json:"experts"`
			DjStatus            int         `json:"djStatus"`
			VipType             int         `json:"vipType"`
			RemarkName          interface{} `json:"remarkName"`
			AuthenticationTypes int         `json:"authenticationTypes"`
			AvatarDetail        interface{} `json:"avatarDetail"`
			Anchor              bool        `json:"anchor"`
			AvatarImgIdStr      string      `json:"avatarImgIdStr"`
			BackgroundImgIdStr  string      `json:"backgroundImgIdStr"`
			AvatarImgIdStr1     string      `json:"avatarImgId_str"`
		} `json:"creator"`
		Tracks []struct {
			Name string `json:"name"`
			Id   uint   `json:"id"`
			Pst  int    `json:"pst"`
			T    int    `json:"t"`
			Ar   []struct {
				Id    int           `json:"id"`
				Name  string        `json:"name"`
				Tns   []interface{} `json:"tns"`
				Alias []interface{} `json:"alias"`
			} `json:"ar"`
			Alia []interface{} `json:"alia"`
			Pop  float64       `json:"pop"`
			St   int           `json:"st"`
			Rt   *string       `json:"rt"`
			Fee  int           `json:"fee"`
			V    int           `json:"v"`
			Crbt interface{}   `json:"crbt"`
			Cf   string        `json:"cf"`
			Al   struct {
				Id     int           `json:"id"`
				Name   string        `json:"name"`
				PicUrl string        `json:"picUrl"`
				Tns    []interface{} `json:"tns"`
				PicStr string        `json:"pic_str,omitempty"`
				Pic    int64         `json:"pic"`
			} `json:"al"`
			Dt int `json:"dt"`
			H  *struct {
				Br   int     `json:"br"`
				Fid  int     `json:"fid"`
				Size int     `json:"size"`
				Vd   float64 `json:"vd"`
			} `json:"h"`
			M *struct {
				Br   int     `json:"br"`
				Fid  int     `json:"fid"`
				Size int     `json:"size"`
				Vd   float64 `json:"vd"`
			} `json:"m"`
			L struct {
				Br   int     `json:"br"`
				Fid  int     `json:"fid"`
				Size int     `json:"size"`
				Vd   float64 `json:"vd"`
			} `json:"l"`
			Sq *struct {
				Br   int     `json:"br"`
				Fid  int     `json:"fid"`
				Size int     `json:"size"`
				Vd   float64 `json:"vd"`
			} `json:"sq"`
			Hr *struct {
				Br   int     `json:"br"`
				Fid  int     `json:"fid"`
				Size int     `json:"size"`
				Vd   float64 `json:"vd"`
			} `json:"hr"`
			A                    interface{}   `json:"a"`
			Cd                   string        `json:"cd"`
			No                   int           `json:"no"`
			RtUrl                interface{}   `json:"rtUrl"`
			Ftype                int           `json:"ftype"`
			RtUrls               []interface{} `json:"rtUrls"`
			DjId                 int           `json:"djId"`
			Copyright            int           `json:"copyright"`
			SId                  int           `json:"s_id"`
			Mark                 int64         `json:"mark"`
			OriginCoverType      int           `json:"originCoverType"`
			OriginSongSimpleData *struct {
				SongId  int    `json:"songId"`
				Name    string `json:"name"`
				Artists []struct {
					Id   int    `json:"id"`
					Name string `json:"name"`
				} `json:"artists"`
				AlbumMeta struct {
					Id   int    `json:"id"`
					Name string `json:"name"`
				} `json:"albumMeta"`
			} `json:"originSongSimpleData"`
			TagPicList        interface{} `json:"tagPicList"`
			ResourceState     bool        `json:"resourceState"`
			Version           int         `json:"version"`
			SongJumpInfo      interface{} `json:"songJumpInfo"`
			EntertainmentTags interface{} `json:"entertainmentTags"`
			AwardTags         interface{} `json:"awardTags"`
			Single            int         `json:"single"`
			NoCopyrightRcmd   interface{} `json:"noCopyrightRcmd"`
			Mst               int         `json:"mst"`
			Cp                int         `json:"cp"`
			Mv                int         `json:"mv"`
			Rtype             int         `json:"rtype"`
			Rurl              interface{} `json:"rurl"`
			PublishTime       int64       `json:"publishTime"`
		} `json:"tracks"`
		VideoIds interface{} `json:"videoIds"`
		Videos   interface{} `json:"videos"`
		TrackIds []struct {
			Id         uint        `json:"id"`
			V          int         `json:"v"`
			T          int         `json:"t"`
			At         int64       `json:"at"`
			Alg        interface{} `json:"alg"`
			Uid        int         `json:"uid"`
			RcmdReason string      `json:"rcmdReason"`
			Sc         interface{} `json:"sc"`
			F          interface{} `json:"f"`
			Sr         interface{} `json:"sr"`
		} `json:"trackIds"`
		BannedTrackIds     interface{} `json:"bannedTrackIds"`
		MvResourceInfos    interface{} `json:"mvResourceInfos"`
		ShareCount         int         `json:"shareCount"`
		CommentCount       int         `json:"commentCount"`
		RemixVideo         interface{} `json:"remixVideo"`
		SharedUsers        interface{} `json:"sharedUsers"`
		HistorySharedUsers interface{} `json:"historySharedUsers"`
		GradeStatus        string      `json:"gradeStatus"`
		Score              interface{} `json:"score"`
		AlgTags            interface{} `json:"algTags"`
		TrialMode          int         `json:"trialMode"`
	} `json:"playlist"`
	Urls       interface{} `json:"urls"`
	Privileges []struct {
		Id                 int         `json:"id"`
		Fee                int         `json:"fee"`
		Payed              int         `json:"payed"`
		RealPayed          int         `json:"realPayed"`
		St                 int         `json:"st"`
		Pl                 int         `json:"pl"`
		Dl                 int         `json:"dl"`
		Sp                 int         `json:"sp"`
		Cp                 int         `json:"cp"`
		Subp               int         `json:"subp"`
		Cs                 bool        `json:"cs"`
		Maxbr              int         `json:"maxbr"`
		Fl                 int         `json:"fl"`
		Pc                 interface{} `json:"pc"`
		Toast              bool        `json:"toast"`
		Flag               int         `json:"flag"`
		PaidBigBang        bool        `json:"paidBigBang"`
		PreSell            bool        `json:"preSell"`
		PlayMaxbr          int         `json:"playMaxbr"`
		DownloadMaxbr      int         `json:"downloadMaxbr"`
		MaxBrLevel         string      `json:"maxBrLevel"`
		PlayMaxBrLevel     string      `json:"playMaxBrLevel"`
		DownloadMaxBrLevel string      `json:"downloadMaxBrLevel"`
		PlLevel            string      `json:"plLevel"`
		DlLevel            string      `json:"dlLevel"`
		FlLevel            string      `json:"flLevel"`
		Rscl               interface{} `json:"rscl"`
		FreeTrialPrivilege struct {
			ResConsumable      bool        `json:"resConsumable"`
			UserConsumable     bool        `json:"userConsumable"`
			ListenType         interface{} `json:"listenType"`
			CannotListenReason *int        `json:"cannotListenReason"`
		} `json:"freeTrialPrivilege"`
		RightSource    int `json:"rightSource"`
		ChargeInfoList []struct {
			Rate          int         `json:"rate"`
			ChargeUrl     interface{} `json:"chargeUrl"`
			ChargeMessage interface{} `json:"chargeMessage"`
			ChargeType    int         `json:"chargeType"`
		} `json:"chargeInfoList"`
	} `json:"privileges"`
	SharedPrivilege interface{} `json:"sharedPrivilege"`
	ResEntrance     interface{} `json:"resEntrance"`
	FromUsers       interface{} `json:"fromUsers"`
	FromUserCount   int         `json:"fromUserCount"`
	SongFromUsers   interface{} `json:"songFromUsers"`
}

type Songs struct {
	Songs []struct {
		Name string `json:"name"`
		Id   int    `json:"id"`
		Pst  int    `json:"pst"`
		T    int    `json:"t"`
		Ar   []struct {
			Id    int           `json:"id"`
			Name  string        `json:"name"`
			Tns   []interface{} `json:"tns"`
			Alias []interface{} `json:"alias"`
		} `json:"ar"`
		Alia []interface{} `json:"alia"`
		Pop  float64       `json:"pop"`
		St   int           `json:"st"`
		Rt   string        `json:"rt"`
		Fee  int           `json:"fee"`
		V    int           `json:"v"`
		Crbt interface{}   `json:"crbt"`
		Cf   string        `json:"cf"`
		Al   struct {
			Id     int           `json:"id"`
			Name   string        `json:"name"`
			PicUrl string        `json:"picUrl"`
			Tns    []interface{} `json:"tns"`
			PicStr string        `json:"pic_str"`
			Pic    int64         `json:"pic"`
		} `json:"al"`
		Dt int `json:"dt"`
		H  struct {
			Br   int     `json:"br"`
			Fid  int     `json:"fid"`
			Size int     `json:"size"`
			Vd   float64 `json:"vd"`
			Sr   int     `json:"sr"`
		} `json:"h"`
		M struct {
			Br   int     `json:"br"`
			Fid  int     `json:"fid"`
			Size int     `json:"size"`
			Vd   float64 `json:"vd"`
			Sr   int     `json:"sr"`
		} `json:"m"`
		L struct {
			Br   int     `json:"br"`
			Fid  int     `json:"fid"`
			Size int     `json:"size"`
			Vd   float64 `json:"vd"`
			Sr   int     `json:"sr"`
		} `json:"l"`
		Sq *struct {
			Br   int     `json:"br"`
			Fid  int     `json:"fid"`
			Size int     `json:"size"`
			Vd   float64 `json:"vd"`
			Sr   int     `json:"sr"`
		} `json:"sq"`
		Hr                   interface{}   `json:"hr"`
		A                    interface{}   `json:"a"`
		Cd                   string        `json:"cd"`
		No                   int           `json:"no"`
		RtUrl                interface{}   `json:"rtUrl"`
		Ftype                int           `json:"ftype"`
		RtUrls               []interface{} `json:"rtUrls"`
		DjId                 int           `json:"djId"`
		Copyright            int           `json:"copyright"`
		SId                  int           `json:"s_id"`
		Mark                 int           `json:"mark"`
		OriginCoverType      int           `json:"originCoverType"`
		OriginSongSimpleData interface{}   `json:"originSongSimpleData"`
		TagPicList           interface{}   `json:"tagPicList"`
		ResourceState        bool          `json:"resourceState"`
		Version              int           `json:"version"`
		SongJumpInfo         interface{}   `json:"songJumpInfo"`
		EntertainmentTags    interface{}   `json:"entertainmentTags"`
		AwardTags            interface{}   `json:"awardTags"`
		Single               int           `json:"single"`
		NoCopyrightRcmd      interface{}   `json:"noCopyrightRcmd"`
		Rtype                int           `json:"rtype"`
		Rurl                 interface{}   `json:"rurl"`
		Mst                  int           `json:"mst"`
		Cp                   int           `json:"cp"`
		Mv                   int           `json:"mv"`
		PublishTime          int64         `json:"publishTime"`
	} `json:"songs"`
	Privileges []struct {
		Id                 int         `json:"id"`
		Fee                int         `json:"fee"`
		Payed              int         `json:"payed"`
		St                 int         `json:"st"`
		Pl                 int         `json:"pl"`
		Dl                 int         `json:"dl"`
		Sp                 int         `json:"sp"`
		Cp                 int         `json:"cp"`
		Subp               int         `json:"subp"`
		Cs                 bool        `json:"cs"`
		Maxbr              int         `json:"maxbr"`
		Fl                 int         `json:"fl"`
		Toast              bool        `json:"toast"`
		Flag               int         `json:"flag"`
		PreSell            bool        `json:"preSell"`
		PlayMaxbr          int         `json:"playMaxbr"`
		DownloadMaxbr      int         `json:"downloadMaxbr"`
		MaxBrLevel         string      `json:"maxBrLevel"`
		PlayMaxBrLevel     string      `json:"playMaxBrLevel"`
		DownloadMaxBrLevel string      `json:"downloadMaxBrLevel"`
		PlLevel            string      `json:"plLevel"`
		DlLevel            string      `json:"dlLevel"`
		FlLevel            string      `json:"flLevel"`
		Rscl               interface{} `json:"rscl"`
		FreeTrialPrivilege struct {
			ResConsumable  bool        `json:"resConsumable"`
			UserConsumable bool        `json:"userConsumable"`
			ListenType     interface{} `json:"listenType"`
		} `json:"freeTrialPrivilege"`
		ChargeInfoList []struct {
			Rate          int         `json:"rate"`
			ChargeUrl     interface{} `json:"chargeUrl"`
			ChargeMessage interface{} `json:"chargeMessage"`
			ChargeType    int         `json:"chargeType"`
		} `json:"chargeInfoList"`
	} `json:"privileges"`
	Code int `json:"code"`
}
