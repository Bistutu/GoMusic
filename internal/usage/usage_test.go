package usage

import (
	"testing"
	"time"
)

func TestBeijingUsageWindows(t *testing.T) {
	for _, tc := range []struct{ at, period, start, end string }{
		{"2026-11-15T15:59:59Z", "day", "2026-11-15", "2026-11-16"},
		{"2026-11-15T16:00:00Z", "day", "2026-11-16", "2026-11-17"},
		{"2026-11-15T15:59:59Z", "week", "2026-11-09", "2026-11-16"},
		{"2026-11-15T16:00:00Z", "week", "2026-11-16", "2026-11-23"},
		{"2026-11-18T03:00:00Z", "week", "2026-11-16", "2026-11-23"},
		{"2026-11-22T05:00:00Z", "week", "2026-11-16", "2026-11-23"},
		{"2026-12-31T15:59:59Z", "month", "2026-12-01", "2027-01-01"},
		{"2026-12-31T16:00:00Z", "month", "2027-01-01", "2027-02-01"},
		{"2027-01-01T00:00:00Z", "week", "2026-12-28", "2027-01-04"},
		{"2027-01-03T16:00:00Z", "week", "2027-01-04", "2027-01-11"},
		{"2024-02-29T04:00:00Z", "month", "2024-02-01", "2024-03-01"},
		{"2026-11-15T16:00:00Z", "all", "0001-01-01", "2026-11-17"},
	} {
		t.Run(tc.period+"/"+tc.at, func(t *testing.T) {
			at, _ := time.Parse(time.RFC3339, tc.at)
			got, err := Period(tc.period, at)
			if err != nil || got.Start.Format("2006-01-02") != tc.start || got.End.Format("2006-01-02") != tc.end {
				t.Fatalf("got %#v err=%v; want %s..%s", got, err, tc.start, tc.end)
			}
		})
	}
	for _, period := range []string{"", "year", "../../", "DAY"} {
		if _, err := Period(period, time.Now()); err == nil {
			t.Fatalf("invalid period accepted: %q", period)
		}
	}
}
