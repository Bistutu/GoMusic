package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

func TestKugouCodeRetrievesPublicPlaylist(t *testing.T) {
	for _, tc := range []struct {
		name, input, behavior        string
		total, returned, limit, want int
		complete, invalid, fail      bool
	}{
		{"original-twelve", "40725013", "collection", 12, 12, 10000, 12, true, false, false},
		{"labelled-code", "酷狗码：40725013", "collection", 12, 12, 10000, 12, true, false, false},
		{"leading-zero", "00725013", "collection", 12, 12, 10000, 12, true, false, false},
		{"other-code", "24681357", "collection", 25, 25, 10000, 25, true, false, false},
		{"large-2800", "40725013", "collection", 2800, 2800, 10000, 2800, true, false, false},
		{"export-limit", "40725013", "collection", 2800, 2800, 500, 500, false, false, false},
		{"smaller-page-total", "40725013", "collection", 2800, 10, 10000, 10, false, false, false},
		{"special-playlist", "40725013", "special", 12, 12, 10000, 12, true, false, false},
		{"inline-complete", "40725013", "inline", 12, 12, 10000, 12, true, false, false},
		{"inline-partial", "40725013", "inline", 12, 10, 10000, 10, false, false, false},
		{"inline-unknown-total", "40725013", "unknown-total", 0, 10, 10000, 10, false, false, false},
		{"expired-code", "40725013", "expired", 0, 0, 10000, 0, false, true, true},
		{"not-a-playlist", "40725013", "single-song", 1, 1, 10000, 0, false, true, true},
		{"missing-info", "40725013", "missing-info", 0, 0, 10000, 0, false, false, true},
		{"untrusted-collection", "40725013", "untrusted", 12, 0, 10000, 0, false, false, true},
		{"bad-json", "40725013", "bad-json", 0, 0, 10000, 0, false, false, true},
		{"upstream-timeout", "40725013", "timeout", 0, 0, 10000, 0, false, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			collection := "collection_3_606105881_4_0"
			if tc.input == "24681357" {
				collection = "collection_3_123456_8_0"
			}
			s := testService(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.URL.Host == "t.kugou.com" && r.URL.Path == "/command/" {
					if r.Method != http.MethodPost || !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
						t.Fatalf("invalid code protocol: %s %s", r.Method, r.Header.Get("Content-Type"))
					}
					var request struct {
						Data string `json:"data"`
					}
					if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
						t.Fatal(err)
					}
					if request.Data != strings.TrimPrefix(tc.input, "酷狗码：") {
						t.Fatalf("wrong code %q", request.Data)
					}
					if tc.behavior == "timeout" {
						return nil, context.DeadlineExceeded
					}
					if tc.behavior == "bad-json" {
						return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("not json")), Header: make(http.Header), Request: r}, nil
					}
					if tc.behavior == "expired" {
						return response(r, map[string]any{"status": 0, "err_code": 10001}), nil
					}
					if tc.behavior == "missing-info" {
						return response(r, map[string]any{"status": 1, "data": map[string]any{}}), nil
					}
					info := map[string]any{"type": 4, "id": "4", "userid": "606105881", "name": "Code playlist", "count": tc.total, "global_collection_id": collection}
					var songs []map[string]any
					switch tc.behavior {
					case "special":
						info["type"], info["id"], info["global_collection_id"] = 2, "42", ""
					case "single-song":
						info["type"] = 1
					case "untrusted":
						info["global_collection_id"] = "https://evil.test/playlist"
					case "inline", "unknown-total":
						delete(info, "global_collection_id")
						songs = regressionSongs("kugou", 1, tc.returned)
						if tc.behavior == "unknown-total" {
							delete(info, "count")
						}
					}
					return response(r, map[string]any{"status": 1, "data": map[string]any{"info": info, "list": songs}}), nil
				}
				if r.URL.Host != "mobiles.kugou.com" && r.URL.Host != "pubsongscdn.kugou.com" && r.URL.Host != "mobilecdn.kugou.com" {
					t.Fatalf("unexpected host %s", r.URL.Host)
				}
				if strings.Contains(r.URL.Path, "/info") {
					return response(r, map[string]any{"status": 1, "data": map[string]any{"specialname": "Collection name", "songcount": tc.returned}}), nil
				}
				if tc.behavior == "special" {
					if r.URL.Query().Get("specialid") != "42" {
						t.Fatal("wrong special ID")
					}
				} else if r.URL.Query().Get("global_collection_id") != collection {
					t.Fatalf("wrong collection: %s", r.URL)
				}
				page, _ := strconv.Atoi(r.URL.Query().Get("page"))
				start := (page-1)*100 + 1
				n := max(0, min(100, tc.returned-start+1))
				return regressionKugouPage(r, tc.returned, regressionSongs("kugou", start, n)), nil
			})
			s.MaxSongs = tc.limit
			list, err := s.Fetch(context.Background(), tc.input, Options{Detailed: true})
			if tc.fail {
				if err == nil || (tc.invalid && !errors.Is(err, ErrInvalidLink)) {
					t.Fatalf("expected failure invalid=%v, got %#v %v", tc.invalid, list, err)
				}
				if tc.behavior == "timeout" && !errors.Is(err, context.DeadlineExceeded) {
					t.Fatal(err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if list.Source != "kugou" || list.Name != "Code playlist" || list.SongsCount != tc.want || list.TotalCount != max(tc.total, tc.want) || list.Complete != tc.complete {
				t.Fatalf("got %d/%d complete=%v name=%q calls=%d", list.SongsCount, list.TotalCount, list.Complete, list.Name, calls)
			}
			if !tc.complete && len(list.Warnings) == 0 {
				t.Fatal("partial result lacks warning")
			}
			if len(list.Tracks) != len(list.Songs) {
				t.Fatal("formatted and raw counts disagree")
			}
			if calls == 0 {
				t.Fatal(fmt.Errorf("code did not reach official API"))
			}
		})
	}
}
