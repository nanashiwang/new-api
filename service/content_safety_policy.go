package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

const (
	ContentSafetyPolicyWindow               = 30 * 24 * time.Hour
	ContentSafetyPolicyBurstWindow          = 10 * time.Minute
	ContentSafetyPolicyCooldown             = 10 * time.Minute
	ContentSafetyPolicyBurstThreshold       = 3
	ContentSafetyPolicyReviewAfterCooldowns = 3
)

var contentSafetyPolicyCodes = map[string]struct{}{
	"content_filter":           {},
	"content_policy_violation": {},
	"cyber_policy":             {},
	"policy_violation":         {},
	"safety":                   {},
	"safety_policy_violation":  {},
	"safety_violation":         {},
}

func IsContentSafetyPolicyError(err *types.NewAPIError) bool {
	if err == nil {
		return false
	}
	oai := err.ToOpenAIError()
	for _, candidate := range []string{string(err.GetErrorCode()), fmt.Sprintf("%v", oai.Code), oai.Type} {
		if IsContentSafetyPolicyCode(candidate) {
			return true
		}
	}
	return false
}

func IsContentSafetyPolicyCode(code string) bool {
	_, ok := contentSafetyPolicyCodes[strings.ToLower(strings.TrimSpace(code))]
	return ok
}

// NormalizeContentSafetyPolicyError preserves the upstream error while making
// explicit policy decisions terminal for this user request, not channel faults.
func NormalizeContentSafetyPolicyError(err *types.NewAPIError) *types.NewAPIError {
	if !IsContentSafetyPolicyError(err) || types.IsSkipRetryError(err) {
		return err
	}
	normalized := types.WithOpenAIError(err.ToOpenAIError(), err.StatusCode, types.ErrOptionWithSkipRetry())
	normalized.Upstream = err.Upstream
	return normalized
}

func RecordContentSafetyPolicyViolation(c *gin.Context, info *relaycommon.RelayInfo, err *types.NewAPIError) (*model.ContentSafetyEnforcementResult, error) {
	if c == nil || info == nil || !IsContentSafetyPolicyError(err) {
		return nil, nil
	}
	if info.UserId <= 0 {
		return nil, fmt.Errorf("content safety violation missing user identity")
	}

	now := time.Now()
	inputHash, hashErr := hashContentSafetyRequest(c)
	if hashErr != nil {
		// Request identity still provides exact retry deduplication. Keep the
		// input hash empty instead of failing enforcement or retaining content.
		inputHash = ""
	}
	oai := err.ToOpenAIError()
	errorCode := canonicalContentSafetyPolicyCode(err)
	classification := classifyContentSafetyViolation(c, err, errorCode)
	requestID := strings.TrimSpace(info.RequestId)
	if requestID == "" {
		requestID = fmt.Sprintf("generated:%d", info.StartTime.UnixNano())
	}
	eventKey := common.GenerateHMAC(fmt.Sprintf("%d\x00%s\x00%s", info.UserId, requestID, errorCode))

	result, recordErr := model.RecordContentSafetyViolation(model.RecordContentSafetyViolationParams{
		UserId: info.UserId, TokenId: info.TokenId, ChannelId: info.ChannelId,
		RequestId: requestID, EventKey: eventKey, ModelName: info.OriginModelName,
		ErrorType: oai.Type, ErrorCode: errorCode,
		OfficialMessage:   classification.OfficialMessage,
		FineCategory:      classification.FineCategory,
		ReasonSource:      classification.ReasonSource,
		ReasonConfidence:  classification.ReasonConfidence,
		ReasonSummary:     classification.ReasonSummary,
		ClassifierVersion: classification.ClassifierVersion,
		InputHash:         inputHash, IsStream: info.IsStream, CreatedAt: now.Unix(),
		WindowStart:          now.Add(-ContentSafetyPolicyWindow).Unix(),
		BurstWindowStart:     now.Add(-ContentSafetyPolicyBurstWindow).Unix(),
		BurstThreshold:       ContentSafetyPolicyBurstThreshold,
		CooldownSeconds:      int64(ContentSafetyPolicyCooldown.Seconds()),
		ReviewAfterCooldowns: ContentSafetyPolicyReviewAfterCooldowns,
	})
	if recordErr != nil || result == nil || result.Duplicate || result.Violation == nil {
		return result, recordErr
	}

	recordContentSafetyUserNotice(result)
	return result, nil
}

func canonicalContentSafetyPolicyCode(err *types.NewAPIError) string {
	if err == nil {
		return ""
	}
	oai := err.ToOpenAIError()
	for _, candidate := range []string{string(err.GetErrorCode()), fmt.Sprintf("%v", oai.Code), oai.Type} {
		candidate = strings.ToLower(strings.TrimSpace(candidate))
		if _, ok := contentSafetyPolicyCodes[candidate]; ok {
			return candidate
		}
	}
	return ""
}

func hashContentSafetyRequest(c *gin.Context) (string, error) {
	var storage common.BodyStorage
	if value, exists := c.Get(common.KeyBodyStorage); exists {
		storage, _ = value.(common.BodyStorage)
	}
	if storage == nil {
		var err error
		storage, err = common.GetBodyStorage(c)
		if err != nil {
			return "", err
		}
	}
	position, err := storage.Seek(0, io.SeekCurrent)
	if err != nil {
		return "", err
	}
	if _, err = storage.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	defer func() {
		_, _ = storage.Seek(position, io.SeekStart)
	}()

	digest := hmac.New(sha256.New, []byte(common.CryptoSecret))
	if _, err = io.Copy(digest, storage); err != nil {
		return "", err
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func recordContentSafetyUserNotice(result *model.ContentSafetyEnforcementResult) {
	if result == nil || result.Violation == nil {
		return
	}
	count := result.Violation.WindowCount
	var content string
	switch result.Violation.Action {
	case model.ContentSafetyActionCooldownStarted:
		content = fmt.Sprintf("内容安全冷静期已开始：10 分钟内连续 %d 次请求被上游策略拒绝，模型请求将暂停 10 分钟。30 天历史累计 %d 次；不会自动永久停用，重复冷静期将提交管理员复核。", result.Violation.BurstCount, count)
	case model.ContentSafetyActionCooldownActive:
		content = fmt.Sprintf("内容安全事件已记录：该请求与并发请求重叠在已开始的冷静期内。30 天历史累计 %d 次，冷静期不会因此延长。", count)
	default:
		content = fmt.Sprintf("重要内容安全警告：上游明确拒绝了本次请求。当前 10 分钟窗口为 %d/%d，达到 %d 次将进入 10 分钟冷静期；30 天历史累计 %d 次。该记录不等同于认定主观恶意。", result.Violation.BurstCount, ContentSafetyPolicyBurstThreshold, ContentSafetyPolicyBurstThreshold, count)
	}
	model.RecordLog(result.Violation.UserId, model.LogTypeSystem, content)
}
