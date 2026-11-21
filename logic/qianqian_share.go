package logic

import (
	"GoMusic/misc/models"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func parseQianqianSharePage(body []byte) (*models.SongList, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	var list *models.SongList
	doc.Find("script").EachWithBreak(func(_ int, script *goquery.Selection) bool {
		state, ok := strings.CutPrefix(strings.TrimSpace(script.Text()), "window.__NUXT__")
		if !ok {
			return true
		}
		state, ok = strings.CutPrefix(strings.TrimSpace(state), "=")
		if !ok {
			return true
		}
		list = qianqianNuxtPlaylist(state)
		return list == nil
	})
	if list == nil {
		return nil, fmt.Errorf("千千分享页未返回可读取的歌单")
	}
	return list, nil
}

// Nuxt 2 embeds literal data, optionally wrapped in an IIFE whose arguments
// hold repeated values. Read only data[].pageData and never execute the script.
func qianqianNuxtPlaylist(state string) *models.SongList {
	state = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(state), ";"))
	vars := map[string]string{}
	if definition, ok := strings.CutPrefix(state, "(function("); ok {
		params, tail, ok := strings.Cut(definition, ")")
		if !ok {
			return nil
		}
		body, tail, ok := qianqianBlock(tail, '{', '}')
		if !ok {
			return nil
		}
		args, tail, ok := qianqianBlock(tail, '(', ')')
		if !ok || tail != ")" {
			return nil
		}
		keys, values := strings.Split(params, ","), qianqianParts(args)
		if len(keys) != len(values) {
			return nil
		}
		for i, key := range keys {
			vars[strings.TrimSpace(key)] = values[i]
		}
		body = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(body), ";"))
		state, ok = strings.CutPrefix(body, "return ")
		if !ok {
			return nil
		}
	}
	root := qianqianObject(state)
	for _, rawPage := range qianqianArray(root["data"]) {
		page := qianqianObject(qianqianObject(rawPage)["pageData"])
		rows := qianqianArray(page["trackList"])
		if len(rows) == 0 {
			continue
		}
		list := &models.SongList{Name: qianqianString(json.RawMessage(qianqianLiteral(page["title"], vars)))}
		seen := map[string]bool{}
		for _, raw := range rows {
			song := qianqianObject(raw)
			text := func(key string) string {
				return qianqianString(json.RawMessage(qianqianLiteral(song[key], vars)))
			}
			id := text("TSID")
			if id == "" {
				id = text("assetId")
			}
			if id == "" {
				id = text("id")
			}
			track := models.Track{ID: id, Title: text("title"), Artists: []string{}}
			for _, rawArtist := range qianqianArray(song["artist"]) {
				name := qianqianObject(rawArtist)["name"]
				if name := qianqianString(json.RawMessage(qianqianLiteral(name, vars))); name != "" {
					track.Artists = append(track.Artists, name)
				}
			}
			appendDistinctTrack(list, track, seen)
		}
		if len(list.Tracks) > 0 {
			total, _, _ := qianqianCount(json.RawMessage(qianqianLiteral(page["trackCount"], vars)))
			list.TotalCount = max(total, len(rows))
			return list
		}
	}
	return nil
}

func qianqianLiteral(value string, vars map[string]string) string {
	if literal, ok := vars[value]; ok {
		return literal
	}
	return value
}

func qianqianObject(raw string) map[string]string {
	fields, tail, ok := qianqianBlock(raw, '{', '}')
	if !ok || tail != "" {
		return nil
	}
	object := map[string]string{}
	for _, field := range qianqianParts(fields) {
		index, ok := qianqianValueEnd(field, ':')
		if !ok || index == len(field) {
			continue
		}
		key := strings.TrimSpace(field[:index])
		if strings.HasPrefix(key, "\"") {
			key = qianqianString(json.RawMessage(key))
		}
		object[key] = strings.TrimSpace(field[index+1:])
	}
	return object
}

func qianqianArray(raw string) []string {
	items, tail, ok := qianqianBlock(raw, '[', ']')
	if !ok || tail != "" {
		return nil
	}
	return qianqianParts(items)
}

func qianqianBlock(raw string, open, close byte) (body, tail string, ok bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw[0] != open {
		return "", "", false
	}
	i, ok := qianqianValueEnd(raw[1:], close)
	if !ok || i == len(raw)-1 {
		return "", "", false
	}
	return raw[1 : i+1], strings.TrimSpace(raw[i+2:]), true
}

func qianqianParts(raw string) []string {
	var parts []string
	for raw = strings.TrimSpace(raw); raw != ""; raw = strings.TrimSpace(raw[1:]) {
		i, ok := qianqianValueEnd(raw, ',')
		if !ok {
			return nil
		}
		parts = append(parts, strings.TrimSpace(raw[:i]))
		raw = raw[i:]
		if raw == "" {
			break
		}
	}
	return parts
}

// Find the next delimiter at the current nesting level while skipping quoted
// text, where commas, brackets and field names are ordinary characters.
func qianqianValueEnd(text string, delimiter byte) (int, bool) {
	var closes []byte
	var quote byte
	for i := 0; i < len(text); i++ {
		char := text[i]
		if quote != 0 {
			if char == '\\' {
				i++
			} else if char == quote {
				quote = 0
			}
			continue
		}
		if len(closes) == 0 && char == delimiter {
			return i, true
		}
		switch char {
		case '"', '\'', '`':
			quote = char
		case '(':
			closes = append(closes, ')')
		case '[':
			closes = append(closes, ']')
		case '{':
			closes = append(closes, '}')
		case ')', ']', '}':
			if len(closes) == 0 || closes[len(closes)-1] != char {
				return 0, false
			}
			closes = closes[:len(closes)-1]
		}
	}
	return len(text), len(closes) == 0 && quote == 0
}
