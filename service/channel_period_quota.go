package service

import (
	"fmt"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"gorm.io/gorm"
)

const (
	periodQuotaScopeChannel = "channel"
	periodQuotaScopeTag     = "tag"
)

var (
	disableChannelPeriodQuota = disablePeriodQuota
	nowChannelPeriodQuota     = time.Now
)

func init() {
	model.PostUpdateChannelUsedQuotaHook = func(channelId, quotaDelta int) {
		RecordChannelPeriodQuota(channelId, int64(quotaDelta), 0)
	}
}

func RecordChannelPeriodCount(channelId int) {
	RecordChannelPeriodQuota(channelId, 0, 1)
}

func RecordChannelPeriodQuota(channelId int, dq, dc int64) {
	if channelId <= 0 || (dq == 0 && dc == 0) {
		return
	}
	if err := recordChannelPeriodQuota(channelId, dq, dc); err != nil {
		common.SysLog(fmt.Sprintf("channel period quota record failed: channel_id=%d, error=%v", channelId, err))
	}
}

func recordChannelPeriodQuota(channelId int, dq, dc int64) error {
	channel, err := model.GetChannelById(channelId, true)
	if err != nil {
		return err
	}
	scope, scopeKey, policy, ok, err := resolveChannelPeriodQuotaPolicy(channel)
	if err != nil || !ok || !policy.IsActive() {
		return err
	}
	start, end := common.CalcAnchoredPeriodWindow(policy.Period, nowChannelPeriodQuota(), policy.AnchorTime)
	if err := model.DeleteStaleChannelQuotaUsage(scope, scopeKey, policy.Period, start, policy.AnchorTime); err != nil {
		return err
	}
	used, count, err := model.IncrChannelQuotaUsage(scope, scopeKey, policy.Period, start, end, dq, dc)
	if err != nil {
		return err
	}
	if !exceedsQuotaPolicy(policy, used, count) {
		return nil
	}
	first, err := model.MarkUsageTriggered(scope, scopeKey, start)
	if err != nil || !first {
		return err
	}
	return disableChannelPeriodQuota(scope, scopeKey, channelId, end)
}

func resolveChannelPeriodQuotaPolicy(channel *model.Channel) (string, string, dto.QuotaPolicy, bool, error) {
	policy := channel.GetSetting().QuotaPolicy
	if policy.IsActive() {
		return periodQuotaScopeChannel, strconv.Itoa(channel.Id), policy, true, nil
	}
	tag := channel.GetTag()
	if tag == "" {
		return "", "", dto.QuotaPolicy{}, false, nil
	}
	policy, found, err := model.GetTagPolicy(tag)
	if err != nil || !found || !policy.IsActive() {
		return "", "", dto.QuotaPolicy{}, false, err
	}
	return periodQuotaScopeTag, tag, policy, true, nil
}

func exceedsQuotaPolicy(policy dto.QuotaPolicy, used, count int64) bool {
	return (policy.QuotaLimit > 0 && used >= policy.QuotaLimit) || (policy.CountLimit > 0 && count >= policy.CountLimit)
}

func disablePeriodQuota(scope, scopeKey string, channelId int, periodEnd int64) error {
	reason := "period quota exceeded"
	if scope == periodQuotaScopeTag {
		return model.AutoDisableChannelsByTagWithReason(scopeKey, reason, periodEnd)
	}
	channel, err := model.GetChannelById(channelId, true)
	if err != nil {
		return err
	}
	info := channel.GetOtherInfo()
	info["status_reason"] = reason
	info["status_time"] = common.GetTimestamp()
	channel.SetOtherInfo(info)
	model.SetPeriodQuotaMeta(channel, periodQuotaScopeChannel, scopeKey, periodEnd)
	channel.Status = common.ChannelStatusAutoDisabled
	if err := channel.SaveWithoutKey(); err != nil {
		return err
	}
	return model.UpdateAbilityStatus(channel.Id, false)
}

func GetCurrentChannelPeriodQuotaUsage(channelId int) (*model.ChannelQuotaUsage, dto.QuotaPolicy, bool, error) {
	channel, err := model.GetChannelById(channelId, true)
	if err != nil {
		return nil, dto.QuotaPolicy{}, false, err
	}
	scope, scopeKey, policy, ok, err := resolveChannelPeriodQuotaPolicy(channel)
	if err != nil || !ok {
		return nil, policy, ok, err
	}
	start, end := common.CalcAnchoredPeriodWindow(policy.Period, nowChannelPeriodQuota(), policy.AnchorTime)
	usage, err := model.GetChannelQuotaUsage(scope, scopeKey, policy.Period, start)
	if err == gorm.ErrRecordNotFound {
		return newZeroPeriodQuotaUsage(scope, scopeKey, policy, start, end), policy, true, nil
	}
	if err != nil {
		return nil, policy, true, err
	}
	if isStalePeriodQuotaUsage(usage, policy) {
		return newZeroPeriodQuotaUsage(scope, scopeKey, policy, start, end), policy, true, nil
	}
	return usage, policy, true, nil
}

func GetCurrentTagPeriodQuotaUsage(tag string) (*model.ChannelQuotaUsage, dto.QuotaPolicy, bool, error) {
	policy, found, err := model.GetTagPolicy(tag)
	if err != nil || !found || !policy.IsActive() {
		return nil, policy, found, err
	}
	start, end := common.CalcAnchoredPeriodWindow(policy.Period, nowChannelPeriodQuota(), policy.AnchorTime)
	usage, err := model.GetChannelQuotaUsage(periodQuotaScopeTag, tag, policy.Period, start)
	if err == gorm.ErrRecordNotFound {
		return newZeroPeriodQuotaUsage(periodQuotaScopeTag, tag, policy, start, end), policy, true, nil
	}
	if err != nil {
		return nil, policy, true, err
	}
	if isStalePeriodQuotaUsage(usage, policy) {
		return newZeroPeriodQuotaUsage(periodQuotaScopeTag, tag, policy, start, end), policy, true, nil
	}
	return usage, policy, true, nil
}

func isStalePeriodQuotaUsage(usage *model.ChannelQuotaUsage, policy dto.QuotaPolicy) bool {
	return usage != nil && policy.AnchorTime > 0 && usage.CreatedAt < policy.AnchorTime
}

func newZeroPeriodQuotaUsage(scope, scopeKey string, policy dto.QuotaPolicy, start, end int64) *model.ChannelQuotaUsage {
	return &model.ChannelQuotaUsage{
		Scope:       scope,
		ScopeKey:    scopeKey,
		Period:      policy.Period,
		PeriodStart: start,
		PeriodEnd:   end,
	}
}
