package db

import (
	"GoMusic/internal/usage"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"
)

func usageIntegrationStore(t *testing.T, dates ...string) (*Store, string) {
	t.Helper()
	dsn := os.Getenv("GOMUSIC_INTEGRATION_MYSQL_DSN")
	if dsn == "" {
		t.Skip("set GOMUSIC_INTEGRATION_MYSQL_DSN for isolated usage persistence tests")
	}
	store, err := Open(context.Background(), dsn, time.Nanosecond)
	if err != nil {
		t.Fatal(err)
	}
	cleanup, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, day := range dates {
			if _, err := cleanup.Exec("DELETE FROM gomusic_usage_visitors WHERE day = ?", day); err != nil {
				t.Error(err)
			}
			if _, err := cleanup.Exec("DELETE FROM gomusic_usage_daily WHERE day = ?", day); err != nil {
				t.Error(err)
			}
		}
		cleanup.Close()
		store.Close()
	})
	return store, dsn
}

func usageHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func TestUsagePersistenceAndPeriodVisitorDedupIntegration(t *testing.T) {
	// Isolated test dates predate this application's usage collection. Current
	// browser activity cannot alter the all-time-as-of-2001 assertions.
	store, dsn := usageIntegrationStore(t, "2001-01-01", "2001-01-02")
	ctx := context.Background()
	a := usageHash(fmt.Sprintf("visitor-a-%d", time.Now().UnixNano()))
	b := usageHash(fmt.Sprintf("visitor-b-%d", time.Now().UnixNano()))
	parse := func(value string) time.Time {
		at, err := time.Parse(time.RFC3339, value)
		if err != nil {
			t.Fatal(err)
		}
		return at
	}
	for _, event := range []usage.Event{
		{At: parse("2001-01-01T00:00:00+08:00"), VisitorHash: a, Succeeded: true, Songs: 3},
		{At: parse("2001-01-02T00:00:00+08:00"), VisitorHash: a, Succeeded: true, Partial: true, Songs: 5},
		{At: parse("2001-01-02T23:59:59+08:00"), VisitorHash: b, Succeeded: false, Songs: 0},
	} {
		if err := store.RecordUsage(ctx, event); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(ctx, dsn, time.Nanosecond)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	for _, tc := range []struct {
		period, date              string
		requests, songs, visitors int64
	}{
		{"day", "2001-01-01T12:00:00+08:00", 1, 3, 1},
		{"day", "2001-01-02T12:00:00+08:00", 2, 5, 2},
		{"week", "2001-01-02T12:00:00+08:00", 3, 8, 2},
		{"month", "2001-01-02T12:00:00+08:00", 3, 8, 2},
		{"all", "2001-01-02T12:00:00+08:00", 3, 8, 2},
	} {
		t.Run(tc.period+tc.date, func(t *testing.T) {
			window, _ := usage.Period(tc.period, parse(tc.date))
			got, err := reopened.UsageStats(ctx, window)
			if err != nil {
				t.Fatal(err)
			}
			if got.Requests != tc.requests || got.Songs != tc.songs || got.UniqueVisitors != tc.visitors {
				t.Fatalf("got %#v; want requests=%d songs=%d visitors=%d", got, tc.requests, tc.songs, tc.visitors)
			}
			if tc.period != "day" && (got.Successes != 2 || got.Failures != 1 || got.PartialResults != 1) {
				t.Fatalf("wrong outcome counts: %#v", got)
			}
			if tc.period == "all" && got.StartDate != "2001-01-01" {
				t.Fatalf("lost first collection date: %#v", got)
			}
		})
	}
}

func TestUsageConcurrentIncrementsAndValidationIntegration(t *testing.T) {
	store, _ := usageIntegrationStore(t, "2002-02-02")
	ctx := context.Background()
	at, _ := time.Parse(time.RFC3339, "2002-02-02T23:30:00+08:00")
	event := usage.Event{At: at, VisitorHash: usageHash("same-client"), Succeeded: true, Songs: 7}
	const requests = 50
	errs := make(chan error, requests)
	var group sync.WaitGroup
	for i := 0; i < requests; i++ {
		group.Add(1)
		go func() { defer group.Done(); errs <- store.RecordUsage(ctx, event) }()
	}
	group.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	window, _ := usage.Period("day", at)
	got, err := store.UsageStats(ctx, window)
	if err != nil || got.Requests != requests || got.Successes != requests || got.Songs != requests*7 || got.UniqueVisitors != 1 {
		t.Fatalf("lost concurrent writes: %#v err=%v", got, err)
	}
	for _, bad := range []usage.Event{
		{At: at, VisitorHash: "bad", Succeeded: true, Songs: 1},
		{At: at, VisitorHash: event.VisitorHash, Succeeded: true, Songs: -1},
		{At: at, VisitorHash: event.VisitorHash, Succeeded: false, Songs: 1},
		{At: at, VisitorHash: event.VisitorHash, Succeeded: false, Partial: true},
		{VisitorHash: event.VisitorHash, Succeeded: true, Songs: 1},
	} {
		if err := store.RecordUsage(ctx, bad); err == nil {
			t.Fatalf("invalid event accepted: %#v", bad)
		}
	}
	got, _ = store.UsageStats(ctx, window)
	if got.Requests != requests || got.Songs != requests*7 {
		t.Fatal("invalid event changed counters")
	}
}
