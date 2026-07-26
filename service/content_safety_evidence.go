package service

import (
	"fmt"
	"html"
	"net/mail"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/types"
	"github.com/bytedance/gopkg/util/gopool"
	"github.com/gin-gonic/gin"
)

const (
	contentSafetyEvidencePurpose      = "content-safety-evidence"
	contentSafetyEvidenceMaxBodyBytes = 2 << 20
	contentSafetyEvidenceMaxTextRunes = 64000
	contentSafetyEmailTemplateVersion = "safety-warning-v1"
)

type contentSafetyEvidenceEnvelope struct {
	Version          string                         `json:"version"`
	ViolationId      int64                          `json:"violation_id"`
	UserId           int                            `json:"user_id"`
	OfficialCode     string                         `json:"official_code"`
	CapturedMessages []ContentSafetyEvidenceMessage `json:"captured_messages"`
}

type ContentSafetyEvidenceMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Index   int    `json:"index"`
}

type ContentSafetyEvidenceView struct {
	OfficialCode     string                         `json:"official_code"`
	CapturedMessages []ContentSafetyEvidenceMessage `json:"captured_messages"`
	CreatedAt        int64                          `json:"created_at"`
	ExpiresAt        int64                          `json:"expires_at"`
}

func GetContentSafetyEvidenceForReview(violationID int64) (*ContentSafetyEvidenceView, error) {
	evidence, violation, err := model.GetContentSafetyEvidenceByViolation(violationID)
	if err != nil {
		return nil, err
	}
	if violation.FineCategory == "child_sexual_content" {
		return nil, fmt.Errorf("该类别证据禁止在普通后台直接展示，请按受限合规流程处理")
	}
	plaintext, err := common.DecryptSensitiveData(evidence.Ciphertext, evidence.Nonce, contentSafetyEvidencePurpose)
	if err != nil {
		return nil, err
	}
	var envelope contentSafetyEvidenceEnvelope
	if err = common.Unmarshal(plaintext, &envelope); err != nil {
		return nil, err
	}
	if envelope.ViolationId != violation.Id || envelope.UserId != violation.UserId || envelope.OfficialCode != violation.ErrorCode {
		return nil, fmt.Errorf("content safety evidence identity mismatch")
	}
	return &ContentSafetyEvidenceView{OfficialCode: envelope.OfficialCode, CapturedMessages: envelope.CapturedMessages, CreatedAt: evidence.CreatedAt, ExpiresAt: evidence.ExpiresAt}, nil
}

func captureContentSafetyEvidence(c *gin.Context, result *model.ContentSafetyEnforcementResult) error {
	if c == nil || result == nil || result.Violation == nil || result.Violation.Id <= 0 {
		return nil
	}
	if !common.HasPersistentCryptoSecret() {
		return fmt.Errorf("persistent CRYPTO_SECRET or SESSION_SECRET is required for evidence encryption")
	}
	violation := result.Violation
	storage, err := common.GetBodyStorage(c)
	if err != nil || storage.Size() <= 0 || storage.Size() > contentSafetyEvidenceMaxBodyBytes {
		return err
	}
	body, err := storage.Bytes()
	if err != nil {
		return err
	}
	var payload any
	if err = common.Unmarshal(body, &payload); err != nil {
		return nil
	}
	messages := extractRoleAwareSafetyMessages(payload)
	if len(messages) == 0 {
		return nil
	}
	envelope := contentSafetyEvidenceEnvelope{
		Version: "role-aware-v1", ViolationId: violation.Id, UserId: violation.UserId,
		OfficialCode: violation.ErrorCode, CapturedMessages: messages,
	}
	plaintext, err := common.Marshal(envelope)
	if err != nil {
		return err
	}
	ciphertext, nonce, err := common.EncryptSensitiveData(plaintext, contentSafetyEvidencePurpose)
	if err != nil {
		return err
	}
	retention := 30 * 24 * time.Hour
	if result.ReviewCase != nil || violation.FineCategory == "child_sexual_content" {
		retention = 90 * 24 * time.Hour
	}
	roles := make([]string, 0, len(messages))
	for _, message := range messages {
		roles = append(roles, message.Role)
	}
	now := time.Now()
	return model.CreateContentSafetyEvidence(&model.ContentSafetyEvidence{
		ViolationId: violation.Id, UserId: violation.UserId, Version: common.EncryptedDataVersion,
		Ciphertext: ciphertext, Nonce: nonce, EvidenceHash: common.GenerateHMAC(string(plaintext)),
		RoleSummary: truncateContentSafetyText(strings.Join(roles, ","), 128), SizeBytes: len(plaintext),
		CreatedAt: now.Unix(), ExpiresAt: now.Add(retention).Unix(),
	})
}

