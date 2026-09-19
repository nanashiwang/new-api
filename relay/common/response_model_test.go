package common

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestObserveResponseModel(t *testing.T) {
	var absent *RelayInfo
	absent.ObserveResponseModel("model")
	info := &RelayInfo{OriginModelName: "alias", ChannelMeta: &ChannelMeta{UpstreamModelName: "gpt-4o"}}
	info.ObserveResponseModel("")
	require.Nil(t, info.ResponseModel)
	info.ObserveResponseModel("gpt-4o-2024-08-06")
	require.False(t, info.ResponseModel.Mismatch)
	require.Equal(t, "gpt-4o", info.ResponseModel.UpstreamModel)
	info.ObserveResponseModel("gpt-4o")
	require.Equal(t, "gpt-4o-2024-08-06", info.ResponseModel.ReturnedModel)
	info.ObserveResponseModel("different-model")
	require.True(t, info.ResponseModel.Mismatch)
	info.ObserveResponseModel("alias")
	info.ObserveResponseModel("")
	require.Equal(t, "different-model", info.ResponseModel.ReturnedModel)
	require.Equal(t, "gpt-4o", info.UpstreamModelName, "diagnostics must not change routing/pricing")
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	info.InitChannelMeta(c)
	require.Nil(t, info.ResponseModel, "retry must discard the previous channel observation")
}

func TestObserveResponseModelWithoutChannelMeta(t *testing.T) {
	info := &RelayInfo{OriginModelName: "GPT-4O"}
	info.ObserveResponseModel("gpt-4o")
	require.False(t, info.ResponseModel.Mismatch)
}
