// Package usage defines persistent, anonymous playlist usage counters.
package usage

import (
	"context"
	"fmt"
	"time"
	_ "time/tzdata"
)

var Shanghai = func() *time.Location {
	zone, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		panic(err)
	}
	return zone
}()

type Event struct {
	At          time.Time
	VisitorHash string
	Succeeded   bool
	Partial     bool
	Songs       int64
}

type Totals struct {
	Requests       int64 `json:"requests"`
	Successes      int64 `json:"successes"`
	Failures       int64 `json:"failures"`
	PartialResults int64 `json:"partial_results"`
	Songs          int64 `json:"songs"`
	UniqueVisitors int64 `json:"unique_visitors"`
}

type Range struct {
	Period     string
	Start, End time.Time // UTC calendar dates for MySQL DATE columns; End is exclusive.
}

type Result struct {
	Period    string `json:"period"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Totals
}

type Repository interface {
	RecordUsage(context.Context, Event) error
	UsageStats(context.Context, Range) (Result, error)
}

// Day first selects the Beijing calendar date, then represents it in UTC for storage.
func Day(at time.Time) time.Time {
	year, month, day := at.In(Shanghai).Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func Period(period string, at time.Time) (Range, error) {
	day := Day(at)
	r := Range{Period: period, Start: day, End: day.AddDate(0, 0, 1)}
	switch period {
	case "day":
	case "week":
		r.Start = day.AddDate(0, 0, -(int(day.Weekday())+6)%7)
		r.End = r.Start.AddDate(0, 0, 7)
	case "month":
		r.Start = time.Date(day.Year(), day.Month(), 1, 0, 0, 0, 0, time.UTC)
		r.End = r.Start.AddDate(0, 1, 0)
	case "all":
		r.Start = time.Time{}
	default:
		return Range{}, fmt.Errorf("period 必须为 day、week、month 或 all")
	}
	return r, nil
}
