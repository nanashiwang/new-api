package service

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/types"
	"github.com/go-redis/redis/v8"
)

func TestChannelModelCircuitScopesByChannelAndMappedModel(t *testing.T) {
	originalRedisEnabled := common.RedisEnabled
	common.RedisEnabled = false
	resetChannelModelCircuitForTest()
	t.Cleanup(func() {
		common.RedisEnabled = originalRedisEnabled
		resetChannelModelCircuitForTest()
	})

	mapping := `{"gpt-5.6-sol":"gpt-5.6"}`
	channelA := &model.Channel{Id: 1, ModelMapping: &mapping}
	channelB := &model.Channel{Id: 2, ModelMapping: &mapping}
	err := types.NewOpenAIError(errors.New("No available OpenAI accounts support the requested model: gpt-5.6-sol"), types.ErrorCodeBadResponseStatusCode, http.StatusBadRequest)
	if !RecordChannelModelCircuitFailure(channelA, "gpt-5.6-sol", err) {
		t.Fatal("expected circuit to open")
	}
	if !IsChannelModelCircuitOpen(channelA, "gpt-5.6-sol") {
		t.Fatal("expected mapped model circuit to be open")
	}
	if IsChannelModelCircuitOpen(channelA, "gpt-5.5") {
		t.Fatal("different model must not be affected")
	}
	if IsChannelModelCircuitOpen(channelB, "gpt-5.6-sol") {
		t.Fatal("different channel must not be affected")
	}
}

func TestChannelModelCircuitIgnoresOtherBadRequestsAndSkipRetry(t *testing.T) {
	originalRedisEnabled := common.RedisEnabled
	common.RedisEnabled = false
	resetChannelModelCircuitForTest()
	t.Cleanup(func() {
		common.RedisEnabled = originalRedisEnabled
		resetChannelModelCircuitForTest()
	})
	channel := &model.Channel{Id: 1}
	other := types.NewOpenAIError(errors.New("invalid request"), types.ErrorCodeBadResponseStatusCode, http.StatusBadRequest)
	if RecordChannelModelCircuitFailure(channel, "gpt-5.6-sol", other) {
		t.Fatal("unrelated bad request must not open circuit")
	}
	skip := types.NewError(errors.New("No available OpenAI accounts support the requested model"), types.ErrorCodeBadResponseStatusCode, types.ErrOptionWithSkipRetry())
	if RecordChannelModelCircuitFailure(channel, "gpt-5.6-sol", skip) {
		t.Fatal("local skip-retry error must not open circuit")
	}
}

func TestChannelModelCircuitKeepsLocalShadowWhenRedisFails(t *testing.T) {
	originalRedisEnabled, originalRDB := common.RedisEnabled, common.RDB
	common.RedisEnabled = true
	common.RDB = redis.NewClient(&redis.Options{
		Addr:         "127.0.0.1:1",
		DialTimeout:  10 * time.Millisecond,
		ReadTimeout:  10 * time.Millisecond,
		WriteTimeout: 10 * time.Millisecond,
	})
	resetChannelModelCircuitForTest()
	t.Cleanup(func() {
		_ = common.RDB.Close()
		common.RedisEnabled, common.RDB = originalRedisEnabled, originalRDB
		resetChannelModelCircuitForTest()
	})
	channel := &model.Channel{Id: 1}
	err := types.NewOpenAIError(errors.New("No available OpenAI accounts support the requested model"), types.ErrorCodeBadResponseStatusCode, http.StatusBadRequest)
	if !RecordChannelModelCircuitFailure(channel, "gpt-5.6-sol", err) {
		t.Fatal("expected local shadow to open")
	}
	if !IsChannelModelCircuitOpen(channel, "gpt-5.6-sol") {
		t.Fatal("Redis failure must not make the circuit fail open")
	}
}

func TestResolveChannelUpstreamModelHandlesCompactAndChains(t *testing.T) {
	mapping := `{"alias-a":"alias-b","alias-b":"gpt-5.6","other":"gpt-5.6"}`
	channel := &model.Channel{Id: 1, ModelMapping: &mapping}
	if got := ResolveChannelUpstreamModel(channel, "alias-a-openai-compact"); got != "gpt-5.6" {
		t.Fatalf("compact chained mapping: got %q", got)
	}
	if channelModelCircuitKey(channel.Id, ResolveChannelUpstreamModel(channel, "alias-a")) !=
		channelModelCircuitKey(channel.Id, ResolveChannelUpstreamModel(channel, "other")) {
		t.Fatal("aliases mapped to the same upstream model must share a circuit")
	}
}
