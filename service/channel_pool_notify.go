package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

// NotifyChannelPoolExhausted 多 Key 渠道整池耗尽时通知管理员。
// 应在异步协程中调用（内部会按 Key 数量做 Redis 冷却状态核对）。
// 去重：同一渠道 6 小时内最多通知一次，渠道恢复成功调用后立即解除去重。
func NotifyChannelPoolExhausted(channelId int) {
	setting := operation_setting.GetMonitorSetting()
	if setting == nil || !setting.PoolExhaustedNotifyEnabled {
		return
	}
	channel, err := model.CacheGetChannel(channelId)
	if err != nil || channel == nil || !channel.ChannelInfo.IsMultiKey {
		return
	}
	// 仍有可用 Key（启用且不在冷却/待禁用）则不是整池耗尽。
	if channel.IsEffectivelyAvailable() {
		return
	}
	if !model.TryMarkChannelPoolExhaustedNotified(channelId) {
		return
	}

	keyCount := len(channel.GetKeys())
	cooldownCount := channel.GetMultiKeyCooldownKeyCount()
	subject := fmt.Sprintf("渠道「%s」（#%d）账号池已耗尽", channel.Name, channel.Id)
	content := fmt.Sprintf(
		"渠道「%s」（#%d）的账号池已无可用 Key：共 %d 把，冷却中 %d 把，其余为禁用状态。常见原因为上游账号余额耗尽或被封禁，请检查并充值或更换 Key。流量已自动切换到其他可用渠道。时间：%s",
		channel.Name, channel.Id, keyCount, cooldownCount,
		time.Now().Format("2006-01-02 15:04:05"),
	)

	emails := parsePoolNotifyEmails(setting.PoolExhaustedNotifyEmails)
	if len(emails) == 0 {
		// 未配置邮箱时回退 root 用户的站内通知偏好（邮件/webhook/Bark 等）。
		NotifyRootUser(fmt.Sprintf("channel_pool_exhausted_%d", channelId), subject, content)
		return
	}
	for _, email := range emails {
		if err := common.SendEmail(subject, email, content); err != nil {
			common.SysError(fmt.Sprintf("send pool exhausted email failed: channel=%d email=%s err=%v", channelId, email, err))
		}
	}
	common.SysLog(fmt.Sprintf("channel pool exhausted notified: channel=%d emails=%d", channelId, len(emails)))
}

// parsePoolNotifyEmails 解析多邮箱配置：支持逗号/分号/换行/空格分隔。
func parsePoolNotifyEmails(raw string) []string {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\r' || r == ' ' || r == '，' || r == '；'
	})
	emails := make([]string, 0, len(fields))
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f != "" && strings.Contains(f, "@") {
			emails = append(emails, f)
		}
	}
	return emails
}
