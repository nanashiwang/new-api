package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"
)

const (
	channelModelCircuitPrefix = "channel:model:circuit"
	channelModelCircuitTTL    = time.Minute
)

var (
	channelModelCircuitMu    sync.Mutex
	channelModelLocalCircuit = map[string]time.Time{}
)

// ResolveChannelUpstreamModel mirrors the channel model-mapping rules used by the relay.
func ResolveChannelUpstreamModel(channel *model.Channel, requestedModel string) string {
	current := strings.TrimSpace(requestedModel)
	if strings.HasSuffix(current, ratio_setting.CompactModelSuffix) {
		current = strings.TrimSuffix(current, ratio_setting.CompactModelSuffix)
	}
	if channel == nil || current == "" {
		return current
	}
	raw := strings.TrimSpace(channel.GetModelMapping())
	if raw == "" || raw == "{}" {
		return current
	}
	mapping := make(map[string]string)
	if err := common.Unmarshal([]byte(raw), &mapping); err != nil {
		return current
	}
	visited := map[string]struct{}{current: {}}
	for {
		next := strings.TrimSpace(mapping[current])
		if next == "" || next == current {
			return current
		}
		if _, exists := visited[next]; exists {
			return current
		}
		visited[next] = struct{}{}
		current = next
	}
}

func RecordChannelModelCircuitFailure(channel *model.Channel, requestedModel string, err *types.NewAPIError) bool {
	if channel == nil || !IsUpstreamModelTemporaryUnavailableError(err) || types.IsSkipRetryError(err) {
		return false
	}
	key := channelModelCircuitKey(channel.Id, ResolveChannelUpstreamModel(channel, requestedModel))
	return openChannelModelCircuitKey(key)
}

func OpenChannelShortCircuit(channel *model.Channel) bool {
	if channel == nil {
		return false
	}
	return openChannelModelCircuitKey(channelModelCircuitKey(channel.Id, "*"))
}

func openChannelModelCircuitKey(key string) bool {
	if key == "" {
		return false
	}
	now := time.Now()
	channelModelCircuitMu.Lock()
	until, alreadyOpen := channelModelLocalCircuit[key]
	alreadyOpen = alreadyOpen && now.Before(until)
	if !alreadyOpen {
		channelModelLocalCircuit[key] = now.Add(channelModelCircuitTTL)
	}
	channelModelCircuitMu.Unlock()

	if common.RedisEnabled && common.RDB != nil {
		_, _ = common.RDB.SetNX(context.Background(), key, "1", channelModelCircuitTTL).Result()
	}
	return !alreadyOpen
}

func IsChannelModelCircuitOpen(channel *model.Channel, requestedModel string) bool {
	if channel == nil {
		return false
	}
	keys := []string{
		channelModelCircuitKey(channel.Id, "*"),
		channelModelCircuitKey(channel.Id, ResolveChannelUpstreamModel(channel, requestedModel)),
	}
	for _, key := range keys {
		if isChannelModelCircuitKeyOpen(key) {
			return true
		}
	}
	return false
}

func isChannelModelCircuitKeyOpen(key string) bool {
	if key == "" {
		return false
	}
	if common.RedisEnabled && common.RDB != nil {
		if _, err := common.RedisGet(key); err == nil {
			return true
		}
	}
	now := time.Now()
	channelModelCircuitMu.Lock()
	defer channelModelCircuitMu.Unlock()
	until, exists := channelModelLocalCircuit[key]
	if !exists {
		return false
	}
	if now.Before(until) {
		return true
	}
	delete(channelModelLocalCircuit, key)
	return false
}

func channelModelCircuitKey(channelID int, upstreamModel string) string {
	upstreamModel = strings.TrimSpace(upstreamModel)
	if channelID <= 0 || upstreamModel == "" {
		return ""
	}
	scope := fmt.Sprintf("channel:%d|model:%s", channelID, upstreamModel)
	return fmt.Sprintf("%s:%s", channelModelCircuitPrefix, common.GenerateHMAC(scope))
}

func resetChannelModelCircuitForTest() {
	channelModelCircuitMu.Lock()
	channelModelLocalCircuit = map[string]time.Time{}
	channelModelCircuitMu.Unlock()
}
