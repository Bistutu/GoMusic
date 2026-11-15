package db

import (
	"GoMusic/misc/models"
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestMySQLMetadataIntegration(t *testing.T) {
	dsn := os.Getenv("GOMUSIC_INTEGRATION_MYSQL_DSN")
	if dsn == "" {
		t.Skip("set GOMUSIC_INTEGRATION_MYSQL_DSN for an isolated test database")
	}
	ctx := context.Background()
	store, err := Open(ctx, dsn, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	provider := fmt.Sprintf("t%x", time.Now().UnixNano()&0xffffffff)
	defer store.db.Where("provider = ?", provider).Delete(&models.SongMetadata{})
	tracks := []models.Track{{ID: "1", Title: strings.Repeat("歌", 700), Artists: []string{"A - B", "C"}}, {ID: "2", Title: strings.Repeat("歌", 700), Artists: []string{"Other"}}}
	if err := store.Put(ctx, provider, tracks); err != nil {
		t.Fatal(err)
	}
	values, err := store.Get(ctx, provider, []string{"1", "2", "missing"})
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 2 || !reflect.DeepEqual(values["1"], tracks[0]) {
		t.Fatalf("metadata damaged: %#v", values)
	}
	tracks[0].Title = "Changed"
	if err := store.Put(ctx, provider, tracks[:1]); err != nil {
		t.Fatal(err)
	}
	values, _ = store.Get(ctx, provider, []string{"1"})
	if values["1"].Title != "Changed" {
		t.Fatal("upsert failed")
	}
	if err := store.db.Model(&models.SongMetadata{}).Where("provider = ? AND id = ?", provider, "1").Update("updated_at", time.Now().Add(-2*time.Hour)).Error; err != nil {
		t.Fatal(err)
	}
	values, _ = store.Get(ctx, provider, []string{"1"})
	if len(values) != 0 {
		t.Fatal("expired metadata returned")
	}
	if err := store.Ping(ctx); err != nil {
		t.Fatal(err)
	}
}
