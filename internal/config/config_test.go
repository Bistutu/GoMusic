package config

import "testing"

func TestConfig(t *testing.T) {
	cfg, err := FromEnv(func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MySQLDSN != "" || cfg.RedisAddr != "" {
		t.Fatal("default must be stateless")
	}
	for key, value := range map[string]string{"GOMUSIC_ADDR": "bad", "GOMUSIC_CONCURRENCY": "0", "GOMUSIC_MAX_SONGS": "10001", "REDIS_DB": "-1", "GOMUSIC_REQUEST_TIMEOUT": "-2s", "GOMUSIC_STATS_COOKIE_SECURE": "invalid"} {
		if _, err := FromEnv(func(k string) string {
			if k == key {
				return value
			}
			return ""
		}); err == nil {
			t.Errorf("accepted invalid %s", key)
		}
	}
	cfg, err = FromEnv(func(k string) string {
		return map[string]string{"MYSQL_DSN": "example", "GOMUSIC_DISABLE_DB": "1", "REDIS_ADDR": "localhost:123", "GOMUSIC_DISABLE_REDIS": "1"}[k]
	})
	if err != nil || cfg.MySQLDSN != "" || cfg.RedisAddr != "" {
		t.Fatal("legacy disable flags were ignored")
	}
	cfg, err = FromEnv(func(k string) string {
		return map[string]string{"GOMUSIC_STATS_TOKEN": " local-test-token ", "GOMUSIC_STATS_COOKIE_SECURE": "true"}[k]
	})
	if err != nil || cfg.StatsToken != "local-test-token" || !cfg.StatsCookieSecure {
		t.Fatal("stats configuration was not loaded")
	}
}
