package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"testing"
)

func TestAppleMusicPublicPlaylistUsesPageDataAndStaysPartial(t *testing.T) {
	page := applePublicPage(t, "Public list", 2, []map[string]any{
		applePageTrack("301", "First", "Artist A", 1),
		applePageTrack("302", "Second", "Artist B", 2),
	})
	service := testService(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "music.apple.com" {
			return nil, fmt.Errorf("unexpected Apple playlist request: %s", r.URL)
		}
		return htmlResponse(r, page), nil
	})

	list, err := service.Fetch(context.Background(), "https://music.apple.com/cn/playlist/public-list/pl.f4d106fed2bd41149aaacabb233eb5eb", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if list.Name != "Public list" || list.SongsCount != 2 || list.TotalCount != 2 || list.Complete || len(list.Warnings) == 0 {
		t.Fatalf("Apple playlist did not preserve counts or partial status: %#v", list)
	}
	if got := []string{list.Tracks[0].ID, list.Tracks[1].ID}; !reflect.DeepEqual(got, []string{"301", "302"}) {
		t.Fatalf("Apple playlist order=%v", got)
	}
}

func applePublicPage(t *testing.T, name string, total int, tracks []map[string]any) string {
	t.Helper()
	trackItems := make([]any, 0, len(tracks))
	for _, track := range tracks {
		trackItems = append(trackItems, track)
	}
	payload := map[string]any{"data": []any{map[string]any{"data": map[string]any{"sections": []any{
		map[string]any{"id": "playlist-detail-header", "items": []any{map[string]any{"title": name}}},
		map[string]any{"id": "playlist-track-list", "trackCount": total, "items": trackItems},
	}}}}}
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf(`<html><head><meta name="description" content="%d songs"></head><body><script type="application/json" id="serialized-server-data">%s</script></body></html>`, total, encoded)
}

func applePageTrack(id, title, artist string, number int) map[string]any {
	return map[string]any{
		"id":          "track-lockup - pl.test - " + id,
		"title":       title,
		"artistName":  artist,
		"trackNumber": number,
		"contentDescriptor": map[string]any{
			"identifiers": map[string]any{"storeAdamID": id},
		},
	}
}
