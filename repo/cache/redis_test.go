package cache

import (
	"GoMusic/misc/models"
	"context"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestRedisMetadataIntegration(t *testing.T) {
	addr := os.Getenv("GOMUSIC_INTEGRATION_REDIS_ADDR")
	if addr == "" {
		t.Skip("set GOMUSIC_INTEGRATION_REDIS_ADDR for an isolated Redis")
	}
	ctx := context.Background()
	store, err := Open(ctx, addr, os.Getenv("GOMUSIC_INTEGRATION_REDIS_PASSWORD"), 0, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	provider := fmt.Sprintf("test-%d", time.Now().UnixNano())
	defer store.client.Del(ctx, key(provider, "1"), key(provider, "2"))
	track := models.Track{ID: "1", Title: "A - B【Live】", Artists: []string{"Artist - Name", "Other"}}
	if err = store.Put(ctx, provider, []models.Track{track}); err != nil {
		t.Fatal(err)
	}
	values, err := store.Get(ctx, provider, []string{"1", "2"})
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 1 || !reflect.DeepEqual(values["1"], track) {
		t.Fatalf("metadata damaged: %#v", values)
	}
	ttl := store.client.PTTL(ctx, key(provider, "1")).Val()
	if ttl <= 0 || ttl > time.Minute {
		t.Fatalf("missing expiration: %v", ttl)
	}
	store.client.Set(ctx, key(provider, "2"), "broken JSON", time.Minute)
	values, err = store.Get(ctx, provider, []string{"2"})
	if err != nil || len(values) != 0 {
		t.Fatal("corrupt cache was accepted")
	}
	store.client.PExpire(ctx, key(provider, "1"), time.Millisecond)
	time.Sleep(5 * time.Millisecond)
	values, err = store.Get(ctx, provider, []string{"1"})
	if err != nil || len(values) != 0 {
		t.Fatal("expired cache was accepted")
	}
	store.Close()
	if _, err = store.Get(ctx, provider, []string{"1"}); err == nil {
		t.Fatal("connection failure hidden")
	}
}
