// Package musicurl validates playlist links and upstream redirects against known music hosts.
package musicurl

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
)

const (
	NetEase = "netease"
	QQ      = "qq"
	Qishui  = "qishui"
	Kugou   = "kugou"
)

var links = regexp.MustCompile(`https?://[^\s<>"\x{3000}]+`)
var digits = regexp.MustCompile(`^[1-9][0-9]{0,19}$`)
var playlistPath = regexp.MustCompile(`(?:^|/)playlist/([0-9]+)(?:/|$)`)
var albumPath = regexp.MustCompile(`(?:^|/)album/([0-9]+)(?:/|$)`)
var kugouSpecial = regexp.MustCompile(`/special/single/([0-9]+)`)
var kugouGcid = regexp.MustCompile(`/songlist/(gcid_[A-Za-z0-9]+)`)
var kugouCollection = regexp.MustCompile(`^collection_[0-9]+_[0-9]+_[0-9]+_[0-9]+$`)

var hosts = map[string]string{
	"music.163.com": NetEase, "163cn.tv": NetEase, "163cn.com": NetEase,
	"y.qq.com": QQ, "i.y.qq.com": QQ, "i2.y.qq.com": QQ, "c6.y.qq.com": QQ, "c.y.qq.com": QQ, "u.y.qq.com": QQ, "u6.y.qq.com": QQ,
	"qishui.douyin.com": Qishui, "music.douyin.com": Qishui, "beta-luna.douyin.com": Qishui,
	"kugou.com": Kugou, "www.kugou.com": Kugou, "m.kugou.com": Kugou, "t.kugou.com": Kugou, "t1.kugou.com": Kugou, "activity.kugou.com": Kugou, "mobilecdn.kugou.com": Kugou, "mobiles.kugou.com": Kugou, "pubsongscdn.kugou.com": Kugou,
}

type Link struct {
	URL                *url.URL
	Provider, Kind, ID string
}

func Validate(u *url.URL) (string, error) {
	if u == nil || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil || u.Opaque != "" || u.Port() != "" {
		return "", errors.New("仅支持音乐平台的 HTTP/HTTPS 链接")
	}
	p, ok := hosts[strings.ToLower(u.Host)]
	if !ok {
		return "", errors.New("仅支持网易云音乐、QQ 音乐、汽水音乐和酷狗音乐的官方链接")
	}
	return p, nil
}

// Parse accepts a URL or the share text copied from a music app.
func Parse(input string) (Link, error) {
	for _, candidate := range links.FindAllString(input, -1) {
		candidate = strings.TrimRight(candidate, "，。；！、）】》〉,;!)\u200b")
		u, err := url.Parse(candidate)
		if err != nil {
			continue
		}
		provider, err := Validate(u)
		if err != nil {
			continue
		}
		u.Scheme = "https"
		link := Link{URL: u, Provider: provider, Kind: "playlist"}
		path, query := u.Path, u.Query()
		if strings.HasPrefix(u.Fragment, "/") {
			fragment, err := url.Parse(u.Fragment)
			if err == nil {
				path, query = fragment.Path, fragment.Query()
			}
		}
		if match := playlistPath.FindStringSubmatch(path); len(match) > 1 {
			link.ID = match[1]
		}
		if provider == NetEase && strings.Contains(path, "/album") {
			link.Kind = "album"
			if match := albumPath.FindStringSubmatch(path); len(match) > 1 {
				link.ID = match[1]
			}
		}
		if link.ID == "" {
			if provider == Qishui {
				link.ID = query.Get("playlist_id")
			} else {
				link.ID = query.Get("id")
			}
		}
		if provider == Kugou {
			// Unwrap the official Concept Edition QR share URL, but keep host validation.
			if nested := query.Get("qrcode"); nested != "" {
				if inner, e := url.Parse(nested); e == nil {
					if p, e := Validate(inner); e == nil && p == Kugou {
						query = inner.Query()
					}
				}
			}
			link.Kind = "share"
			link.ID = ""
			if collection := query.Get("global_specialid"); kugouCollection.MatchString(collection) {
				link.Kind = "collection"
				link.ID = collection
			}
			if match := kugouGcid.FindStringSubmatch(path); len(match) > 1 {
				link.Kind = "gcid"
				link.ID = match[1]
			}
			if link.ID == "" {
				if match := kugouSpecial.FindStringSubmatch(path); len(match) > 1 {
					link.Kind = "special"
					link.ID = match[1]
				}
			}
			if link.ID == "" && digits.MatchString(query.Get("specialid")) {
				link.Kind = "special"
				link.ID = query.Get("specialid")
			}
			if link.ID == "" && regexp.MustCompile(`^[A-Za-z0-9]{10,24}$`).MatchString(query.Get("src_cid")) {
				link.Kind = "gcid"
				link.ID = "gcid_" + query.Get("src_cid")
			}
			return link, nil
		}

		if link.ID != "" && provider == NetEase && !(strings.HasSuffix(path, "/playlist") || strings.HasSuffix(path, "/album") || playlistPath.MatchString(path) || albumPath.MatchString(path)) {
			return Link{}, errors.New("请使用歌单或专辑分享链接")
		}
		if link.ID != "" && provider == QQ && !(playlistPath.MatchString(path) || strings.HasSuffix(path, "/taoge.html") || strings.HasSuffix(path, "/playlist.html")) {
			return Link{}, errors.New("请使用 QQ 音乐的歌单分享链接")
		}
		if link.ID != "" && !digits.MatchString(link.ID) {
			return Link{}, errors.New("歌单 ID 无效")
		}
		return link, nil
	}
	return Link{}, errors.New("请输入网易云音乐、QQ 音乐、汽水音乐或酷狗音乐的歌单链接")
}
