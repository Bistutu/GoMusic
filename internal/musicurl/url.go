// Package musicurl validates supported music links and upstream redirects against known music hosts.
package musicurl

import (
	"errors"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

const (
	NetEase  = "netease"
	QQ       = "qq"
	Qishui   = "qishui"
	Kugou    = "kugou"
	Kuwo     = "kuwo"
	Migu     = "migu"
	Qianqian = "qianqian"
	Bodian   = "bodian"
	Deezer   = "deezer"
)

var links = regexp.MustCompile(`(?i)https?://[^\s<>"'\x{3000}\x{200b}，。；！、（）【】《》〉,;!()\]]+`)
var htmlQuerySeparators = regexp.MustCompile(`(?i)&(?:amp|#0*38|#x0*26);`)
var jsonEscapedSlashes = regexp.MustCompile(`\\/`)
var jsonEscapedAmpersands = regexp.MustCompile(`(?i)\\u0026`)
var protocolRelativeLinks = regexp.MustCompile(`(?i)(^|[^a-z0-9@./:-])(//[a-z0-9-]+(\.[a-z0-9-]+)+/[^\s<>"'，。；！、）】》〉,;!)]+)`)
var bareLinks = regexp.MustCompile(`(?i)(^|[^a-z0-9@./-])([a-z0-9-]+(\.[a-z0-9-]+)+/[^\s<>"'，。；！、）】》〉,;!)]+)`)
var digits = regexp.MustCompile(`^[1-9][0-9]{0,19}$`)
var playlistPath = regexp.MustCompile(`(?:^|/)playlist/([0-9]+)(?:/|$)`)
var albumPath = regexp.MustCompile(`(?:^|/)album/([0-9]+)(?:/|$)`)
var netEaseSongPath = regexp.MustCompile(`^/(?:m/)?song/?$`)
var qqAlbumPath = regexp.MustCompile(`^/n/ryqq(?:_v2)?/albumDetail/([A-Za-z0-9]{14})(?:/|$)`)
var qqSongPath = regexp.MustCompile(`^/n/ryqq/songDetail/([A-Za-z0-9]{14})(?:/|$)`)
var qqLegacySongPath = regexp.MustCompile(`^/n/yqq/song/([A-Za-z0-9]{14})\.html$`)
var qishuiShortSharePath = regexp.MustCompile(`^/s/[A-Za-z0-9_-]{4,128}/?$`)
var qishuiPlaylistSharePath = regexp.MustCompile(`^/qishui/share/playlist/?$`)
var qishuiPlaylistDetailPath = regexp.MustCompile(`^/luna/playlist/detail/?$`)
var qqMID = regexp.MustCompile(`^[A-Za-z0-9]{14}$`)
var kuwoPlaylistPath = regexp.MustCompile(`^/playlist_detail/([0-9]+)(?:/|$)`)
var kuwoMobilePlaylistPath = regexp.MustCompile(`^/newh5app/playlist_detail/([0-9]+)(?:/|$)`)
var kuwoLegacyPlaylistIndexPath = regexp.MustCompile(`^/playlist/index/?$`)
var kuwoPlaylistRootPath = regexp.MustCompile(`^/playlist_detail/?$`)
var kuwoAlbumPath = regexp.MustCompile(`^/album_detail/([0-9]+)(?:/|$)`)
var kuwoTrackPath = regexp.MustCompile(`^/play_detail/([0-9]+)(?:/|$)`)
var kuwoLegacyTrackPath = regexp.MustCompile(`^/yinyue/([0-9]+)/?$`)
var kuwoRanklistPath = regexp.MustCompile(`^/newh5app/ranklist_detail/([0-9]+)(?:/|$)`)
var kuwoBangContentPath = regexp.MustCompile(`^/bang/content/?$`)
var kuwoH5PlaylistPath = regexp.MustCompile(`^/newh5(?:app)?/playlist/content/?$`)
var kuwoShortSharePath = regexp.MustCompile(`^/s/[A-Za-z0-9_-]{4,128}/?$`)
var kuwoShortHostPath = regexp.MustCompile(`^/(?:s/)?[A-Za-z0-9_-]{4,128}/?$`)
var miguPlaylistPath = regexp.MustCompile(`^/v3/music/playlist/([0-9]+)(?:/|$)`)
var miguAlbumPath = regexp.MustCompile(`^/v3/music/album/([0-9]+)(?:/|$)`)
var miguV5RootPath = regexp.MustCompile(`^/v5/?$`)
var miguHashPlaylistPath = regexp.MustCompile(`^/playlist/?$`)
var qianqianPlaylistPath = regexp.MustCompile(`^/songlist/([0-9]+)(?:/|$)`)
var bodianCollectionPath = regexp.MustCompile(`^/m/bodian/collection\.html/?$`)
var bodianSourceID = regexp.MustCompile(`^[0-9]{1,3}$`)
var deezerPlaylistPath = regexp.MustCompile(`^/(?:[a-z]{2}(?:-[a-z]{2})?/)?playlist/([0-9]+)(?:/|$)`)
var deezerAlbumPath = regexp.MustCompile(`^/(?:[a-z]{2}(?:-[a-z]{2})?/)?album/([0-9]+)(?:/|$)`)
var deezerTrackPath = regexp.MustCompile(`^/(?:[a-z]{2}(?:-[a-z]{2})?/)?track/([0-9]+)(?:/|$)`)
var deezerShortSharePath = regexp.MustCompile(`^/s/[A-Za-z0-9_-]{4,128}/?$`)
var kugouSpecial = regexp.MustCompile(`/special/single/([0-9]+)`)
var kugouGcid = regexp.MustCompile(`/songlist/(gcid_[A-Za-z0-9]+)`)
var kugouCollection = regexp.MustCompile(`^collection_[0-9]+_[0-9]+_[0-9]+_[0-9]+$`)
var kugouCode = regexp.MustCompile(`(?i)^(?:(?:酷狗码|kugou\s+code)\s*[:：]?\s*)?([0-9]{1,20})$`)

var hosts = map[string]string{
	"music.163.com": NetEase, "y.music.163.com": NetEase, "163cn.tv": NetEase, "163cn.com": NetEase,
	"y.qq.com": QQ, "i.y.qq.com": QQ, "i2.y.qq.com": QQ, "c6.y.qq.com": QQ, "c.y.qq.com": QQ, "u.y.qq.com": QQ, "u6.y.qq.com": QQ,
	"qishui.douyin.com": Qishui, "music.douyin.com": Qishui, "beta-luna.douyin.com": Qishui,
	"kugou.com": Kugou, "www.kugou.com": Kugou, "m.kugou.com": Kugou, "t.kugou.com": Kugou, "t1.kugou.com": Kugou, "activity.kugou.com": Kugou, "mobilecdn.kugou.com": Kugou, "mobiles.kugou.com": Kugou, "pubsongscdn.kugou.com": Kugou,
	"kuwo.cn": Kuwo, "www.kuwo.cn": Kuwo, "m.kuwo.cn": Kuwo, "t.kuwo.cn": Kuwo, "nplserver.kuwo.cn": Kuwo, "kbangserver.kuwo.cn": Kuwo,
	"music.migu.cn": Migu, "app.c.nf.migu.cn": Migu,
	"music.91q.com": Qianqian,
	"h5app.kuwo.cn": Bodian, "bodian.kuwo.cn": Bodian, "bd-api.kuwo.cn": Bodian,
	"deezer.com": Deezer, "www.deezer.com": Deezer, "link.deezer.com": Deezer, "api.deezer.com": Deezer,
}

type Link struct {
	URL                *url.URL
	Provider, Kind, ID string
}

func Validate(u *url.URL) (string, error) {
	if u == nil || (!strings.EqualFold(u.Scheme, "https") && !strings.EqualFold(u.Scheme, "http")) || u.User != nil || u.Opaque != "" || u.Port() != "" {
		return "", errors.New("仅支持音乐平台的 HTTP/HTTPS 链接")
	}
	p, ok := hosts[strings.ToLower(u.Host)]
	if !ok {
		return "", errors.New("仅支持网易云音乐、QQ 音乐、汽水音乐、酷狗音乐、酷我音乐、咪咕音乐、千千音乐、波点音乐和 Deezer 的官方链接")
	}
	return p, nil
}

// Parse accepts an official URL, share text, or a numeric Kugou playlist code.
func Parse(input string) (Link, error) {
	link, err := parseLinkInput(input)
	if err == nil {
		return link, nil
	}
	for depth := 0; depth < 2; depth++ {
		decoded, decodeErr := unescapeShareText(input)
		if decodeErr != nil || decoded == input {
			break
		}
		input = decoded
		candidate, candidateErr := parseLinkInput(input)
		if candidateErr == nil {
			return candidate, nil
		}
		link, err = candidate, candidateErr
	}
	return link, err
}

func unescapeShareText(input string) (string, error) {
	var encoded strings.Builder
	for i := 0; i < len(input); i++ {
		if input[i] != '%' {
			encoded.WriteByte(input[i])
			continue
		}
		if i+2 < len(input) && isHexDigit(input[i+1]) && isHexDigit(input[i+2]) {
			encoded.WriteString(input[i : i+3])
			i += 2
			continue
		}
		encoded.WriteString("%25")
	}
	return url.PathUnescape(encoded.String())
}

func isHexDigit(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
}

func parseLinkInput(input string) (Link, error) {
	input = jsonEscapedSlashes.ReplaceAllString(input, "/")
	input = jsonEscapedAmpersands.ReplaceAllString(input, "&")
	input = htmlQuerySeparators.ReplaceAllString(input, "&")
	var routeErr error
	type candidateURL struct {
		start int
		value string
	}
	var candidates []candidateURL
	for _, match := range links.FindAllStringIndex(input, -1) {
		candidates = append(candidates, candidateURL{start: match[0], value: input[match[0]:match[1]]})
	}
	for _, match := range protocolRelativeLinks.FindAllStringSubmatchIndex(input, -1) {
		candidates = append(candidates, candidateURL{start: match[4], value: "https:" + input[match[4]:match[5]]})
	}
	for _, match := range bareLinks.FindAllStringSubmatchIndex(input, -1) {
		candidates = append(candidates, candidateURL{start: match[4], value: "https://" + input[match[4]:match[5]]})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].start < candidates[j].start
	})
	for _, found := range candidates {
		candidate := found.value
		candidate = strings.TrimRight(candidate, "，。；！、）】》〉,;!)\u200b")
		u, err := url.Parse(candidate)
		if err != nil {
			continue
		}
		legacyDeezerHost := strings.ToLower(u.Hostname())
		validScheme := strings.EqualFold(u.Scheme, "https") || strings.EqualFold(u.Scheme, "http")
		if validScheme && u.User == nil && u.Opaque == "" && u.Port() == "" && (legacyDeezerHost == "deezer.page.link" || legacyDeezerHost == "dzr.page.link") {
			routeErr = errors.New("Deezer 旧版 page.link 短链已停止跳转，请重新复制当前公开歌单分享链接（如 link.deezer.com/s/...）或粘贴 deezer.com 歌单链接")
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
		if provider == QQ {
			if match := qqAlbumPath.FindStringSubmatch(path); len(match) > 1 {
				link.Kind = "album"
				link.ID = match[1]
			} else if match := qqSongPath.FindStringSubmatch(path); len(match) > 1 {
				if strings.ToLower(u.Hostname()) != "y.qq.com" {
					routeErr = errors.New("请使用 QQ 音乐的公开歌单分享链接")
					continue
				}
				link.Kind = "track"
				link.ID = match[1]
			} else if match := qqLegacySongPath.FindStringSubmatch(path); len(match) > 1 {
				if strings.ToLower(u.Hostname()) != "y.qq.com" {
					routeErr = errors.New("请使用 QQ 音乐的公开歌单分享链接")
					continue
				}
				link.Kind = "track"
				link.ID = match[1]
			} else if strings.HasPrefix(path, "/n/ryqq/songDetail/") || strings.HasPrefix(path, "/n/yqq/song/") {
				routeErr = errors.New("请使用 QQ 音乐的公开歌单分享链接")
				continue
			}
		}
		if provider == NetEase {
			if strings.Contains(path, "/album") {
				link.Kind = "album"
				if match := albumPath.FindStringSubmatch(path); len(match) > 1 {
					link.ID = match[1]
				}
			} else if netEaseSongPath.MatchString(path) {
				host := strings.ToLower(u.Hostname())
				if host != "music.163.com" && host != "y.music.163.com" {
					routeErr = errors.New("请使用网易云音乐的公开歌单分享链接")
					continue
				}
				link.Kind = "track"
				link.ID = query.Get("id")
				if !digits.MatchString(link.ID) {
					routeErr = errors.New("请使用网易云音乐的公开歌单分享链接")
					continue
				}
			}
		}
		if link.ID == "" {
			if provider == Qishui {
				link.ID = query.Get("playlist_id")
			} else if provider == Kuwo {
				link.ID = query.Get("pid")
				if link.ID == "" {
					link.ID = query.Get("id")
				}
			} else if provider == Migu {
				link.ID = query.Get("playlistId")
			} else if provider == Bodian {
				link.ID = query.Get("playlistId")
				if link.ID == "" {
					link.ID = query.Get("playListId")
				}
				if link.ID == "" {
					link.ID = query.Get("pid")
				}
			} else {
				link.ID = query.Get("id")
			}
		}
		if provider == Qishui {
			host := strings.ToLower(u.Hostname())
			switch host {
			case "qishui.douyin.com":
				if !qishuiShortSharePath.MatchString(path) {
					routeErr = errors.New("请使用汽水音乐的公开歌单分享链接")
					continue
				}
				// The public short host carries no playlist ID until it redirects.
				link.ID = ""
			case "music.douyin.com":
				if !qishuiPlaylistSharePath.MatchString(path) || !digits.MatchString(link.ID) {
					routeErr = errors.New("请使用汽水音乐的公开歌单分享链接")
					continue
				}
			case "beta-luna.douyin.com":
				if !qishuiPlaylistDetailPath.MatchString(path) || !digits.MatchString(link.ID) {
					routeErr = errors.New("请使用汽水音乐的公开歌单分享链接")
					continue
				}
			}
		}
		if provider == Kuwo {
			host := strings.ToLower(u.Hostname())
			bangContent := (host == "kuwo.cn" || host == "www.kuwo.cn") && kuwoBangContentPath.MatchString(path)
			if match := kuwoAlbumPath.FindStringSubmatch(path); len(match) > 1 {
				link.Kind = "album"
				link.ID = match[1]
			} else if match := kuwoTrackPath.FindStringSubmatch(path); len(match) > 1 {
				link.Kind = "track"
				link.ID = match[1]
			} else if match := kuwoLegacyTrackPath.FindStringSubmatch(path); len(match) > 1 {
				link.Kind = "track"
				link.ID = match[1]
			} else if match := kuwoPlaylistPath.FindStringSubmatch(path); len(match) > 1 {
				link.ID = match[1]
			} else if match := kuwoMobilePlaylistPath.FindStringSubmatch(path); len(match) > 1 {
				link.ID = match[1]
			} else if match := kuwoRanklistPath.FindStringSubmatch(path); len(match) > 1 {
				link.Kind = "ranklist"
				link.ID = match[1]
			} else if bangContent {
				link.Kind = "ranklist"
				link.ID = query.Get("bangId")
			}
			webHost := host == "kuwo.cn" || host == "www.kuwo.cn" || host == "m.kuwo.cn"
			legacyPlaylistIndex := (host == "kuwo.cn" || host == "www.kuwo.cn") && kuwoLegacyPlaylistIndexPath.MatchString(path)
			shortShare := (host == "t.kuwo.cn" && kuwoShortHostPath.MatchString(path)) || (webHost && kuwoShortSharePath.MatchString(path))
			if shortShare {
				// The short host is accepted only for a compact path and must resolve
				// through the same-provider redirect guard before fetching a playlist.
				link.Kind = "playlist"
				link.ID = ""
			} else {
				routeOK := kuwoAlbumPath.MatchString(path) || kuwoTrackPath.MatchString(path) || kuwoLegacyTrackPath.MatchString(path) || kuwoPlaylistPath.MatchString(path) || kuwoMobilePlaylistPath.MatchString(path) || legacyPlaylistIndex || kuwoRanklistPath.MatchString(path) || bangContent || kuwoPlaylistRootPath.MatchString(path) || kuwoH5PlaylistPath.MatchString(path)
				if link.ID == "" || !routeOK || !digits.MatchString(link.ID) {
					routeErr = errors.New("请使用酷我音乐的公开歌单链接")
					continue
				}
			}
		}
		if provider == Migu {
			if match := miguPlaylistPath.FindStringSubmatch(path); len(match) > 1 {
				link.ID = match[1]
			}
			if match := miguAlbumPath.FindStringSubmatch(path); len(match) > 1 {
				link.Kind = "album"
				link.ID = match[1]
			}
			if link.ID == "" {
				link.ID = query.Get("playlistId")
			}
			routeOK := miguPlaylistPath.MatchString(path) || miguAlbumPath.MatchString(path) || miguV5RootPath.MatchString(path) || (miguHashPlaylistPath.MatchString(path) && miguV5RootPath.MatchString(u.Path))
			if link.ID == "" || !routeOK || !digits.MatchString(link.ID) {
				routeErr = errors.New("请使用咪咕音乐的公开歌单分享链接")
				continue
			}
		}
		if provider == Qianqian {
			if match := qianqianPlaylistPath.FindStringSubmatch(path); len(match) > 1 {
				link.ID = match[1]
			}
			if link.ID == "" || !qianqianPlaylistPath.MatchString(path) || !digits.MatchString(link.ID) {
				routeErr = errors.New("请使用千千音乐的歌单分享链接")
				continue
			}
		}
		if provider == Bodian {
			source := query.Get("source")
			if source == "" {
				source = query.Get("sourceType")
			}
			sourceValid := source == "" || bodianSourceID.MatchString(source)
			if link.ID == "" || !bodianCollectionPath.MatchString(path) || !digits.MatchString(link.ID) || !sourceValid {
				routeErr = errors.New("请使用波点音乐的公开歌单分享链接")
				continue
			}
		}
		if provider == Deezer {
			host := strings.ToLower(u.Hostname())
			if host == "link.deezer.com" {
				if !deezerShortSharePath.MatchString(path) {
					routeErr = errors.New("请使用 Deezer 的公开歌单分享链接")
					continue
				}
				link.ID = ""
			} else if host != "deezer.com" && host != "www.deezer.com" {
				routeErr = errors.New("请使用 Deezer 的公开歌单分享链接")
				continue
			} else {
				if match := deezerPlaylistPath.FindStringSubmatch(path); len(match) > 1 {
					link.Kind = "playlist"
					link.ID = match[1]
				} else if match := deezerAlbumPath.FindStringSubmatch(path); len(match) > 1 {
					link.Kind = "album"
					link.ID = match[1]
				} else if match := deezerTrackPath.FindStringSubmatch(path); len(match) > 1 {
					link.Kind = "track"
					link.ID = match[1]
				} else {
					routeErr = errors.New("请使用 Deezer 的公开歌单分享链接")
					continue
				}
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

		if link.ID != "" && provider == NetEase && !(strings.HasSuffix(path, "/playlist") || playlistPath.MatchString(path)) {
			routeErr = errors.New("请使用网易云音乐的公开歌单分享链接")
			continue
		}
		if link.ID != "" && provider == QQ && !(playlistPath.MatchString(path) || strings.HasSuffix(path, "/taoge.html") || strings.HasSuffix(path, "/playlist.html")) {
			routeErr = errors.New("请使用 QQ 音乐的公开歌单分享链接")
			continue
		}
		if link.Kind == "track" || link.Kind == "album" || link.Kind == "ranklist" {
			routeErr = errors.New("GoMusic 当前仅支持获取公开歌单")
			continue
		}
		validQQResourceID := provider == QQ && (link.Kind == "album" || link.Kind == "track") && qqMID.MatchString(link.ID)
		if link.ID != "" && !digits.MatchString(link.ID) && !validQQResourceID {
			routeErr = errors.New("歌单 ID 无效")
			continue
		}
		return link, nil
	}
	if match := kugouCode.FindStringSubmatch(strings.TrimSpace(input)); len(match) > 1 {
		return Link{Provider: Kugou, Kind: "code", ID: match[1]}, nil
	}
	if routeErr != nil {
		return Link{}, routeErr
	}
	return Link{}, errors.New("请输入音乐平台的官方歌单链接或有效酷狗码")
}
