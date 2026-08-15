package service

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestIsContentSafetyPolicyErrorUsesExactAllowlist(t *testing.T) {
	tests := []struct {
		name string
		code types.ErrorCode
		typ  string
		want bool
	}{
		{name: "cyber policy", code: "cyber_policy", typ: "invalid_request", want: true},
		{name: "content filter in type", code: "unknown_error", typ: "content_filter", want: true},
		{name: "policy violation", code: "policy_violation", typ: "invalid_request", want: true},
		{name: "context length", code: "context_length_exceeded", typ: "invalid_request_error", want: false},
		{name: "rate limit", code: "rate_limit_exceeded", typ: "rate_limit_error", want: false},
		{name: "server error", code: "server_error", typ: "server_error", want: false},
		{name: "substring is not enough", code: "not_cyber_policy_related", typ: "invalid_request", want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := types.WithOpenAIError(types.OpenAIError{
				Message: "upstream rejected request", Type: test.typ, Code: string(test.code),
			}, http.StatusBadRequest)
			require.Equal(t, test.want, IsContentSafetyPolicyError(err))
		})
	}
}

func TestNormalizeContentSafetyPolicyErrorPreservesErrorAndSkipsRetry(t *testing.T) {
	original := types.WithOpenAIError(types.OpenAIError{
		Message: "request rejected by policy", Type: "invalid_request", Code: "cyber_policy",
	}, http.StatusBadRequest)
	original.Upstream = &types.UpstreamDiagnostics{RequestID: "upstream-request"}

	normalized := NormalizeContentSafetyPolicyError(original)
	require.True(t, types.IsSkipRetryError(normalized))
	require.Equal(t, types.ErrorCode("cyber_policy"), normalized.GetErrorCode())
	require.Equal(t, "request rejected by policy", normalized.ToOpenAIError().Message)
	require.Equal(t, "upstream-request", normalized.Upstream.RequestID)

	contextLimit := types.WithOpenAIError(types.OpenAIError{
		Message: "too long", Type: "invalid_request_error", Code: "context_length_exceeded",
	}, http.StatusBadRequest)
	require.Same(t, contextLimit, NormalizeContentSafetyPolicyError(contextLimit))
}

func TestHashContentSafetyRequestDoesNotChangeStoragePosition(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	storage, err := common.CreateBodyStorage([]byte(`{"model":"gpt-5.6-sol","input":"sensitive user content"}`))
	require.NoError(t, err)
	t.Cleanup(func() { _ = storage.Close() })
	c.Set(common.KeyBodyStorage, storage)
	_, err = storage.Seek(7, io.SeekStart)
	require.NoError(t, err)

	hash, err := hashContentSafetyRequest(c)
	require.NoError(t, err)
	require.Len(t, hash, 64)
	position, err := storage.Seek(0, io.SeekCurrent)
	require.NoError(t, err)
	require.EqualValues(t, 7, position)
	require.NotContains(t, hash, "sensitive")
}

func TestNormalizeContentSafetyPolicyErrorHandlesNil(t *testing.T) {
	require.Nil(t, NormalizeContentSafetyPolicyError(nil))
	require.False(t, IsContentSafetyPolicyError(types.NewError(errors.New("network reset"), types.ErrorCodeDoRequestFailed)))
}

func TestContentSafetyRecordOnlyGroupUsesExactMatch(t *testing.T) {
	require.True(t, IsContentSafetyRecordOnlyGroup("破甲"))
	require.False(t, IsContentSafetyRecordOnlyGroup("破甲测试"))
	require.False(t, IsContentSafetyRecordOnlyGroup(" 破甲"))
	require.False(t, IsContentSafetyRecordOnlyGroup("破甲 "))
}

func TestRecordContentSafetyPolicyViolationRecordOnlyKeepsAuditWithoutUserAction(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:content_safety_record_only?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.User{}, &model.Log{}, &model.UserSubscription{},
		&model.ContentSafetyViolation{}, &model.ContentSafetyReviewCase{},
		&model.ContentSafetyEvidence{}, &model.ContentSafetyNotification{},
	))
	originalDB, originalSecret := model.DB, common.CryptoSecret
	model.DB, common.CryptoSecret = db, "content-safety-record-only-secret"
	t.Setenv("CRYPTO_SECRET", "content-safety-record-only-secret")
	t.Cleanup(func() {
		model.DB, common.CryptoSecret = originalDB, originalSecret
		sqlDB, sqlErr := db.DB()
		if sqlErr == nil {
			_ = sqlDB.Close()
		}
	})

	user := &model.User{Username: "record-only-policy-user", Role: common.RoleCommonUser, Status: common.UserStatusEnabled}
	require.NoError(t, db.Create(user).Error)

	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	storage, err := common.CreateBodyStorage([]byte(`{"model":"gpt-5.6-sol","input":"audit-only test"}`))
	require.NoError(t, err)
	t.Cleanup(func() { _ = storage.Close() })
	c.Set(common.KeyBodyStorage, storage)

	policyErr := types.WithOpenAIError(types.OpenAIError{
		Message: "request rejected by policy", Type: "invalid_request", Code: "cyber_policy",
	}, http.StatusBadRequest)
	var latest *model.ContentSafetyEnforcementResult
	for sequence := 1; sequence <= 4; sequence++ {
		info := &relaycommon.RelayInfo{
			UserId: user.Id, TokenId: 10, ChannelMeta: &relaycommon.ChannelMeta{ChannelId: 20},
			RequestId:       fmt.Sprintf("record-only-request-%d", sequence),
			OriginModelName: "gpt-5.6-sol", UsingGroup: ContentSafetyRecordOnlyGroup,
			StartTime: time.Now(), IsStream: true,
		}
		latest, err = RecordContentSafetyPolicyViolation(c, info, policyErr)
		require.NoError(t, err)
		require.NotNil(t, latest)
		require.False(t, latest.Duplicate)
		require.Equal(t, model.ContentSafetyActionRecorded, latest.Violation.Action)
		require.Zero(t, latest.Violation.CooldownUntil)
		require.Nil(t, latest.ReviewCase)
	}

	var violations, evidences, notifications, reviews, userLogs int64
	require.NoError(t, db.Model(&model.ContentSafetyViolation{}).Where("user_id = ?", user.Id).Count(&violations).Error)
	require.NoError(t, db.Model(&model.ContentSafetyEvidence{}).Count(&evidences).Error)
	require.NoError(t, db.Model(&model.ContentSafetyNotification{}).Where("user_id = ?", user.Id).Count(&notifications).Error)
	require.NoError(t, db.Model(&model.ContentSafetyReviewCase{}).Where("user_id = ?", user.Id).Count(&reviews).Error)
	require.NoError(t, db.Model(&model.Log{}).Where("user_id = ? AND type = ?", user.Id, model.LogTypeSystem).Count(&userLogs).Error)
	require.EqualValues(t, 4, violations)
	require.EqualValues(t, 4, evidences)
	require.Zero(t, notifications)
	require.Zero(t, reviews)
	require.Zero(t, userLogs)

	state, err := model.GetUserContentSafetyState(user.Id)
	require.NoError(t, err)
	require.Equal(t, model.ContentSafetyLevelNormal, state.Level)
	require.False(t, state.HasUnreadWarning)
	require.Zero(t, state.WindowCount)
	require.Zero(t, state.CooldownUntil)
	require.Same(t, policyErr, EnrichContentSafetyClientError(policyErr, latest))
}
