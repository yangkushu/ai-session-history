package cli

import (
	"testing"
	"time"
)

func TestCompactAge(t *testing.T) {
	now := time.Date(2026, 7, 17, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		age  time.Duration
		want string
	}{
		{name: "future", age: -time.Minute, want: "now"},
		{name: "seconds", age: 59 * time.Second, want: "now"},
		{name: "minute", age: time.Minute, want: "1m"},
		{name: "minutes", age: 59 * time.Minute, want: "59m"},
		{name: "hour", age: time.Hour, want: "1h"},
		{name: "hours", age: 23 * time.Hour, want: "23h"},
		{name: "day", age: 24 * time.Hour, want: "1d"},
		{name: "days", age: 29 * 24 * time.Hour, want: "29d"},
		{name: "month", age: 30 * 24 * time.Hour, want: "1mo"},
		{name: "months", age: 364 * 24 * time.Hour, want: "12mo"},
		{name: "year", age: 365 * 24 * time.Hour, want: "1y"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := compactAge(now.Add(-tt.age), now); got != tt.want {
				t.Fatalf("compactAge() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatListTime(t *testing.T) {
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	value := time.Date(2026, 7, 17, 10, 0, 0, 0, time.UTC)
	local := time.FixedZone("UTC+8", 8*60*60)

	if got := formatListTime(&value, now, local); got != "2026-07-17 18:00 (2h)" {
		t.Fatalf("formatListTime() = %q", got)
	}
	if got := formatListTime(nil, now, local); got != "unknown" {
		t.Fatalf("formatListTime(nil) = %q", got)
	}
}

func TestFormatListTimeCalculatesCreatedAndUpdatedAgesIndependently(t *testing.T) {
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	created := now.Add(-25 * time.Hour)
	updated := now.Add(-20 * time.Minute)

	if got := formatListTime(&updated, now, time.UTC); got != "2026-07-17 11:40 (20m)" {
		t.Fatalf("updated time = %q", got)
	}
	if got := formatListTime(&created, now, time.UTC); got != "2026-07-16 11:00 (1d)" {
		t.Fatalf("created time = %q", got)
	}
}
