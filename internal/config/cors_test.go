package config

import "testing"

func TestAllowedOriginsValidation(t *testing.T) {
	for _, tc := range []struct {
		name, value string
		count       int
		valid       bool
	}{
		{"same-origin-default", "", 0, true},
		{"explicit-frontends", " https://music.example.test, http://127.0.0.1:5173 ", 2, true},
		{"ipv6", "http://[::1]:5173", 1, true},
		{"wildcard", "*", 0, false},
		{"subdomain-wildcard", "https://*.example.test", 0, false},
		{"null-origin", "null", 0, false},
		{"missing-host", "https://", 0, false},
		{"wrong-scheme", "ftp://example.test", 0, false},
		{"credentials", "https://user:password@example.test", 0, false},
		{"path", "https://example.test/frontend", 0, false},
		{"query", "https://example.test?frontend", 0, false},
		{"fragment", "https://example.test#frontend", 0, false},
		{"invalid-port", "https://example.test:bad", 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := FromEnv(func(key string) string {
				if key == "GOMUSIC_ALLOWED_ORIGINS" {
					return tc.value
				}
				return ""
			})
			if tc.valid != (err == nil) || (tc.valid && len(cfg.AllowedOrigins) != tc.count) {
				t.Fatalf("origins=%v error=%v", cfg.AllowedOrigins, err)
			}
		})
	}
}
