package logic

import (
	"GoMusic/misc/models"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestFormatTrackPreservesHyphens(t *testing.T) {
	track := models.Track{Title: "A - B（Live）【tag】", Artists: []string{"C - D", "E"}}
	if got := FormatTrack(track, Options{Format: "singer-song"}); got != "C - D / E - A - B (Live)" {
		t.Fatal(got)
	}
	if got := FormatTrack(track, Options{Format: "song", Detailed: true}); got != track.Title {
		t.Fatal(got)
	}
}

// 实测旧接口可能把预览条数当作歌单总数，主接口失败时无法据此确认完整性。
func TestNetEaseLegacyResultWithoutPrimaryTotalIsUnconfirmed(t *testing.T) {
	s := testService(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/api/v6/playlist/detail" {
			return &http.Response{StatusCode: 500, Body: io.NopCloser(strings.NewReader("err")), Header: make(http.Header), Request: r}, nil
		}
		if r.URL.Path == "/api/playlist/detail" {
			return response(r, map[string]any{"code": 200, "result": map[string]any{
				"name": "备用歌单", "trackCount": 2,
				"tracks": []map[string]any{
					{"id": 1, "name": "歌曲甲", "artists": []map[string]string{{"name": "歌手甲"}}},
					{"id": 2, "name": "歌曲乙", "artists": []map[string]string{{"name": "歌手乙"}}},
				},
			}}), nil
		}
		return nil, fmt.Errorf("unexpected path %s", r.URL.Path)
	})
	list, err := s.Fetch(context.Background(), "https://music.163.com/playlist?id=42", Options{})
	if err != nil {
		t.Fatalf("fallback error = %v", err)
	}
	if list.Name != "备用歌单" || list.SongsCount != 2 || list.TotalCount != 2 || list.Complete {
		t.Fatalf("list=%#v", list)
	}
	if len(list.Warnings) != 1 || !strings.Contains(list.Warnings[0], "无法核对") {
		t.Fatalf("warnings=%q", list.Warnings)
	}
	if list.Tracks[0].Title != "歌曲甲" || list.Tracks[1].Artists[0] != "歌手乙" {
		t.Fatalf("tracks=%#v", list.Tracks)
	}
}

// 两个接口都失败时，应保留主接口（v6）的错误，指向真正的歌单访问问题。
func TestNetEaseLegacyFailureKeepsPrimaryError(t *testing.T) {
	s := testService(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/api/v6/playlist/detail" {
			return response(r, map[string]any{"code": 400, "playlist": nil}), nil
		}
		return response(r, map[string]any{"code": 400}), nil
	})
	_, err := s.Fetch(context.Background(), "https://music.163.com/playlist?id=42", Options{})
	if err == nil {
		t.Fatal("expected error when both endpoints fail")
	}
}
