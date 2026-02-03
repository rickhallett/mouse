package cron

import (
	"testing"
	"time"
)

func TestParseSchedule(t *testing.T) {
	parsed, err := parseSchedule("0 8 * * *")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsed.hour != 8 || parsed.minute != 0 {
		t.Fatalf("unexpected schedule parsed")
	}
}

func TestNextSchedule(t *testing.T) {
	parsed, _ := parseSchedule("0 8 * * *")
	base := time.Date(2026, 2, 3, 7, 0, 0, 0, time.UTC)
	next := parsed.next(base)
	if next.Hour() != 8 || next.Minute() != 0 {
		t.Fatalf("unexpected next time")
	}
}
