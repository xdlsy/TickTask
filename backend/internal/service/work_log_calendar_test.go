package service

import (
	"testing"
	"time"

	"ticktask/internal/model"
)

func mustParse(t *testing.T, s string) time.Time {
	t.Helper()
	v, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatalf("parse %s: %v", s, err)
	}
	return v
}

func TestWeeklyRange_Midweek(t *testing.T) {
	wed := mustParse(t, "2026-08-04")
	start, end := WeeklyRange(wed)
	if start.Format("2006-01-02") != "2026-08-03" {
		t.Errorf("start = %s, want 2026-08-03 (Mon)", start.Format("2006-01-02"))
	}
	if end.Format("2006-01-02") != "2026-08-10" {
		t.Errorf("end = %s, want 2026-08-10 (next Mon)", end.Format("2006-01-02"))
	}
}

func TestWeeklyRange_Sunday(t *testing.T) {
	sun := mustParse(t, "2026-08-02")
	start, end := WeeklyRange(sun)
	if start.Format("2006-01-02") != "2026-07-27" {
		t.Errorf("start = %s, want 2026-07-27", start.Format("2006-01-02"))
	}
	if end.Format("2006-01-02") != "2026-08-03" {
		t.Errorf("end = %s, want 2026-08-03", end.Format("2006-01-02"))
	}
}

func TestWeeklyKey(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"2026-08-02", "2026-W31"},
		{"2026-08-04", "2026-W32"},
		{"2026-01-01", "2026-W01"},
	}
	for _, c := range cases {
		got := WeeklyKey(mustParse(t, c.in))
		if got != c.want {
			t.Errorf("WeeklyKey(%s) = %s, want %s", c.in, got, c.want)
		}
	}
}

func TestMonthlyRangeAndKey(t *testing.T) {
	jul := mustParse(t, "2026-07-15")
	start, end := MonthlyRange(jul)
	if start.Format("2006-01-02") != "2026-07-01" {
		t.Errorf("start = %s", start)
	}
	if end.Format("2006-01-02") != "2026-08-01" {
		t.Errorf("end = %s", end)
	}
	if MonthlyKey(jul) != "2026-07" {
		t.Errorf("key wrong")
	}
}

func TestHalfYearRange_H1_H2(t *testing.T) {
	jan := mustParse(t, "2026-01-15")
	start, end := HalfYearRange(jan)
	if start.Format("2006-01-02") != "2026-01-01" || end.Format("2006-01-02") != "2026-07-01" {
		t.Errorf("H1 wrong: %s ~ %s", start, end)
	}
	if HalfYearKey(jan) != "2026-H1" {
		t.Errorf("H1 key wrong")
	}

	aug := mustParse(t, "2026-08-15")
	start2, end2 := HalfYearRange(aug)
	if start2.Format("2006-01-02") != "2026-07-01" || end2.Format("2006-01-02") != "2027-01-01" {
		t.Errorf("H2 wrong: %s ~ %s", start2, end2)
	}
	if HalfYearKey(aug) != "2026-H2" {
		t.Errorf("H2 key wrong")
	}
}

func TestYearlyRangeAndKey(t *testing.T) {
	t1 := mustParse(t, "2026-06-15")
	start, end := YearlyRange(t1)
	if start.Format("2006-01-02") != "2026-01-01" || end.Format("2006-01-02") != "2027-01-01" {
		t.Errorf("year range wrong: %s ~ %s", start, end)
	}
	if YearlyKey(t1) != "2026" {
		t.Errorf("year key wrong")
	}
}

func TestMissingDays_AllPresent(t *testing.T) {
	start := mustParse(t, "2026-08-01")
	end := mustParse(t, "2026-08-04")
	existing := []string{"2026-08-01", "2026-08-02", "2026-08-03"}
	if got := MissingDays(start, end, existing); got != "" {
		t.Errorf("got = %q, want empty", got)
	}
}

func TestMissingDays_SomeMissing(t *testing.T) {
	start := mustParse(t, "2026-08-01")
	end := mustParse(t, "2026-08-04")
	existing := []string{"2026-08-02"}
	got := MissingDays(start, end, existing)
	if got != "2026-08-01,2026-08-03" {
		t.Errorf("got = %q", got)
	}
}

func TestRangeForType_AllTypes(t *testing.T) {
	moment := mustParse(t, "2026-08-02")
	for _, ty := range []model.WorkReportType{model.ReportWeekly, model.ReportMonthly, model.ReportHalfYear, model.ReportYearly} {
		s, e := RangeForType(ty, moment)
		if s.IsZero() || e.IsZero() || !e.After(s) {
			t.Errorf("type %v: bad range %v ~ %v", ty, s, e)
		}
		if KeyForType(ty, moment) == "" {
			t.Errorf("type %v: empty key", ty)
		}
	}
}
