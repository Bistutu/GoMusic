package logic

import (
	"net/url"
	"regexp"

	"GoMusic/common/models"
	"GoMusic/httputil"
	"GoMusic/initialize/log"
)

const (
	KuGouShort = `t1\.kugou`
	KuGouPC    = `wwwapi\.kugou`
	KuGouPhone = `m\.kugou`
)

var (
	KuGouShortRegx = regexp.MustCompile(KuGouShort)
	KuGouPCRegx    = regexp.MustCompile(KuGouPC)
	KuGouPhoneRegx = regexp.MustCompile(KuGouPhone)
)

// 短链：https://t1.kugou.com/aRgNRccBhV2
// http://wwwapi.kugou.com/share/zlist.html
// https://m.kugou.com/songlist/gcid_3zbivkavz4z072

func KuGouDiscover(link string) (*models.SongList, error) {

	return nil, nil
}

func getRealUrl(link string) (string, error) {
	var err error

	// 如果是长链附加短链的形式
	if KuGouPCRegx.MatchString(link) {
		query, err := url.ParseQuery(link)
		if err != nil {
			log.Error("fail to parse query: %v", err)
			return "", err
		}

		chain := query.Get("chain")
		if chain == "" {
			log.Error("error param, chain is empty")
			return "", err
		}
		link = "https://t1.kugou.com/" + chain
	}

	// 如果是短链
	if KuGouShortRegx.MatchString(link) {
		link, err = httputil.GetRedirectLocation(link)
		if err != nil {
			log.Error("fail to get redirect location: %v", err)
			return "", err
		}
	}

	return link, nil
}
