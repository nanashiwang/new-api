package model

import "strconv"

const (
	PeriodQuotaDisabledKey = "period_quota_disabled"
	PeriodQuotaUntilKey    = "period_quota_until"
	PeriodQuotaScopeKey    = "period_quota_scope"
	PeriodQuotaScopeRefKey = "period_quota_scope_ref"
)

func SetPeriodQuotaMeta(channel *Channel, scope, scopeRef string, until int64) {
	info := channel.GetOtherInfo()
	info[PeriodQuotaDisabledKey] = true
	info[PeriodQuotaUntilKey] = until
	info[PeriodQuotaScopeKey] = scope
	info[PeriodQuotaScopeRefKey] = scopeRef
	channel.SetOtherInfo(info)
}

func ClearPeriodQuotaMeta(channel *Channel) {
	info := channel.GetOtherInfo()
	delete(info, PeriodQuotaDisabledKey)
	delete(info, PeriodQuotaUntilKey)
	delete(info, PeriodQuotaScopeKey)
	delete(info, PeriodQuotaScopeRefKey)
	channel.SetOtherInfo(info)
}

func HasPeriodQuotaMeta(channel *Channel) bool {
	if channel == nil {
		return false
	}
	v, ok := channel.GetOtherInfo()[PeriodQuotaDisabledKey]
	if !ok {
		return false
	}
	b, ok := v.(bool)
	return ok && b
}

func GetPeriodQuotaUntil(channel *Channel) int64 {
	if channel == nil {
		return 0
	}
	v, ok := channel.GetOtherInfo()[PeriodQuotaUntilKey]
	if !ok {
		return 0
	}
	switch x := v.(type) {
	case int64:
		return x
	case int:
		return int64(x)
	case float64:
		return int64(x)
	case string:
		n, _ := strconv.ParseInt(x, 10, 64)
		return n
	default:
		return 0
	}
}
