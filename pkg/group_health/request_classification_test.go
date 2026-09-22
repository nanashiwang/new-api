package grouphealth

import (
	"context"
	"errors"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"testing"
	"time"
)

func TestRequestFailureClassification(t *testing.T) {
	now := time.Now()
	info := &relaycommon.RelayInfo{UsingGroup: "group-a"}
	userErr := types.NewErrorWithStatusCode(errors.New("bad body"), types.ErrorCodeBadRequestBody, 400)
	r := NewRequest("model", true)
	r.ObserveResult(info, userErr, true, now)
	if len(r.Samples(now)) != 0 {
		t.Fatal("user error entered denominator")
	}
	r.Failure("group-a")
	r.ObserveResult(info, userErr, true, now)
	if samples := r.Samples(now); len(samples) != 1 || samples[0].Success {
		t.Fatal("user error erased service failure")
	}
	r.ObserveResult(info, nil, true, now)
	if samples := r.Samples(now); len(samples) != 1 || !samples[0].Success {
		t.Fatal("successful same-group retry should fold")
	}
	info.UsingGroup = "group-b"
	r.ObserveResult(info, types.NewErrorWithStatusCode(context.Canceled, types.ErrorCodeDoRequestFailed, 500), false, now)
	if len(r.Samples(now)) != 2 {
		t.Fatal("cancellation must not silently inflate availability")
	}
	r.ObserveResult(info, types.WithOpenAIError(types.OpenAIError{Code: "model_not_found"}, 404), true, now)
	for _, s := range r.Samples(now) {
		if s.Group == "group-b" && s.Success {
			t.Fatal("unsupported upstream model is not success")
		}
	}
}
