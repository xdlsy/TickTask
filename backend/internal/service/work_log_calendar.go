package service

import (
	"fmt"
	"time"

	"ticktask/internal/model"
)

// WeeklyRange 返回 ISO 周一 00:00 到下周一 00:00（即周日结束）
func WeeklyRange(t time.Time) (start, end time.Time) {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	start = time.Date(t.Year(), t.Month(), t.Day()-weekday+1, 0, 0, 0, 0, t.Location())
	end = start.AddDate(0, 0, 7)
	return
}

// WeeklyKey "2026-W31"
func WeeklyKey(t time.Time) string {
	year, week := t.ISOWeek()
	return fmt.Sprintf("%d-W%02d", year, week)
}

// MonthlyRange 当月 1 号到下月 1 号
func MonthlyRange(t time.Time) (start, end time.Time) {
	start = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	end = start.AddDate(0, 1, 0)
	return
}

// MonthlyKey "2026-07"
func MonthlyKey(t time.Time) string {
	return t.Format("2006-01")
}

// HalfYearRange H1=1~6 月，H2=7~12 月
func HalfYearRange(t time.Time) (start, end time.Time) {
	month := int(t.Month())
	startYear := t.Year()
	if month <= 6 {
		start = time.Date(startYear, 1, 1, 0, 0, 0, 0, t.Location())
	} else {
		start = time.Date(startYear, 7, 1, 0, 0, 0, 0, t.Location())
	}
	end = start.AddDate(0, 6, 0)
	return
}

// HalfYearKey "2026-H1" or "2026-H2"
func HalfYearKey(t time.Time) string {
	if t.Month() <= 6 {
		return fmt.Sprintf("%d-H1", t.Year())
	}
	return fmt.Sprintf("%d-H2", t.Year())
}

// YearlyRange 自然年
func YearlyRange(t time.Time) (start, end time.Time) {
	start = time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location())
	end = start.AddDate(1, 0, 0)
	return
}

// YearlyKey "2026"
func YearlyKey(t time.Time) string {
	return fmt.Sprintf("%d", t.Year())
}

// RangeForType 根据 type 取 range
func RangeForType(t model.WorkReportType, moment time.Time) (start, end time.Time) {
	switch t {
	case model.ReportWeekly:
		return WeeklyRange(moment)
	case model.ReportMonthly:
		return MonthlyRange(moment)
	case model.ReportHalfYear:
		return HalfYearRange(moment)
	case model.ReportYearly:
		return YearlyRange(moment)
	}
	return moment, moment
}

// KeyForType 根据 type 取 period key
func KeyForType(t model.WorkReportType, moment time.Time) string {
	switch t {
	case model.ReportWeekly:
		return WeeklyKey(moment)
	case model.ReportMonthly:
		return MonthlyKey(moment)
	case model.ReportHalfYear:
		return HalfYearKey(moment)
	case model.ReportYearly:
		return YearlyKey(moment)
	}
	return ""
}

// DateRangeToYMD 把 range 转 YYYY-MM-DD 字符串（end 是 exclusive，转成 inclusive 的最后一天）
func DateRangeToYMD(start, end time.Time) (string, string) {
	return start.Format("2006-01-02"), end.AddDate(0, 0, -1).Format("2006-01-02")
}

// MissingDays 计算周期内有日报的日期与全部日期的差集
func MissingDays(start, end time.Time, existing []string) string {
	existSet := make(map[string]bool, len(existing))
	for _, d := range existing {
		existSet[d] = true
	}
	var missing []string
	for d := start; d.Before(end); d = d.AddDate(0, 0, 1) {
		ds := d.Format("2006-01-02")
		if !existSet[ds] {
			missing = append(missing, ds)
		}
	}
	if len(missing) == 0 {
		return ""
	}
	result := missing[0]
	for _, m := range missing[1:] {
		result += "," + m
	}
	return result
}
