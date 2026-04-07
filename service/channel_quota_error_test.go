package service

import (
	"errors"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/types"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestIsQuotaRelatedErrorByCode(t *testing.T) {
	err := types.NewError(errors.New("insufficient"), types.ErrorCodeInsufficientUserQuota)
	if !IsQuotaRelatedError(err) {
		t.Fatalf("expected quota related error by error code")
	}
}

func TestIsQuotaRelatedErrorByOpenAIType(t *testing.T) {
	err := types.WithOpenAIError(types.OpenAIError{
		Message: "quota exceeded",
		Type:    "insufficient_quota",
		Code:    "insufficient_quota",
	}, 403)
	if !IsQuotaRelatedError(err) {
		t.Fatalf("expected quota related error by openai type/code")
	}
}

func TestIsQuotaRelatedErrorNegative(t *testing.T) {
	err := types.NewError(errors.New("upstream timeout"), types.ErrorCodeDoRequestFailed)
	if IsQuotaRelatedError(err) {
		t.Fatalf("did not expect timeout to be quota related")
	}
}

func TestIsChannelModelMismatchError_CodexUnsupportedModel(t *testing.T) {
	err := types.WithOpenAIError(types.OpenAIError{
		Message: "The 'gpt-5.4' model is not supported when using Codex with a ChatGPT account.",
		Type:    "bad_response_status_code",
		Code:    "bad_response_status_code",
	}, 400)
	if !IsChannelModelMismatchError(err) {
		t.Fatalf("expected codex unsupported model to be treated as channel mismatch")
	}
}

func TestIsChannelModelMismatchError_StreamRequired(t *testing.T) {
	err := types.NewOpenAIError(errors.New("bad response status code 400, message: Stream must be set to true"), types.ErrorCodeBadResponseStatusCode, 400)
	if !IsChannelModelMismatchError(err) {
		t.Fatalf("expected stream-required error to be treated as channel mismatch")
	}
}

func TestIsChannelModelMismatchError_RequestedModelUnavailable(t *testing.T) {
	err := types.WithOpenAIError(types.OpenAIError{
		Message: "No available OpenAI accounts support the requested model: gpt-5.4",
		Type:    "upstream_error",
		Code:    nil,
	}, 503)
	if !IsChannelModelMismatchError(err) {
		t.Fatalf("expected requested-model-unavailable error to be treated as channel mismatch")
	}
}

func TestIsChannelModelMismatchError_InvalidRequestNegative(t *testing.T) {
	err := types.WithOpenAIError(types.OpenAIError{
		Message: "Invalid request",
		Type:    "<nil>",
		Code:    nil,
	}, 400)
	if IsChannelModelMismatchError(err) {
		t.Fatalf("did not expect generic invalid request to be treated as channel mismatch")
	}
}

func TestIsRetryableUpstreamQuotaError_LocalQuotaNegative(t *testing.T) {
	err := types.NewError(errors.New("用户额度不足"), types.ErrorCodeInsufficientUserQuota, types.ErrOptionWithSkipRetry())
	if IsRetryableUpstreamQuotaError(err) {
		t.Fatalf("did not expect local skip-retry quota error to be treated as retryable upstream quota")
	}
}

func TestIsRetryableUpstreamQuotaError_UpstreamQuotaPositive(t *testing.T) {
	err := types.WithOpenAIError(types.OpenAIError{
		Message: "用户额度不足",
		Type:    "insufficient_user_quota",
		Code:    "insufficient_user_quota",
	}, 403)
	if !IsRetryableUpstreamQuotaError(err) {
		t.Fatalf("expected upstream quota error to be retryable")
	}
}

func TestApplyChannelFailureRetryExclusion_UsesTagGroup(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	originDB := model.DB
	originLogDB := model.LOG_DB
	originMemoryCacheEnabled := common.MemoryCacheEnabled
	model.DB = db
	model.LOG_DB = db
	common.MemoryCacheEnabled = false
	t.Cleanup(func() {
		model.DB = originDB
		model.LOG_DB = originLogDB
		common.MemoryCacheEnabled = originMemoryCacheEnabled
	})

	if err := db.AutoMigrate(&model.Channel{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	tag := "shared-upstream"
	channels := []model.Channel{
		{Id: 1, Name: "primary", Status: common.ChannelStatusEnabled, Tag: &tag},
		{Id: 2, Name: "sibling", Status: common.ChannelStatusEnabled, Tag: &tag},
		{Id: 3, Name: "other", Status: common.ChannelStatusEnabled},
	}
	if err := db.Create(&channels).Error; err != nil {
		t.Fatalf("seed channels: %v", err)
	}

	param := &RetryParam{}
	retryErr := types.WithOpenAIError(types.OpenAIError{
		Message: "No available OpenAI accounts support the requested model: gpt-5.4",
		Type:    "upstream_error",
		Code:    nil,
	}, 503)

	ApplyChannelFailureRetryExclusion(param, &channels[0], retryErr)

	if len(param.ExcludeChannels) != 2 {
		t.Fatalf("expected two excluded channels, got %v", param.ExcludeChannels)
	}
	if param.ExcludeChannels[0] != 1 || param.ExcludeChannels[1] != 2 {
		t.Fatalf("unexpected excluded channels: %v", param.ExcludeChannels)
	}
}

func TestApplyChannelFailureRetryExclusion_FallsBackToChannelID(t *testing.T) {
	param := &RetryParam{}
	channel := &model.Channel{Id: 7, Name: "single"}
	retryErr := types.WithOpenAIError(types.OpenAIError{
		Message: "No available OpenAI accounts support the requested model: gpt-5.4",
		Type:    "upstream_error",
		Code:    nil,
	}, 503)

	ApplyChannelFailureRetryExclusion(param, channel, retryErr)

	if len(param.ExcludeChannels) != 1 || param.ExcludeChannels[0] != 7 {
		t.Fatalf("unexpected excluded channels: %v", param.ExcludeChannels)
	}
}
