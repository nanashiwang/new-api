package common

import "time"

func CalcPeriodWindow(period string, t time.Time) (start, end int64) {
	local := t.In(time.Local)
	year, month, day := local.Date()
	loc := local.Location()

	var startTime time.Time
	switch period {
	case "week":
		weekday := int(local.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		startTime = time.Date(year, month, day-weekday+1, 0, 0, 0, 0, loc)
		return startTime.Unix(), startTime.AddDate(0, 0, 7).Unix()
	case "month":
		startTime = time.Date(year, month, 1, 0, 0, 0, 0, loc)
		return startTime.Unix(), startTime.AddDate(0, 1, 0).Unix()
	default:
		startTime = time.Date(year, month, day, 0, 0, 0, 0, loc)
		return startTime.Unix(), startTime.AddDate(0, 0, 1).Unix()
	}
}
