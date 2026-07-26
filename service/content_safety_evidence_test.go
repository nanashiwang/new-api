package service

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestCaptureContentSafetyEvidenceEncryptsPlaintextAndCanBeReviewed(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.ContentSafetyViolation{}, &model.ContentSafetyEvidence{}))
	originalDB, originalSecret := model.DB, common.CryptoSecret
	model.DB, common.CryptoSecret = db, "content-safety-test-secret"
	t.Setenv("CRYPTO_SECRET", "content-safety-test-secret")
	t.Cleanup(func() { model.DB, common.CryptoSecret = originalDB, originalSecret })

	violation := &model.ContentSafetyViolation{UserId: 42, EventKey: "evidence-test", ErrorCode: "cyber_policy", FineCategory: "malware", CreatedAt: 1}
	require.NoError(t, db.Create(violation).Error)
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	body := `{"messages":[{"role":"assistant","content":"nearby context"},{"role":"user","content":"unique risky evidence text"}]}`
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(body))
	_, err = common.GetBodyStorage(c)
	require.NoError(t, err)
	t.Cleanup(func() { common.CleanupBodyStorage(c) })

	result := &model.ContentSafetyEnforcementResult{Violation: violation}
	require.NoError(t, captureContentSafetyEvidence(c, result))
	evidence, _, err := model.GetContentSafetyEvidenceByViolation(violation.Id)
	require.NoError(t, err)
	require.False(t, bytes.Contains(evidence.Ciphertext, []byte("unique risky evidence text")))
	view, err := GetContentSafetyEvidenceForReview(violation.Id)
	require.NoError(t, err)
	require.Equal(t, "user", view.CapturedMessages[len(view.CapturedMessages)-1].Role)
	require.Equal(t, "unique risky evidence text", view.CapturedMessages[len(view.CapturedMessages)-1].Content)
}

func TestRoleAwareEvidenceKeepsContextButAttributesLatestUser(t *testing.T) {
	payload := map[string]any{"messages": []any{
		map[string]any{"role": "system", "content": "system policy"},
		map[string]any{"role": "user", "content": "benign earlier question"},
		map[string]any{"role": "assistant", "content": "malware keyword in assistant history"},
		map[string]any{"role": "tool", "content": "tool output"},
		map[string]any{"role": "user", "content": "latest user request"},
	}}
	messages := extractRoleAwareSafetyMessages(payload)
	require.Len(t, messages, 3)
	require.Equal(t, "assistant", messages[0].Role)
	require.Equal(t, "tool", messages[1].Role)
	require.Equal(t, "user", messages[2].Role)
	require.Equal(t, "latest user request", messages[2].Content)
}

func TestRoleAwareEvidenceRequiresUserAuthoredInput(t *testing.T) {
	payload := map[string]any{"messages": []any{
		map[string]any{"role": "assistant", "content": "malware"},
		map[string]any{"role": "tool", "content": "dangerous output"},
	}}
	require.Empty(t, extractRoleAwareSafetyMessages(payload))
}

func TestValidSafetyEmailUsesStrictMailbox(t *testing.T) {
	require.Equal(t, "USER@example.com", validSafetyEmail(" USER@example.com "))
	require.Empty(t, validSafetyEmail("Display <user@example.com>"))
	require.Empty(t, validSafetyEmail("not-an-email"))
}

func TestEnrichContentSafetyClientErrorSeparatesOfficialAndLocalFields(t *testing.T) {
	original := types.WithOpenAIError(types.OpenAIError{Message: "rejected", Type: "invalid_request", Code: "cyber_policy"}, http.StatusBadRequest)
	result := &model.ContentSafetyEnforcementResult{Violation: &model.ContentSafetyViolation{
		ErrorCode: "cyber_policy", FineCategory: "malware", ReasonSource: "local_rule", ReasonConfidence: "medium",
		BurstCount: 2, WindowCount: 4, Action: model.ContentSafetyActionWarning,
	}}
	enriched := EnrichContentSafetyClientError(original, result)
	require.True(t, types.IsSkipRetryError(enriched))
	require.Contains(t, enriched.ToOpenAIError().Message, "2/3")
	require.Contains(t, string(enriched.ToOpenAIError().Metadata), `"official_code":"cyber_policy"`)
	require.Contains(t, string(enriched.ToOpenAIError().Metadata), `"local_category":"malware"`)
}
