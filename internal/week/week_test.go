package week

import (
	"testing"
	"time"
)

func localDate(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.Local)
}

func TestParseWeek(t *testing.T) {
	cases := map[string]time.Time{
		"2026-W01": localDate(2025, time.December, 29),
		"2025-W52": localDate(2025, time.December, 22),
		"2020-W53": localDate(2020, time.December, 28),
	}
	for input, want := range cases {
		got, err := ParseWeek(input)
		if err != nil {
			t.Errorf("ParseWeek(%q): %v", input, err)
			continue
		}
		if !got.Equal(want) {
			t.Errorf("ParseWeek(%q) = %s, want %s", input, got, want)
		}
		year, number := got.ISOWeek()
		if WeekString(got) != input {
			t.Errorf("ParseWeek(%q) não preservou a semana: ano=%d semana=%d", input, year, number)
		}
	}
}

func TestParseWeekInvalid(t *testing.T) {
	for _, input := range []string{
		"", "2026-01", "2026-W1", "2026-W00", "2026-W54", "2021-W53", "abc",
	} {
		if _, err := ParseWeek(input); err == nil {
			t.Errorf("ParseWeek(%q) deveria falhar", input)
		}
	}
}

func TestWeekString(t *testing.T) {
	cases := map[time.Time]string{
		localDate(2025, time.December, 29): "2026-W01",
		localDate(2026, time.November, 2):  "2026-W45",
	}
	for date, want := range cases {
		if got := WeekString(date); got != want {
			t.Errorf("WeekString(%s) = %q, want %q", date, got, want)
		}
	}
}

func TestPreviousWeek(t *testing.T) {
	start := localDate(2026, time.January, 5)
	want := localDate(2025, time.December, 29)
	if got := PreviousWeek(start); !got.Equal(want) {
		t.Errorf("PreviousWeek = %s, want %s", got, want)
	}
}

func TestWeekDays(t *testing.T) {
	start := localDate(2025, time.December, 29)
	days := WeekDays(start)
	if len(days) != 7 {
		t.Fatalf("WeekDays devolveu %d dias, want 7", len(days))
	}
	for i, day := range days {
		want := start.AddDate(0, 0, i)
		if !day.Equal(want) {
			t.Errorf("dia %d = %s, want %s", i, day, want)
		}
	}
	if last := days[len(days)-1]; last.Format("2006-01-02") != "2026-01-04" {
		t.Errorf("último dia = %s, want 2026-01-04", last)
	}
}
