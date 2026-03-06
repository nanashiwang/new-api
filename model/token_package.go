package model

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	TokenPackagePeriodNone    = "none"
	TokenPackagePeriodDaily   = "daily"
	TokenPackagePeriodWeekly  = "weekly"
	TokenPackagePeriodMonthly = "monthly"
	TokenPackagePeriodCustom  = "custom"

	TokenPackagePeriodModeRelative = "relative"
	TokenPackagePeriodModeNatural  = "natural"
)

func NormalizeTokenPackagePeriod(period string) string {
	switch strings.ToLower(strings.TrimSpace(period)) {
	case TokenPackagePeriodDaily:
		return TokenPackagePeriodDaily
	case TokenPackagePeriodWeekly:
		return TokenPackagePeriodWeekly
	case TokenPackagePeriodMonthly:
		return TokenPackagePeriodMonthly
	case TokenPackagePeriodCustom:
		return TokenPackagePeriodCustom
	default:
		return TokenPackagePeriodNone
	}
}

func NormalizeTokenPackagePeriodMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case TokenPackagePeriodModeNatural:
		return TokenPackagePeriodModeNatural
	default:
		return TokenPackagePeriodModeRelative
	}
}

func calcNextTokenPackageResetTime(base time.Time, period string, customSeconds int64, periodMode string) (int64, error) {
	periodMode = NormalizeTokenPackagePeriodMode(periodMode)
	switch NormalizeTokenPackagePeriod(period) {
	case TokenPackagePeriodDaily:
		if periodMode == TokenPackagePeriodModeRelative {
			return base.Add(24 * time.Hour).Unix(), nil
		}
		next := time.Date(base.Year(), base.Month(), base.Day(), 0, 0, 0, 0, base.Location()).AddDate(0, 0, 1)
		return next.Unix(), nil
	case TokenPackagePeriodWeekly:
		if periodMode == TokenPackagePeriodModeRelative {
			return base.Add(7 * 24 * time.Hour).Unix(), nil
		}
		weekday := int(base.Weekday()) // Sunday=0
		if weekday == 0 {
			weekday = 7
		}
		daysUntil := 8 - weekday // next Monday 00:00
		next := time.Date(base.Year(), base.Month(), base.Day(), 0, 0, 0, 0, base.Location()).AddDate(0, 0, daysUntil)
		return next.Unix(), nil
	case TokenPackagePeriodMonthly:
		if periodMode == TokenPackagePeriodModeRelative {
			return base.AddDate(0, 1, 0).Unix(), nil
		}
		next := time.Date(base.Year(), base.Month(), 1, 0, 0, 0, 0, base.Location()).AddDate(0, 1, 0)
		return next.Unix(), nil
	case TokenPackagePeriodCustom:
		if customSeconds <= 0 {
			return 0, errors.New("package_custom_seconds must be > 0")
		}
		return base.Add(time.Duration(customSeconds) * time.Second).Unix(), nil
	default:
		return 0, nil
	}
}

func ValidateTokenPackageConfig(token *Token) error {
	if token == nil {
		return errors.New("token is nil")
	}
	if !token.PackageEnabled {
		token.PackagePeriod = TokenPackagePeriodNone
		token.PackageLimitQuota = 0
		token.PackageCustomSeconds = 0
		token.PackageUsedQuota = 0
		token.PackageNextResetTime = 0
		return nil
	}

	token.PackagePeriod = NormalizeTokenPackagePeriod(token.PackagePeriod)
	token.PackagePeriodMode = NormalizeTokenPackagePeriodMode(token.PackagePeriodMode)
	if token.PackagePeriod == TokenPackagePeriodNone {
		return errors.New("套餐周期不能为空")
	}
	if token.PackageLimitQuota <= 0 {
		return errors.New("套餐周期额度必须大于 0")
	}
	if token.PackagePeriod == TokenPackagePeriodCustom {
		if token.PackageCustomSeconds <= 0 {
			return errors.New("自定义周期秒数必须大于 0")
		}
	} else {
		token.PackageCustomSeconds = 0
	}
	if token.PackageUsedQuota < 0 {
		token.PackageUsedQuota = 0
	}
	if token.PackageNextResetTime < 0 {
		token.PackageNextResetTime = 0
	}
	return nil
}

func ValidateTokenQuotaPackageRelation(token *Token) error {
	if token == nil {
		return errors.New("token is nil")
	}
	if !token.PackageEnabled || token.UnlimitedQuota {
		return nil
	}
	totalQuota := token.RemainQuota + token.UsedQuota
	if totalQuota < token.PackageLimitQuota {
		return errors.New("总额度不能小于套餐周期额度")
	}
	return nil
}

func MaybeResetTokenPackageState(token *Token, nowUnix int64) (bool, error) {
	if token == nil {
		return false, errors.New("token is nil")
	}
	if !token.PackageEnabled {
		changed := false
		if token.PackagePeriod != TokenPackagePeriodNone {
			token.PackagePeriod = TokenPackagePeriodNone
			changed = true
		}
		if token.PackageCustomSeconds != 0 {
			token.PackageCustomSeconds = 0
			changed = true
		}
		if token.PackageUsedQuota != 0 {
			token.PackageUsedQuota = 0
			changed = true
		}
		if token.PackageNextResetTime != 0 {
			token.PackageNextResetTime = 0
			changed = true
		}
		return changed, nil
	}

	if err := ValidateTokenPackageConfig(token); err != nil {
		return false, err
	}

	now := time.Unix(nowUnix, 0)
	changed := false

	if token.PackageNextResetTime <= 0 {
		nextReset, err := calcNextTokenPackageResetTime(now, token.PackagePeriod, token.PackageCustomSeconds, token.PackagePeriodMode)
		if err != nil {
			return false, err
		}
		if token.PackageUsedQuota != 0 {
			token.PackageUsedQuota = 0
			changed = true
		}
		token.PackageNextResetTime = nextReset
		changed = true
		return changed, nil
	}

	if token.PackageNextResetTime > nowUnix {
		return changed, nil
	}

	switch token.PackagePeriod {
	case TokenPackagePeriodCustom:
		nextReset := token.PackageNextResetTime
		for nextReset > 0 && nextReset <= nowUnix {
			nextReset += token.PackageCustomSeconds
		}
		if nextReset <= nowUnix {
			return false, fmt.Errorf("invalid custom package reset time, next=%d now=%d", nextReset, nowUnix)
		}
		token.PackageNextResetTime = nextReset
	default:
		nextReset := token.PackageNextResetTime
		const maxAdvance = 10000
		for i := 0; i < maxAdvance && nextReset > 0 && nextReset <= nowUnix; i++ {
			next, err := calcNextTokenPackageResetTime(time.Unix(nextReset, 0), token.PackagePeriod, token.PackageCustomSeconds, token.PackagePeriodMode)
			if err != nil {
				return false, err
			}
			if next <= nextReset {
				return false, errors.New("invalid token package next reset time progression")
			}
			nextReset = next
		}
		if nextReset <= nowUnix {
			return false, fmt.Errorf("failed to advance token package reset time, next=%d now=%d", nextReset, nowUnix)
		}
		token.PackageNextResetTime = nextReset
	}

	if token.PackageUsedQuota != 0 {
		token.PackageUsedQuota = 0
	}
	return true, nil
}
