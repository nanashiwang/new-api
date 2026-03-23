package service

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/logger"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

func FinalizeConsumeLogAfterSettle(logContent string, other map[string]interface{}, actualQuota int, relayInfo *relaycommon.RelayInfo, settleErr error) (int, string, map[string]interface{}) {
	loggedQuota := actualQuota
	if settleErr == nil || relayInfo == nil || relayInfo.BillingSource != BillingSourceToken {
		return loggedQuota, logContent, other
	}

	loggedQuota = relayInfo.FinalPreConsumedQuota
	if loggedQuota < 0 {
		loggedQuota = 0
	}
	if loggedQuota == actualQuota {
		return loggedQuota, logContent, other
	}

	if other == nil {
		other = make(map[string]interface{})
	}
	other["actual_quota"] = actualQuota
	other["charged_quota"] = loggedQuota
	other["settle_failed"] = true
	other["settle_error"] = settleErr.Error()

	var note string
	if loggedQuota < actualQuota {
		note = fmt.Sprintf("实际消耗 %s，但令牌仅成功扣费 %s（超额部分未成功结算）", logger.FormatQuota(actualQuota), logger.FormatQuota(loggedQuota))
	} else {
		note = fmt.Sprintf("实际消耗 %s，但令牌仍按 %s 扣费（返还未成功）", logger.FormatQuota(actualQuota), logger.FormatQuota(loggedQuota))
	}

	if strings.TrimSpace(logContent) == "" {
		logContent = note
	} else {
		logContent += "，" + note
	}
	return loggedQuota, logContent, other
}
