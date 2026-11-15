package logic

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

func TestKugouSharePreviewAndPublicFullListMatrix(t *testing.T) {
	const collection = "collection_3_606105881_2_0"
	for _, tc := range []struct {
		name                  string
		total, preview        int
		kind, behavior        string
		want                  int
		complete, previewOnly bool
	}{
		{"small-complete", 4, 4, "", "", 4, true, false},
		{"owner-missing-ten-of-twelve", 12, 10, "", "", 10, false, true},
		{"large-share-stays-preview", 2800, 10, "", "", 10, false, true},
		{"missing-total", -1, 10, "", "", 10, false, true},
		{"inconsistent-total", 0, 10, "", "", 10, false, true},
		{"root-collection-two-pages", 12, 10, "root", "full", 12, true, false},
		{"nested-collection-two-pages", 12, 10, "nested", "full", 12, true, false},
		{"public-special-two-pages", 12, 10, "special", "full", 12, true, false},
		{"known-total-survives-missing-counts", 12, 10, "root", "missing-counts", 12, true, false},
		{"known-total-survives-smaller-counts", 12, 10, "root", "smaller-counts", 12, true, false},
		{"fallback-failure-keeps-preview", 12, 10, "root", "failure", 10, false, true},
		{"fallback-gets-one-extra-then-fails", 12, 10, "root", "partial", 11, false, false},
		{"repeated-fallback-stops", 12, 10, "root", "repeat", 10, false, true},
		{"invalid-collection-does-not-fetch", 12, 10, "invalid", "", 10, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pages := 0
			service := testService(func(r *http.Request) (*http.Response, error) {
				if strings.Contains(r.URL.Path, "/songlist/") {
					info := map[string]any{"name": "Shared playlist", "specialid": 0}
					if tc.total >= 0 {
						info["count"] = tc.total
					}
					payload := map[string]any{"info": map[string]any{"listinfo": info, "songs": regressionSongs("kugou", 1, tc.preview)}}
					switch tc.kind {
					case "root":
						payload["global_collection_id"] = collection
					case "nested":
						info["global_collection_id"] = collection
					case "special":
						info["specialid"] = 12345
					case "invalid":
						payload["global_collection_id"] = "collection_3_606105881_2_0/../../evil"
					}
					return regressionHTML(r, "window.$output", payload), nil
				}
				if tc.kind == "" || tc.kind == "invalid" {
					t.Errorf("unavailable public identity caused request to %s", r.URL)
				}
				count := 12
				if tc.behavior == "smaller-counts" {
					count = 6
				}
				if strings.Contains(r.URL.Path, "/info") {
					data := map[string]any{"specialname": "Other metadata name"}
					if tc.behavior != "missing-counts" {
						data["songcount"] = count
					}
					return response(r, map[string]any{"status": 1, "data": data}), nil
				}
				pages++
				if pages > 3 {
					return nil, fmt.Errorf("pagination did not stop")
				}
				if tc.behavior == "failure" || (tc.behavior == "partial" && pages == 2) {
					return nil, fmt.Errorf("public fallback unavailable")
				}
				page, _ := strconv.Atoi(r.URL.Query().Get("page"))
				start, n := 1, 10
				if page == 2 {
					start, n = 11, 2
				}
				if tc.behavior == "partial" {
					n = 11
				}
				if tc.behavior == "repeat" {
					start, n = 1, 10
				}
				data := map[string]any{"info": regressionSongs("kugou", start, n)}
				if tc.behavior != "missing-counts" {
					data["count"] = count
				}
				return response(r, map[string]any{"status": 1, "data": data}), nil
			})
			list, err := service.Fetch(context.Background(), "https://m.kugou.com/songlist/gcid_3zbivkavz4z072/?src_cid=3zbivkavz4z072&uid=606105881", Options{})
			if err != nil {
				t.Fatal(err)
			}
			if list.SongsCount != tc.want || list.Complete != tc.complete || list.Preview != tc.previewOnly || list.Name != "Shared playlist" {
				t.Fatalf("got %d/%d complete=%v preview=%v title=%q warnings=%v", list.SongsCount, list.TotalCount, list.Complete, list.Preview, list.Name, list.Warnings)
			}
			if !list.Complete && len(list.Warnings) == 0 {
				t.Fatal("incomplete response lacked a warning")
			}
			if tc.total > 0 && list.TotalCount < tc.total {
				t.Fatalf("known total was reduced: %d", list.TotalCount)
			}
		})
	}
}
