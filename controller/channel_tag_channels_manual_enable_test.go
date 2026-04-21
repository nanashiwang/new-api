package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func seedChannelForManualEnableEndpointTest(t *testing.T, id int, name string, status int, tag string) *model.Channel {
	t.Helper()

	priority := int64(0)
	weight := uint(10)
	channel := &model.Channel{
		Id:       id,
		Name:     name,
		Key:      "sk-test-key",
		Type:     constant.ChannelTypeOpenAI,
		Status:   status,
		Group:    "default",
		Models:   "gpt-4o",
		Tag:      common.GetPointer(tag),
		Priority: &priority,
		Weight:   &weight,
	}
	require.NoError(t, channel.Insert())
	return channel
}

func TestGetTagChannels_ReturnsAllChannelsWithRuntimeMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupChannelSearchControllerTestDB(t)

	tag := "team-a"
	priority := int64(0)
	weight := uint(10)

	pendingChannel := &model.Channel{
		Id:       201,
		Name:     "pending-single",
		Key:      "pending-key",
		Type:     constant.ChannelTypeOpenAI,
		Status:   common.ChannelStatusEnabled,
		Group:    "default",
		Models:   "gpt-4o",
		Tag:      common.GetPointer(tag),
		Priority: &priority,
		Weight:   &weight,
	}
	pendingChannel.SetPendingDisable(1_900_000_123, "upstream unstable")
	require.NoError(t, pendingChannel.Insert())

	multiKeyChannel := &model.Channel{
		Id:       202,
		Name:     "multi-key",
		Key:      "key-1\nkey-2",
		Type:     constant.ChannelTypeOpenAI,
		Status:   common.ChannelStatusEnabled,
		Group:    "default",
		Models:   "gpt-4o",
		Tag:      common.GetPointer(tag),
		Priority: &priority,
		Weight:   &weight,
		ChannelInfo: model.ChannelInfo{
			IsMultiKey:                   true,
			MultiKeySize:                 2,
			MultiKeyStatusList:           map[int]int{},
			MultiKeyPendingDisableUntil:  map[int]int64{0: 1_900_000_999},
			MultiKeyPendingDisableReason: map[int]string{0: "retry later"},
			MultiKeyDisabledReason:       map[int]string{0: "should-be-hidden"},
			MultiKeyDisabledTime:         map[int]int64{0: 1_800_000_111},
		},
	}
	require.NoError(t, multiKeyChannel.Insert())

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/channel/tag/channels?tag=team-a", nil)

	GetTagChannels(ctx)
	require.Equal(t, http.StatusOK, recorder.Code)

	var resp struct {
		Success bool `json:"success"`
		Data    []struct {
			ID                          int   `json:"id"`
			EffectiveAvailable          bool  `json:"effective_available"`
			PendingDisableUntil         int64 `json:"pending_disable_until"`
			MultiKeyPendingDisableCount int   `json:"multi_key_pending_disable_count"`
			MultiKeyCooldownKeyCount    int   `json:"multi_key_cooldown_key_count"`
			ChannelInfo                 struct {
				MultiKeyDisabledReason map[string]string `json:"multi_key_disabled_reason"`
				MultiKeyDisabledTime   map[string]int64  `json:"multi_key_disabled_time"`
			} `json:"channel_info"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.True(t, resp.Success)
	require.Len(t, resp.Data, 2)

	itemsByID := make(map[int]struct {
		EffectiveAvailable          bool
		PendingDisableUntil         int64
		MultiKeyPendingDisableCount int
		MultiKeyDisabledReason      map[string]string
		MultiKeyDisabledTime        map[string]int64
	}, len(resp.Data))
	for _, item := range resp.Data {
		itemsByID[item.ID] = struct {
			EffectiveAvailable          bool
			PendingDisableUntil         int64
			MultiKeyPendingDisableCount int
			MultiKeyDisabledReason      map[string]string
			MultiKeyDisabledTime        map[string]int64
		}{
			EffectiveAvailable:          item.EffectiveAvailable,
			PendingDisableUntil:         item.PendingDisableUntil,
			MultiKeyPendingDisableCount: item.MultiKeyPendingDisableCount,
			MultiKeyDisabledReason:      item.ChannelInfo.MultiKeyDisabledReason,
			MultiKeyDisabledTime:        item.ChannelInfo.MultiKeyDisabledTime,
		}
	}

	require.False(t, itemsByID[201].EffectiveAvailable)
	require.EqualValues(t, 1_900_000_123, itemsByID[201].PendingDisableUntil)
	require.Equal(t, 0, itemsByID[201].MultiKeyPendingDisableCount)

	require.True(t, itemsByID[202].EffectiveAvailable)
	require.Equal(t, 1, itemsByID[202].MultiKeyPendingDisableCount)
	require.Nil(t, itemsByID[202].MultiKeyDisabledReason)
	require.Nil(t, itemsByID[202].MultiKeyDisabledTime)
}

func TestManualEnableChannel_RecoversPendingDisableEnabledChannel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupChannelSearchControllerTestDB(t)

	channel := seedChannelForManualEnableEndpointTest(t, 301, "pending-enable", common.ChannelStatusEnabled, "recover")
	channel.SetPendingDisable(1_900_000_123, "wait confirm")
	require.NoError(t, channel.SaveWithoutKey())

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "id", Value: "301"}}
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/channel/301/manual_enable", nil)

	ManualEnableChannel(ctx)
	require.Equal(t, http.StatusOK, recorder.Code)

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			ID                 int    `json:"id"`
			Key                string `json:"key"`
			Status             int    `json:"status"`
			EffectiveAvailable bool   `json:"effective_available"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.True(t, resp.Success)
	require.Equal(t, 301, resp.Data.ID)
	require.Equal(t, "", resp.Data.Key)
	require.Equal(t, common.ChannelStatusEnabled, resp.Data.Status)
	require.True(t, resp.Data.EffectiveAvailable)

	updated, err := model.GetChannelById(301, true)
	require.NoError(t, err)
	updated.PopulateRuntimeAvailability()
	require.Equal(t, common.ChannelStatusEnabled, updated.Status)
	require.Zero(t, updated.PendingDisableUntil)
	require.Empty(t, updated.PendingDisableReason)
	require.True(t, updated.EffectiveAvailable)
}

func TestManualEnableChannel_RecoversManuallyDisabledChannel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	setupChannelSearchControllerTestDB(t)

	seedChannelForManualEnableEndpointTest(t, 302, "manual-disabled", common.ChannelStatusManuallyDisabled, "recover")

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "id", Value: "302"}}
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/channel/302/manual_enable", nil)

	ManualEnableChannel(ctx)
	require.Equal(t, http.StatusOK, recorder.Code)

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			ID                 int  `json:"id"`
			Status             int  `json:"status"`
			EffectiveAvailable bool `json:"effective_available"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.True(t, resp.Success)
	require.Equal(t, 302, resp.Data.ID)
	require.Equal(t, common.ChannelStatusEnabled, resp.Data.Status)
	require.True(t, resp.Data.EffectiveAvailable)

	updated, err := model.GetChannelById(302, true)
	require.NoError(t, err)
	require.Equal(t, common.ChannelStatusEnabled, updated.Status)
}
