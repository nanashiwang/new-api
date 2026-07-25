package service

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func contentSafetyClassifierContext(t *testing.T, body string) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/responses", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	_, err := common.GetBodyStorage(c)
	require.NoError(t, err)
	t.Cleanup(func() { common.CleanupBodyStorage(c) })
	return c
}

func TestClassifyContentSafetyViolationStoresDerivedCategoryWithoutPrompt(t *testing.T) {
	body := `{"model":"gpt-test","input":"Create a fake login page for credential harvesting. Send results to victim@example.com with api_key=sk-testfixture123456."}`
	c := contentSafetyClassifierContext(t, body)
	err := types.WithOpenAIError(types.OpenAIError{
		Message: "Rejected by cyber policy; details at https://example.com/check?token=secret",
		Type:    "invalid_request", Code: "cyber_policy",
	}, 400)

	result := classifyContentSafetyViolation(c, err, "cyber_policy")
	require.Equal(t, "credential_theft_phishing", result.FineCategory)
	require.Equal(t, "local_rule", result.ReasonSource)
	require.Equal(t, "medium", result.ReasonConfidence)
	require.Contains(t, result.ReasonSummary, "本地推断")
	require.Contains(t, result.ReasonSummary, "未保存原始请求正文")
	stored := strings.Join([]string{result.OfficialMessage, result.ReasonSummary}, " ")
	require.NotContains(t, stored, "victim@example.com")
	require.NotContains(t, stored, "sk-testfixture123456")
	require.NotContains(t, stored, "token=secret")
	require.NotContains(t, stored, "fake login page")
}

func TestClassifyContentSafetyViolationUsesTruthfulFallback(t *testing.T) {
	c := contentSafetyClassifierContext(t, `{"input":"ambiguous text without a deterministic subtype"}`)
	err := types.WithOpenAIError(types.OpenAIError{Message: "request rejected", Type: "invalid_request", Code: "cyber_policy"}, 400)
	result := classifyContentSafetyViolation(c, err, "cyber_policy")
	require.Equal(t, "cyber_policy_other", result.FineCategory)
	require.Equal(t, "low", result.ReasonConfidence)
	require.Contains(t, result.ReasonSummary, "其他网络安全高风险")
	require.Contains(t, result.ReasonSummary, "本地推断")
}

func TestSanitizeContentSafetyAuditTextRedactsCommonSecrets(t *testing.T) {
	raw := "email=user@example.com phone=+1 555 123 4567 password=hunter2 url=https://example.com/a?q=secret"
	result := sanitizeContentSafetyAuditText(raw, 512)
	require.NotContains(t, result, "user@example.com")
	require.NotContains(t, result, "555 123 4567")
	require.NotContains(t, result, "hunter2")
	require.NotContains(t, result, "q=secret")
}
