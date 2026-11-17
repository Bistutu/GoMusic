package logic

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestLiveBodianPublicPlaylist(t *testing.T) {
	if os.Getenv("GOMUSIC_LIVE_TEST") != "1" {
		t.Skip("set GOMUSIC_LIVE_TEST=1 for public upstream smoke tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	input := "https://h5app.kuwo.cn/m/bodian/collection.html?uid=45780003&playlistId=86749638&source=5&owerId="
	list, err := NewService(nil, nil, nil, 10000, 4).Fetch(ctx, input, Options{Detailed: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("source=%q name=%q fetched=%d reported=%d complete=%v warnings=%v", list.Source, list.Name, list.SongsCount, list.TotalCount, list.Complete, list.Warnings)
	if list.Source != "bodian" || list.Name == "" || list.SongsCount == 0 || list.SongsCount != list.TotalCount || !list.Complete {
		t.Fatalf("Bodian public playlist result is incomplete: %#v", list)
	}
}
