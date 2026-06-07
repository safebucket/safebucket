package helpers

import (
	"strconv"
	"testing"
	"time"
)

func TestResolveActivityRangeDefaults(t *testing.T) {
	start, end, err := ResolveActivityRange(time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	span := end.Sub(start)
	wantSpan := 30 * 24 * time.Hour
	if span < wantSpan-time.Minute || span > wantSpan+time.Minute {
		t.Errorf("expected default span: 30d, got %v", span)
	}

	if end.Location() != time.UTC || start.Location() != time.UTC {
		t.Errorf("expected UTC bounds, got start=%v end=%v", start.Location(), end.Location())
	}
}

func TestResolveActivityRangeValid(t *testing.T) {
	from := time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)

	start, end, err := ResolveActivityRange(from, to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !start.Equal(from) || !end.Equal(to) {
		t.Errorf("expected start=%v end=%v, got start=%v end=%v", from, to, start, end)
	}
}

func TestResolveActivityRangeFromAfterTo(t *testing.T) {
	from := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC)

	if _, _, err := ResolveActivityRange(from, to); err == nil {
		t.Fatal("expected error when from > to")
	}
}

func TestResolveActivityRangeMaxWindow(t *testing.T) {
	to := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)

	withinMax := to.AddDate(0, 0, -maxActivityWindowDays)
	if _, _, err := ResolveActivityRange(withinMax, to); err != nil {
		t.Fatalf("unexpected error for 90d range: %v", err)
	}

	overMax := to.AddDate(0, 0, -(maxActivityWindowDays + 1))
	if _, _, err := ResolveActivityRange(overMax, to); err == nil {
		t.Fatal("expected error when range exceeds 90 days")
	}
}

func TestParseActivityCursor(t *testing.T) {
	now := time.Unix(0, 1_700_000_000_000_000_000).UTC()
	cursor := strconv.FormatInt(now.UnixNano(), 10)

	got, err := ParseActivityCursor(cursor)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Equal(now) {
		t.Errorf("expected %v, got %v", now, got)
	}

	if _, parseErr := ParseActivityCursor("not-a-number"); parseErr == nil {
		t.Fatal("expected error for malformed cursor")
	}
}

func TestPaginateActivity(t *testing.T) {
	rows := []map[string]interface{}{
		{"timestamp": "300"},
		{"timestamp": "200"},
		{"timestamp": "100"},
	}

	data, cursor := PaginateActivity(rows, 2)
	if len(data) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(data))
	}
	if cursor == nil || *cursor != "200" {
		t.Errorf("expected next cursor 200, got %v", cursor)
	}

	data, cursor = PaginateActivity(rows[:2], 2)
	if len(data) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(data))
	}
	if cursor != nil {
		t.Errorf("expected nil cursor on last page, got %v", *cursor)
	}
}
