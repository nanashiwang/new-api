package ratio_setting

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/types"
)

type TimeRatioRule struct {
	ID         string   `json:"id"`
	Enabled    bool     `json:"enabled"`
	Timezone   string   `json:"timezone,omitempty"`
	Start      string   `json:"start"`
	End        string   `json:"end"`
	Days       []string `json:"days,omitempty"`
	Ratio      float64  `json:"ratio"`
	Models     []string `json:"models,omitempty"`
	Groups     []string `json:"groups,omitempty"`
	UserGroups []string `json:"user_groups,omitempty"`
	Priority   int      `json:"priority,omitempty"`
}

var (
	timeRatioRulesMu sync.RWMutex
	timeRatioRules   []TimeRatioRule
)

func TimeRatioRules2JSONString() string {
	timeRatioRulesMu.RLock()
	defer timeRatioRulesMu.RUnlock()

	if len(timeRatioRules) == 0 {
		return "[]"
	}
	jsonBytes, err := common.Marshal(timeRatioRules)
	if err != nil {
		common.SysError("error marshalling time ratio rules: " + err.Error())
		return "[]"
	}
	return string(jsonBytes)
}

func GetTimeRatioRulesCopy() []TimeRatioRule {
	timeRatioRulesMu.RLock()
	defer timeRatioRulesMu.RUnlock()

	rules := make([]TimeRatioRule, len(timeRatioRules))
	copy(rules, timeRatioRules)
	return rules
}

func UpdateTimeRatioRulesByJSONString(jsonStr string) error {
	rules, err := ParseTimeRatioRules(jsonStr)
	if err != nil {
		return err
	}

	timeRatioRulesMu.Lock()
	defer timeRatioRulesMu.Unlock()
	timeRatioRules = rules
	InvalidateExposedDataCache()
	return nil
}

func ParseTimeRatioRules(jsonStr string) ([]TimeRatioRule, error) {
	if strings.TrimSpace(jsonStr) == "" {
		return nil, nil
	}

	var rules []TimeRatioRule
	if err := common.UnmarshalJsonStr(jsonStr, &rules); err != nil {
		return nil, fmt.Errorf("时间倍率规则必须是 JSON 数组: %w", err)
	}
	if err := validateTimeRatioRules(rules); err != nil {
		return nil, err
	}

	sort.SliceStable(rules, func(i, j int) bool {
		return rules[i].Priority > rules[j].Priority
	})
	return rules, nil
}

func ResolveTimeRatio(modelName, usingGroup, userGroup string, requestTime time.Time) types.TimeRatioInfo {
	if requestTime.IsZero() {
		requestTime = time.Now()
	}

	timeRatioRulesMu.RLock()
	rules := make([]TimeRatioRule, len(timeRatioRules))
	copy(rules, timeRatioRules)
	timeRatioRulesMu.RUnlock()

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		if !matchAnyPattern(rule.Models, modelName) {
			continue
		}
		if !matchAnyPattern(rule.Groups, usingGroup) {
			continue
		}
		if !matchAnyPattern(rule.UserGroups, userGroup) {
			continue
		}
		localTime, timezone, ok := localTimeForRule(requestTime, rule.Timezone)
		if !ok {
			continue
		}
		if !matchRuleTime(rule, localTime) {
			continue
		}
		return types.TimeRatioInfo{
			Ratio:     rule.Ratio,
			RuleID:    rule.ID,
			Timezone:  timezone,
			MatchedAt: localTime.Format(time.RFC3339),
		}
	}

	return types.TimeRatioInfo{Ratio: 1}
}

func validateTimeRatioRules(rules []TimeRatioRule) error {
	seen := make(map[string]struct{}, len(rules))
	for idx, rule := range rules {
		if strings.TrimSpace(rule.ID) == "" {
			return fmt.Errorf("第 %d 条时间倍率规则缺少 id", idx+1)
		}
		if _, ok := seen[rule.ID]; ok {
			return fmt.Errorf("时间倍率规则 id 重复: %s", rule.ID)
		}
		seen[rule.ID] = struct{}{}

		if !rule.Enabled {
			continue
		}
		if rule.Ratio <= 0 {
			return fmt.Errorf("时间倍率规则 %s 的 ratio 必须大于 0", rule.ID)
		}
		if _, err := parseClockMinutes(rule.Start); err != nil {
			return fmt.Errorf("时间倍率规则 %s 的 start 无效: %w", rule.ID, err)
		}
		if _, err := parseClockMinutes(rule.End); err != nil {
			return fmt.Errorf("时间倍率规则 %s 的 end 无效: %w", rule.ID, err)
		}
		if _, _, ok := localTimeForRule(time.Now(), rule.Timezone); !ok {
			return fmt.Errorf("时间倍率规则 %s 的 timezone 无效: %s", rule.ID, rule.Timezone)
		}
		for _, day := range rule.Days {
			if !isWildcard(day) {
				if _, ok := parseWeekday(day); !ok {
					return fmt.Errorf("时间倍率规则 %s 的 days 包含无效值: %s", rule.ID, day)
				}
			}
		}
	}
	return nil
}

