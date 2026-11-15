package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config contains process configuration. Empty storage addresses enable stateless mode.
type Config struct {
	Addr, StaticDir, MySQLDSN, RedisAddr, RedisPassword string
	RedisDB, MaxSongs, Concurrency                      int
	RequestTimeout, UpstreamTimeout, CacheTTL           time.Duration
	AllowedOrigins                                      []string
}

func Load() (Config, error) { return FromEnv(os.Getenv) }

func FromEnv(get func(string) string) (Config, error) {
	c := Config{Addr: ":8081", StaticDir: "static/dist", MaxSongs: 10000, Concurrency: 4,
		RequestTimeout: 90 * time.Second, UpstreamTimeout: 12 * time.Second, CacheTTL: 30 * 24 * time.Hour}
	for key, dest := range map[string]*string{"GOMUSIC_ADDR": &c.Addr, "GOMUSIC_STATIC_DIR": &c.StaticDir,
		"MYSQL_DSN": &c.MySQLDSN, "REDIS_ADDR": &c.RedisAddr, "REDIS_PASSWORD": &c.RedisPassword} {
		if value := strings.TrimSpace(get(key)); value != "" {
			*dest = value
		}
	}
	if _, _, err := net.SplitHostPort(c.Addr); err != nil {
		return c, fmt.Errorf("GOMUSIC_ADDR must be host:port: %w", err)
	}
	for key, dest := range map[string]*int{"REDIS_DB": &c.RedisDB, "GOMUSIC_MAX_SONGS": &c.MaxSongs, "GOMUSIC_CONCURRENCY": &c.Concurrency} {
		if value := get(key); value != "" {
			n, err := strconv.Atoi(value)
			if err != nil || n < 0 || (key != "REDIS_DB" && n == 0) {
				return c, fmt.Errorf("invalid %s", key)
			}
			*dest = n
		}
	}
	if c.MaxSongs > 10000 || c.Concurrency > 16 {
		return c, fmt.Errorf("max songs must be <= 10000 and concurrency <= 16")
	}
	for key, dest := range map[string]*time.Duration{"GOMUSIC_REQUEST_TIMEOUT": &c.RequestTimeout,
		"GOMUSIC_UPSTREAM_TIMEOUT": &c.UpstreamTimeout, "GOMUSIC_CACHE_TTL": &c.CacheTTL} {
		if value := get(key); value != "" {
			d, err := time.ParseDuration(value)
			if err != nil || d <= 0 {
				return c, fmt.Errorf("invalid %s", key)
			}
			*dest = d
		}
	}
	if get("GOMUSIC_DISABLE_DB") == "1" {
		c.MySQLDSN = ""
	}
	if get("GOMUSIC_DISABLE_REDIS") == "1" {
		c.RedisAddr = ""
	}
	for _, origin := range strings.Split(get("GOMUSIC_ALLOWED_ORIGINS"), ",") {
		if origin = strings.TrimSpace(origin); origin != "" {
			c.AllowedOrigins = append(c.AllowedOrigins, origin)
		}
	}
	return c, nil
}
