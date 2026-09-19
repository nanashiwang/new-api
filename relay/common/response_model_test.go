package common

import (
	"net/http/httptest"
	"os"
	"testing"

	projectcommon "github.com/QuantumNous/new-api/common"
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
	require.False(t, info.ResponseModel.Mismatch())
	require.Equal(t, "gpt-4o", info.ResponseModel.UpstreamModel)
	info.ObserveResponseModel("gpt-4o")
	require.Equal(t, "gpt-4o-2024-08-06", info.ResponseModel.ReturnedModel)
	info.ObserveResponseModel("different-model")
	require.True(t, info.ResponseModel.Mismatch())
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
	require.False(t, info.ResponseModel.Mismatch())
}

func TestResponseModelSharedComparisonCases(t *testing.T) {
	raw, err := os.ReadFile("testdata/response_model_cases.json")
	require.NoError(t, err)
	var cases []struct {
		Name        string        `json:"name"`
		Observation ResponseModel `json:"observation"`
		Mismatch    bool          `json:"mismatch"`
	}
	require.NoError(t, projectcommon.Unmarshal(raw, &cases))
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			require.Equal(t, tc.Mismatch, tc.Observation.Mismatch())
			serialized, err := projectcommon.Marshal(tc.Observation)
			require.NoError(t, err)
			require.NotContains(t, string(serialized), "mismatch", "persist names only, including after loading old rows")
		})
	}
	var absent *ResponseModel
	require.False(t, absent.Mismatch())
}

func TestObserveResponseModelPreservesFirstMismatch(t *testing.T) {
	info := &RelayInfo{OriginModelName: "alias", ChannelMeta: &ChannelMeta{UpstreamModelName: "gpt-4o"}}
	for _, name := range []string{"gpt-4o", "vendor/GPT-4O-2024-08-06", "gpt-4o"} {
		info.ObserveResponseModel(name)
		require.False(t, info.ResponseModel.Mismatch())
	}
	require.Equal(t, "vendor/GPT-4O-2024-08-06", info.ResponseModel.ReturnedModel)
	for _, name := range []string{"gpt-4o-mini", "not-gpt-4o", "gpt-4o", " ", "alias"} {
		info.ObserveResponseModel(name)
		require.True(t, info.ResponseModel.Mismatch())
		require.Equal(t, "gpt-4o-mini", info.ResponseModel.ReturnedModel)
	}
}