func localTimeForRule(t time.Time, timezone string) (time.Time, string, bool) {
	timezone = strings.TrimSpace(timezone)
	if timezone == "" {
		timezone = time.Local.String()
		if timezone == "Local" {
			return t.In(time.Local), timezone, true
		}
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return time.Time{}, timezone, false
	}
	return t.In(loc), timezone, true
}

func matchRuleTime(rule TimeRatioRule, localTime time.Time) bool {
	start, err := parseClockMinutes(rule.Start)
	if err != nil {
		return false
	}
	end, err := parseClockMinutes(rule.End)
	if err != nil {
		return false
	}

	minute := localTime.Hour()*60 + localTime.Minute()
	dayTime := localTime
	if start > end && minute < end {
		dayTime = localTime.AddDate(0, 0, -1)
	}
	if !matchRuleDay(rule.Days, dayTime.Weekday()) {
		return false
	}

	if start == end {
		return true
	}
	if start < end {
		return minute >= start && minute < end
	}
	return minute >= start || minute < end
}

func parseClockMinutes(value string) (int, error) {
	value = strings.TrimSpace(value)
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("必须是 HH:MM 格式")
	}

	hour, ok := parseTwoDigitClockPart(parts[0])
	if !ok || hour > 23 {
		return 0, fmt.Errorf("小时必须是 00-23")
	}
	minute, ok := parseTwoDigitClockPart(parts[1])
	if !ok || minute > 59 {
		return 0, fmt.Errorf("分钟必须是 00-59")
	}
	return hour*60 + minute, nil
}

func parseTwoDigitClockPart(value string) (int, bool) {
	if len(value) != 2 {
		return 0, false
	}
	n := 0
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return 0, false
		}
		n = n*10 + int(ch-'0')
	}
	return n, true
}

func matchRuleDay(days []string, weekday time.Weekday) bool {
	if len(days) == 0 {
		return true
	}
	for _, day := range days {
		if isWildcard(day) {
			return true
		}
		if parsed, ok := parseWeekday(day); ok && parsed == weekday {
			return true
		}
	}
	return false
}

func parseWeekday(day string) (time.Weekday, bool) {
	switch strings.ToLower(strings.TrimSpace(day)) {
	case "sun", "sunday", "0", "7":
		return time.Sunday, true
	case "mon", "monday", "1":
		return time.Monday, true
	case "tue", "tues", "tuesday", "2":
		return time.Tuesday, true
	case "wed", "wednesday", "3":
		return time.Wednesday, true
	case "thu", "thur", "thurs", "thursday", "4":
		return time.Thursday, true
	case "fri", "friday", "5":
		return time.Friday, true
	case "sat", "saturday", "6":
		return time.Saturday, true
	default:
		return time.Sunday, false
	}
}

func matchAnyPattern(patterns []string, value string) bool {
	if len(patterns) == 0 {
		return true
	}
	for _, pattern := range patterns {
		if matchPattern(pattern, value) {
			return true
		}
	}
	return false
}

func matchPattern(pattern, value string) bool {
	pattern = strings.TrimSpace(pattern)
	if isWildcard(pattern) {
		return true
	}
	if !strings.Contains(pattern, "*") {
		return pattern == value
	}

	parts := strings.Split(pattern, "*")
	pos := 0
	for i, part := range parts {
		if part == "" {
			continue
		}
		idx := strings.Index(value[pos:], part)
		if idx < 0 {
			return false
		}
		if i == 0 && !strings.HasPrefix(pattern, "*") && idx != 0 {
			return false
		}
		pos += idx + len(part)
	}
	last := parts[len(parts)-1]
	if last != "" && !strings.HasSuffix(pattern, "*") && !strings.HasSuffix(value, last) {
		return false
	}
	return true
}

func isWildcard(value string) bool {
	return strings.TrimSpace(value) == "*"
}
