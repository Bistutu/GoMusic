package logic

import (
	"fmt"
	"testing"
)

// https://t1.kugou.com/aRgNRccBhV2
// http://wwwapi.kugou.com/share/zlist.html
// https://m.kugou.com/songlist/gcid_3zbivkavz4z072

func TestKuGouDiscover(t *testing.T) {

}

var (
	UrlShortParam = "https://t1.kugou.com/song.html?id=aZYRx66BhV2"
	UrlShort      = "https://t1.kugou.com/song.html?id=aZYRx66BhV2"
	UrlPc         = "http://wwwapi.kugou.com/share/zlist.html?listid=4&type=0&uid=606105881&share_type=collect&from=pcCode&_t=1701149199374&sign=344d6afe7eaa30508f54e8db35e09be8&chain=aRgNRccBhV2"
	UrlPhone      = "https://m.kugou.com/songlist/gcid_3zbivkavz4z072/?src_cid=3zbivkavz4z072&uid=606105881&chl=link&iszlist=1"
)

func TestKuGouRegex(t *testing.T) {
	fmt.Println(KuGouShortRegx.MatchString(UrlShort))
	fmt.Println(KuGouPCRegx.MatchString(UrlPc))
	fmt.Println(KuGouPhoneRegx.MatchString(UrlPhone))
}

func TestGetRealUrl(t *testing.T) {
	fmt.Println(getRealUrl(UrlShort))
	fmt.Println(getRealUrl(UrlPc))
	fmt.Println(getRealUrl(UrlPhone))
}
