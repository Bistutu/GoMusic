package cache

import (
	"GoMusic/misc/models"
	"context"
	"encoding/json"
	"github.com/redis/go-redis/v9"
	"time"
)

type Store struct {
	client *redis.Client
	ttl    time.Duration
}

func Open(ctx context.Context, addr, password string, database int, ttl time.Duration) (*Store, error) {
	client := redis.NewClient(&redis.Options{Addr: addr, Password: password, DB: database, DialTimeout: time.Second, ReadTimeout: time.Second, WriteTimeout: time.Second, MaxRetries: -1, DialerRetries: 1, ContextTimeoutEnabled: true})
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, err
	}
	return &Store{client: client, ttl: ttl}, nil
}
func key(provider, id string) string { return "gomusic:v2:" + provider + ":song:" + id }
func (s *Store) Get(ctx context.Context, provider string, ids []string) (map[string]models.Track, error) {
	found := make(map[string]models.Track, len(ids))
	if len(ids) == 0 {
		return found, nil
	}
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = key(provider, id)
	}
	values, err := s.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}
	for i, value := range values {
		text, ok := value.(string)
		if !ok {
			continue
		}
		var track models.Track
		if json.Unmarshal([]byte(text), &track) == nil && track.ID == ids[i] && track.Title != "" {
			found[ids[i]] = track
		}
	}
	return found, nil
}
func (s *Store) Put(ctx context.Context, provider string, tracks []models.Track) error {
	if len(tracks) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	pipe := s.client.Pipeline()
	for _, track := range tracks {
		data, err := json.Marshal(track)
		if err != nil {
			return err
		}
		pipe.Set(ctx, key(provider, track.ID), data, s.ttl)
	}
	_, err := pipe.Exec(ctx)
	return err
}
func (s *Store) Ping(ctx context.Context) error { return s.client.Ping(ctx).Err() }
func (s *Store) Close() error                   { return s.client.Close() }
