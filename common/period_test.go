package common

import (
	"testing"
	"time"
)

func withLocal(t *testing.T, loc *time.Location) {
	t.Helper()
	old := time.Local
	time.Local = loc
	t.Cleanup(func() { time.Local = old })
}

func TestCalcPeriodWindowDay(t *testing.T) {
	loc := time.FixedZone("UTC+8", 8*3600)
	withLocal(t, loc)
	start, end := CalcPeriodWindow("day", time.Date(2024, 3, 5, 13, 1, 0, 0, loc))
	if time.Unix(start, 0).In(loc) != time.Date(2024, 3, 5, 0, 0, 0, 0, loc) {
		t.Fatalf("bad day start")
	}
	if time.Unix(end, 0).In(loc) != time.Date(2024, 3, 6, 0, 0, 0, 0, loc) {
		t.Fatalf("bad day end")
	}
}

func TestCalcPeriodWindowWeekStartsMonday(t *testing.T) {
	loc := time.UTC
	withLocal(t, loc)
	start, end := CalcPeriodWindow("week", time.Date(2024, 3, 10, 12, 0, 0, 0, loc))
	if time.Unix(start, 0).In(loc) != time.Date(2024, 3, 4, 0, 0, 0, 0, loc) {
		t.Fatalf("bad week start")
	}
	if time.Unix(end, 0).In(loc) != time.Date(2024, 3, 11, 0, 0, 0, 0, loc) {
		t.Fatalf("bad week end")
	}
}

func TestCalcPeriodWindowMonthLeap(t *testing.T) {
	loc := time.UTC
	withLocal(t, loc)
	start, end := CalcPeriodWindow("month", time.Date(2024, 2, 29, 23, 0, 0, 0, loc))
	if time.Unix(start, 0).In(loc) != time.Date(2024, 2, 1, 0, 0, 0, 0, loc) {
		t.Fatalf("bad month start")
	}
	if time.Unix(end, 0).In(loc) != time.Date(2024, 3, 1, 0, 0, 0, 0, loc) {
		t.Fatalf("bad month end")
	}
}

func TestCalcPeriodWindowDST(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skip(err)
	}
	withLocal(t, loc)
	start, end := CalcPeriodWindow("day", time.Date(2024, 3, 10, 12, 0, 0, 0, loc))
	if time.Unix(start, 0).In(loc) != time.Date(2024, 3, 10, 0, 0, 0, 0, loc) {
		t.Fatalf("bad dst day start")
	}
	if time.Unix(end, 0).In(loc) != time.Date(2024, 3, 11, 0, 0, 0, 0, loc) {
		t.Fatalf("bad dst day end")
	}
	if end-start != 23*3600 {
		t.Fatalf("dst day should be 23h, got %d", end-start)
	}
}
