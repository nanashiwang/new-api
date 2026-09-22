package types

import (
	"context"
	"errors"
	"testing"
)

func TestClassifyFailure(t *testing.T) {
	cases := []struct {
		name string
		err  *NewAPIError
		want FailureClass
	}{
		{"quota", NewErrorWithStatusCode(errors.New("quota"), ErrorCodeInsufficientUserQuota, 403), FailureUserRequest},
		{"bad body", NewErrorWithStatusCode(errors.New("bad json"), ErrorCodeBadRequestBody, 400), FailureUserRequest},
		{"adapter bad body", NewErrorWithStatusCode(errors.New("websocket write failed"), ErrorCodeBadRequestBody, 500), FailureService},
		{"internal validation", NewErrorWithStatusCode(errors.New("invalid request type"), ErrorCodeInvalidRequest, 400), FailureUnknown},
		{"model not found", WithOpenAIError(OpenAIError{Code: "model_not_found"}, 404), FailureUnknown},
		{"context too long", WithOpenAIError(OpenAIError{Code: "context_length_exceeded"}, 400), FailureUserRequest},
		{"rate limit", WithOpenAIError(OpenAIError{Code: "rate_limit_exceeded"}, 429), FailureService},
		{"cancel", NewErrorWithStatusCode(context.Canceled, ErrorCodeDoRequestFailed, 500), FailureClientCanceled},
		{"stream cancel", NewErrorWithStatusCode(errors.New("client canceled while receiving responses stream"), ErrorCodeDoRequestFailed, 500), FailureClientCanceled},
		{"timeout", NewErrorWithStatusCode(context.DeadlineExceeded, ErrorCodeDoRequestFailed, 504), FailureService},
		{"no route", NewErrorWithStatusCode(errors.New("no route"), ErrorCodeGetChannelFailed, 503), FailureService},
		{"upstream quota code", WithOpenAIError(OpenAIError{Code: "insufficient_user_quota"}, 403), FailureUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ClassifyFailure(tc.err); got != tc.want {
				t.Fatalf("got %s want %s", got, tc.want)
			}
		})
	}
}