func extractRoleAwareSafetyMessages(payload any) []ContentSafetyEvidenceMessage {
	root, ok := payload.(map[string]any)
	if !ok {
		return nil
	}
	items := make([]ContentSafetyEvidenceMessage, 0, 8)
	for _, key := range []string{"prompt", "query"} {
		if text, ok := root[key].(string); ok && strings.TrimSpace(text) != "" {
			items = append(items, ContentSafetyEvidenceMessage{Role: "user", Content: limitEvidenceText(text), Index: 0})
		}
	}
	for _, key := range []string{"messages", "input"} {
		value, exists := root[key]
		if !exists {
			continue
		}
		if text, ok := value.(string); ok && strings.TrimSpace(text) != "" {
			items = append(items, ContentSafetyEvidenceMessage{Role: "user", Content: limitEvidenceText(text), Index: 0})
			continue
		}
		array, ok := value.([]any)
		if !ok {
			continue
		}
		for index, raw := range array {
			entry, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			role, _ := entry["role"].(string)
			role = strings.ToLower(strings.TrimSpace(role))
			if role == "" {
				role = "unknown"
			}
			content := extractEvidenceContent(entry["content"])
			if content == "" {
				content = extractEvidenceContent(entry["text"])
			}
			if content != "" {
				items = append(items, ContentSafetyEvidenceMessage{Role: role, Content: limitEvidenceText(content), Index: index})
			}
		}
	}
	lastUser := -1
	for index := len(items) - 1; index >= 0; index-- {
		if items[index].Role == "user" {
			lastUser = index
			break
		}
	}
	if lastUser < 0 {
		return nil
	}
	start, end := lastUser-2, lastUser+1
	if start < 0 {
		start = 0
	}
	if end > len(items) {
		end = len(items)
	}
	return items[start:end]
}

func extractEvidenceContent(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case []any:
		parts := make([]string, 0, len(typed))
		for _, raw := range typed {
			entry, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			kind, _ := entry["type"].(string)
			if kind != "" && kind != "text" && kind != "input_text" && kind != "output_text" {
				continue
			}
			if text, ok := entry["text"].(string); ok {
				parts = append(parts, text)
			}
		}
		return strings.TrimSpace(strings.Join(parts, "\n"))
	default:
		return ""
	}
}

func limitEvidenceText(value string) string {
	return truncateContentSafetyText(value, contentSafetyEvidenceMaxTextRunes)
}

func scheduleContentSafetyEmail(result *model.ContentSafetyEnforcementResult) error {
	if result == nil || result.Violation == nil || result.Duplicate || result.Violation.Action == model.ContentSafetyActionCooldownActive {
		return nil
	}
	email, username, err := model.GetContentSafetyNotificationIdentity(result.Violation.UserId)
	if err != nil {
		return err
	}
	recipient, source := validSafetyEmail(email), "email"
	if recipient == "" {
		recipient, source = validSafetyEmail(username), "username"
	}
	if recipient == "" {
		return nil
	}
	violation := result.Violation
	kind := violation.Action
	if result.ReviewCase != nil {
		kind = model.ContentSafetyActionReviewRequired
	}
	deliveryKey := common.GenerateHMAC(fmt.Sprintf("%d\x00%d\x00%s\x00%s", violation.UserId, violation.Id, kind, contentSafetyEmailTemplateVersion))
	now := time.Now().Unix()
	notification := &model.ContentSafetyNotification{
		ViolationId: violation.Id, UserId: violation.UserId, DeliveryKey: deliveryKey,
		Kind: kind, Recipient: recipient, RecipientSource: source,
		TemplateVersion: contentSafetyEmailTemplateVersion, Status: model.ContentSafetyNotificationPending,
		CreatedAt: now, UpdatedAt: now,
	}
	created, err := model.CreateContentSafetyNotification(notification, time.Now().Add(-time.Hour).Unix(), 3)
	if err != nil || !created {
		return err
	}
	gopool.Go(func() { deliverContentSafetyEmail(notification, violation) })
	return nil
}

func validSafetyEmail(raw string) string {
	normalized := common.NormalizeEmailAddress(raw)
	parsed, err := mail.ParseAddress(normalized)
	if err != nil || !strings.EqualFold(parsed.Address, normalized) || len(normalized) > 254 {
		return ""
	}
	return normalized
}

