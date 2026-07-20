package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/types"
)

const (
	crsMemoryPressureCircuitPrefix = "crs:memory_pressure:circuit"
	crsMemoryPressureDefaultTTL    = time.Minute
)

type CRSShortCircuitTrip struct {
	Scope      string
	Key        string
	TTLSeconds int
	Opened     bool
}

type crsCircuitScope struct {
	scope string
	key   string
}

var (
	crsLocalCircuitMu sync.Mutex
	crsLocalCircuit   = map[string]time.Time{}
)

func IsCRSMemoryPressureError(err *types.NewAPIError) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error() + " " + err.ToOpenAIError().Message)
	compact := strings.Join(strings.Fields(message), "")
	return strings.Contains(message, "crs is under memory pressure") ||
		strings.Contains(compact, "crsisundermemorypressure")
}

func ApplyCRSMemoryPressureShortCircuit(channel *model.Channel, err *types.NewAPIError) []CRSShortCircuitTrip {
	if channel == nil || !IsCRSMemoryPressureError(err) {
		return nil
	}
	return ApplyChannelScopedShortCircuit(channel, "CRS memory pressure")
}

func ApplyChannelScopedShortCircuit(channel *model.Channel, reason string) []CRSShortCircuitTrip {
	if channel == nil {
		return nil
	}
	ttl := crsShortCircuitTTL()
	if ttl <= 0 {
		return nil
	}
	if reason == "" {
		reason = "temporary upstream instability"
	}
	scopes := crsCircuitScopes(channel)
	trips := make([]CRSShortCircuitTrip, 0, len(scopes))
	for _, scope := range scopes {
		opened := setCRSCircuit(scope, ttl, fmt.Sprintf("channel=%d reason=%s", channel.Id, reason))
		trips = append(trips, CRSShortCircuitTrip{
			Scope:      scope.scope,
			Key:        scope.key,
			TTLSeconds: int(ttl.Seconds()),
			Opened:     opened,
		})
	}
	return trips
}

func IsCRSShortCircuitOpenForChannel(channel *model.Channel) bool {
	for _, scope := range crsCircuitScopes(channel) {
		if isCRSCircuitOpen(scope) {
			return true
		}
	}
	return false
}

func crsShortCircuitTTL() time.Duration {
	setting := operation_setting.GetMonitorSetting()
	if setting != nil && setting.PreDisableWaitEnabled && setting.PreDisableWaitMinutes > 0 {
		return time.Duration(setting.PreDisableWaitMinutes * float64(time.Minute))
	}
	return crsMemoryPressureDefaultTTL
}

func crsCircuitScopes(channel *model.Channel) []crsCircuitScope {
	if channel == nil {
		return nil
	}
	scopes := make([]crsCircuitScope, 0, 1)
	if channel.BaseURL != nil {
		if baseURL := normalizeCRSCircuitBaseURL(*channel.BaseURL); baseURL != "" {
			scopes = append(scopes, crsCircuitScope{scope: "base_url", key: baseURL})
		}
	}
	return scopes
}

func normalizeCRSCircuitBaseURL(raw string) string {
	return model.NormalizeChannelBaseURL(raw)
}

func crsCircuitRedisEnabled() bool {
	return common.RedisEnabled && common.RDB != nil
}

func crsCircuitKey(scope crsCircuitScope) string {
	return fmt.Sprintf("%s:%s:%s", crsMemoryPressureCircuitPrefix, scope.scope, common.GenerateHMAC(scope.key))
}

func setCRSCircuit(scope crsCircuitScope, ttl time.Duration, reason string) bool {
	if scope.scope == "" || scope.key == "" || ttl <= 0 {
		return false
	}
	key := crsCircuitKey(scope)
	if crsCircuitRedisEnabled() {
		ok, err := common.RDB.SetNX(context.Background(), key, reason, ttl).Result()
		if err == nil {
			return ok
		}
	}

	crsLocalCircuitMu.Lock()
	defer crsLocalCircuitMu.Unlock()
	if until, ok := crsLocalCircuit[key]; ok && time.Now().Before(until) {
		return false
	}
	crsLocalCircuit[key] = time.Now().Add(ttl)
	return true
}

func isCRSCircuitOpen(scope crsCircuitScope) bool {
	if scope.scope == "" || scope.key == "" {
		return false
	}
	key := crsCircuitKey(scope)
	if crsCircuitRedisEnabled() {
		_, err := common.RedisGet(key)
		if err == nil {
			return true
		}
	}

	return isLocalCRSCircuitOpen(key)
}

func isLocalCRSCircuitOpen(key string) bool {
	crsLocalCircuitMu.Lock()
	defer crsLocalCircuitMu.Unlock()
	until, ok := crsLocalCircuit[key]
	if !ok {
		return false
	}
	if time.Now().Before(until) {
		return true
	}
	delete(crsLocalCircuit, key)
	return false
}

func resetCRSShortCircuitForTest() {
	crsLocalCircuitMu.Lock()
	defer crsLocalCircuitMu.Unlock()
	crsLocalCircuit = map[string]time.Time{}
}
