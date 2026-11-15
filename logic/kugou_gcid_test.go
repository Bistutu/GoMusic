package logic

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

func TestKugouGCIDIdentityDecoding(t *testing.T) {
	for _, tc := range []struct {
		name, input string
		owner       int64
		want        string
	}{
		{"user-share", "gcid_3zbivkavz4z072", 606105881, "collection_3_606105881_4_0"},
		{"second-public-pair", "gcid_3z17seve8z2z051", 2248574973, "collection_3_2248574973_2_0"},
		{"missing-owner", "gcid_3zbivkavz4z072", 0, ""},
		{"negative-owner", "gcid_3zbivkavz4z072", -1, ""},
		{"different-owner", "gcid_3zbivkavz4z072", 606105882, ""},
		{"missing-prefix", "3zbivkavz4z072", 606105881, ""},
		{"extra-field", "gcid_3zbivkavz4z072z0", 606105881, ""},
		{"missing-field", "gcid_3zbivkavz4", 606105881, ""},
		{"overflow", "gcid_3zyyyyyyyyyyyyyyyyyyyyz4z072", 606105881, ""},
		{"zero-list", "gcid_3zbivkavz0z072", 606105881, ""},
		{"noncanonical-owner", "gcid_3z0bivkavz4z072", 606105881, ""},
		{"noncanonical-list", "gcid_3zbivkavz04z072", 606105881, ""},
		{"short-trailer", "gcid_3zbivkavz4z07", 606105881, ""},
		{"unsupported-kind", "gcid_1zbivkavz4z072", 606105881, ""},
		{"unsupported-tag", "gcid_3zbivkavz4z172", 606105881, ""},
		{"invalid-trailer", "gcid_3zbivkavz4z0!!", 606105881, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := kugouCollectionFromShare(tc.input, tc.owner); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestKugouEncodedShareRetrievesAndVerifiesPublicCollection(t *testing.T) {
	for _, tc := range []struct {
		name, gcid, behavior string
		owner                int64
		total, limit, want   int
		complete, preview    bool
		fetch                bool
	}{
		{"original-twelve", "gcid_3zbivkavz4z072", "full", 606105881, 12, 10000, 12, true, false, true},
		{"case-insensitive-hashes", "gcid_3zbivkavz4z072", "lowercase", 606105881, 12, 10000, 12, true, false, true},
		{"large-2800", "gcid_3zbivkavz4z072", "full", 606105881, 2800, 10000, 2800, true, false, true},
		{"export-limit", "gcid_3zbivkavz4z072", "full", 606105881, 2800, 500, 500, false, false, true},
		{"missing-total-recovered", "gcid_3zbivkavz4z072", "missing-total", 606105881, 12, 10000, 12, true, false, true},
		{"owner-mismatch", "gcid_3zbivkavz4z072", "full", 606105882, 12, 10000, 10, false, true, false},
		{"owner-missing", "gcid_3zbivkavz4z072", "full", 0, 12, 10000, 10, false, true, false},
		{"unsupported-shape", "gcid_example12345", "full", 606105881, 12, 10000, 10, false, true, false},
		{"unsupported-tag", "gcid_3zbivkavz4z172", "full", 606105881, 12, 10000, 10, false, true, false},
		{"different-prefix", "gcid_3zbivkavz4z072", "different", 606105881, 12, 10000, 10, false, true, true},
		{"reordered-prefix", "gcid_3zbivkavz4z072", "reorder", 606105881, 12, 10000, 10, false, true, true},
		{"missing-preview-id", "gcid_3zbivkavz4z072", "missing-id", 606105881, 12, 10000, 10, false, true, true},
		{"collection-fails", "gcid_3zbivkavz4z072", "failure", 606105881, 12, 10000, 10, false, true, true},
		{"collection-stops-at-eleven", "gcid_3zbivkavz4z072", "partial", 606105881, 12, 10000, 11, false, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			songs := func(start, count int) []map[string]any {
				items := regressionSongs("kugou", start, count)
				for i, song := range items {
					song["hash"] = fmt.Sprintf("%032X", start+i)
				}
				return items
			}
			service := testService(func(r *http.Request) (*http.Response, error) {
				if strings.Contains(r.URL.Path, "/songlist/") {
					info := map[string]any{"name": "Original shared playlist", "list_create_userid": tc.owner, "specialid": 0}
					if tc.behavior != "missing-total" {
						info["count"] = tc.total
					}
					preview := songs(1, 10)
					if tc.behavior == "missing-id" {
						preview[0]["hash"] = ""
					}
					return regressionHTML(r, "window.$output", map[string]any{"global_collection_id": "", "info": map[string]any{"listinfo": info, "songs": preview}}), nil
				}
				calls++
				if !tc.fetch {
					t.Fatalf("unvalidated share caused request %s", r.URL)
				}
				if strings.Contains(r.URL.Path, "/info") {
					if r.URL.Query().Get("global_specialid") != "collection_3_606105881_4_0" {
						t.Fatalf("wrong public identity: %s", r.URL)
					}
					return response(r, map[string]any{"status": 1, "data": map[string]any{"specialname": "Full metadata name", "songcount": tc.total}}), nil
				}
				if r.URL.Query().Get("global_collection_id") != "collection_3_606105881_4_0" {
					t.Fatal("wrong collection ID")
				}
				page, _ := strconv.Atoi(r.URL.Query().Get("page"))
				if tc.behavior == "failure" || (tc.behavior == "partial" && page > 1) {
					return nil, fmt.Errorf("collection unavailable")
				}
				start := (page-1)*100 + 1
				n := max(0, min(100, tc.total-start+1))
				if tc.behavior == "partial" {
					n = 11
				}
				tracks := songs(start, n)
				if page == 1 {
					switch tc.behavior {
					case "different":
						tracks[0]["hash"] = "OTHER"
					case "reorder":
						tracks[0], tracks[1] = tracks[1], tracks[0]
					case "lowercase":
						for _, track := range tracks {
							track["hash"] = strings.ToLower(track["hash"].(string))
						}
					}
				}
				return regressionKugouPage(r, tc.total, tracks), nil
			})
			service.MaxSongs = tc.limit
			list, err := service.Fetch(context.Background(), "https://m.kugou.com/songlist/"+tc.gcid+"/", Options{})
			if err != nil {
				t.Fatal(err)
			}
			if list.SongsCount != tc.want || list.TotalCount != tc.total || list.Complete != tc.complete || list.Preview != tc.preview || list.Name != "Original shared playlist" {
				t.Fatalf("got %d/%d complete=%v preview=%v name=%q warnings=%v", list.SongsCount, list.TotalCount, list.Complete, list.Preview, list.Name, list.Warnings)
			}
			if tc.fetch != (calls > 0) {
				t.Fatalf("calls=%d; expected public fetch=%v", calls, tc.fetch)
			}
		})
	}
}