func deliverContentSafetyEmail(notification *model.ContentSafetyNotification, violation *model.ContentSafetyViolation) {
	now := time.Now().Unix()
	claimed, err := model.ClaimContentSafetyNotification(notification.Id, now)
	if err != nil || !claimed {
		return
	}
	requestRef := violation.RequestId
	if len(requestRef) > 12 {
		requestRef = requestRef[len(requestRef)-12:]
	}
	content := fmt.Sprintf("<p>您好：</p><p>上游服务明确拒绝了您的一次请求，官方错误码为 <strong>%s</strong>。</p><p>本站处理：%s；当前短时计数 %d/%d，30 天累计 %d 次。</p><p>请求参考：%s。邮件不包含您的对话正文，也不代表本站对主观意图作出认定。请停止重复提交可能违反上游政策的内容；反复触发将进入冷静期并提交管理员复核。</p>",
		html.EscapeString(violation.ErrorCode), html.EscapeString(contentSafetyActionDescription(notification.Kind)), violation.BurstCount,
		ContentSafetyPolicyBurstThreshold, violation.WindowCount, html.EscapeString(requestRef))
	err = common.SendEmail("重要内容安全警告", notification.Recipient, content)
	status, lastError := model.ContentSafetyNotificationSent, ""
	if err != nil {
		status, lastError = model.ContentSafetyNotificationFailed, sanitizeContentSafetyAuditText(err.Error(), 256)
	}
	_ = model.FinishContentSafetyNotification(notification.Id, status, lastError, time.Now().Unix())
}

func RetryContentSafetyEmails() error {
	before := time.Now().Add(-5 * time.Minute).Unix()
	if err := model.RecoverStaleContentSafetyNotifications(before); err != nil {
		return err
	}
	items, err := model.GetRetryableContentSafetyNotifications(before, 100)
	if err != nil {
		return err
	}
	for index := range items {
		notification := items[index]
		violation, lookupErr := model.GetContentSafetyViolationByID(notification.ViolationId)
		if lookupErr != nil {
			_ = model.FinishContentSafetyNotification(notification.Id, model.ContentSafetyNotificationSkipped, "violation unavailable", time.Now().Unix())
			continue
		}
		gopool.Go(func() { deliverContentSafetyEmail(&notification, violation) })
	}
	return nil
}

func StartContentSafetyMaintenanceTask() {
	gopool.Go(func() {
		for {
			if err := RetryContentSafetyEmails(); err != nil {
				common.SysError("content safety email retry failed: " + sanitizeContentSafetyAuditText(err.Error(), 256))
			}
			if err := model.DeleteExpiredContentSafetyEvidence(time.Now()); err != nil {
				common.SysError("content safety evidence cleanup failed: " + sanitizeContentSafetyAuditText(err.Error(), 256))
			}
			time.Sleep(time.Hour)
		}
	})
}

func EnrichContentSafetyClientError(err *types.NewAPIError, result *model.ContentSafetyEnforcementResult) *types.NewAPIError {
	if err == nil || result == nil || result.Violation == nil {
		return err
	}
	upstream := err.ToOpenAIError()
	violation := result.Violation
	action := violation.Action
	if result.ReviewCase != nil {
		action = model.ContentSafetyActionReviewRequired
	}
	message := fmt.Sprintf("上游内容安全策略已拒绝本次请求（官方代码：%s）。本站警告：短时计数 %d/%d，30 天累计 %d 次；当前处理：%s。请勿重复提交类似内容。",
		violation.ErrorCode, violation.BurstCount, ContentSafetyPolicyBurstThreshold, violation.WindowCount, contentSafetyActionDescription(action))
	metadata, _ := common.Marshal(map[string]any{
		"content_safety": map[string]any{"official_code": violation.ErrorCode, "action": action,
			"burst_count": violation.BurstCount, "burst_threshold": ContentSafetyPolicyBurstThreshold,
			"window_count": violation.WindowCount, "local_category": violation.FineCategory,
			"local_category_source": violation.ReasonSource, "local_category_confidence": violation.ReasonConfidence},
	})
	upstream.Message = message
	upstream.Metadata = metadata
	enriched := types.WithOpenAIError(upstream, err.StatusCode, types.ErrOptionWithSkipRetry())
	enriched.Upstream = err.Upstream
	return enriched
}

func contentSafetyActionDescription(action string) string {
	switch action {
	case model.ContentSafetyActionCooldownStarted:
		return "已进入 10 分钟冷静期"
	case model.ContentSafetyActionCooldownActive:
		return "冷静期内的重复请求已记录，冷静期不延长"
	case model.ContentSafetyActionReviewRequired:
		return "已提交管理员人工复核"
	default:
		return "正式警告"
	}
}
