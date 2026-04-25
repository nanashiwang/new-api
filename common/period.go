package common

import "time"

func CalcPeriodWindow(period string, t time.Time) (start, end int64) {
	return CalcAnchoredPeriodWindow(period, t, 0)
}

func CalcAnchoredPeriodWindow(period string, t time.Time, anchorUnix int64) (start, end int64) {
	local := t.In(time.Local)
	year, month, day := local.Date()
	loc := local.Location()
	dayStart := time.Date(year, month, day, 0, 0, 0, 0, loc)

	switch period {
	case "week":
		anchor := normalizePeriodAnchor(anchorUnix, dayStart)
		return calcAnchoredFixedWindow(local, anchor, 7*24*time.Hour)
	case "month":
		anchor := normalizePeriodAnchor(anchorUnix, dayStart)
		return calcAnchoredMonthlyWindow(local, anchor)
	default:
		return dayStart.Unix(), dayStart.AddDate(0, 0, 1).Unix()
	}
}

func normalizePeriodAnchor(anchorUnix int64, fallback time.Time) time.Time {
	if anchorUnix <= 0 {
		return fallback
	}
	anchor := time.Unix(anchorUnix, 0).In(time.Local)
	year, month, day := anchor.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, anchor.Location())
}

func calcAnchoredFixedWindow(local time.Time, anchor time.Time, duration time.Duration) (start, end int64) {
	if local.Before(anchor) {
		return anchor.Unix(), anchor.Add(duration).Unix()
	}
	elapsed := local.Sub(anchor)
	periods := int64(elapsed / duration)
	startTime := anchor.Add(time.Duration(periods) * duration)
	return startTime.Unix(), startTime.Add(duration).Unix()
}

func calcAnchoredMonthlyWindow(local time.Time, anchor time.Time) (start, end int64) {
	startTime := anchor
	for !startTime.AddDate(0, 1, 0).After(local) {
		startTime = startTime.AddDate(0, 1, 0)
	}
	return startTime.Unix(), startTime.AddDate(0, 1, 0).Unix()
}
