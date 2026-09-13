package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/stretchr/testify/require"
)

func TestVolcEngineUpstreamModelsEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v3/models", r.URL.Path)
		require.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{"data":[{"id":"doubao-test"}]}`))
	}))
	defer server.Close()
	service.InitHttpClient()
	channel := &model.Channel{Type: constant.ChannelTypeVolcEngine, Key: "test-key", BaseURL: common.GetPointer(server.URL + "/")}
	models, err := fetchChannelUpstreamModelIDs(channel)
	require.NoError(t, err)
	require.Equal(t, []string{"doubao-test"}, models)
}

func TestChannelTestUsesModelCapabilities(t *testing.T) {
	for _, name := range []string{"gpt-5.2", "gpt-6-astra", "gpt-6-astra-2026-09-03", "o3"} {
		t.Run(name, func(t *testing.T) {
			req := buildTestRequest(name, "", &model.Channel{Type: constant.ChannelTypeOpenAI}, false).(*dto.GeneralOpenAIRequest)
			require.Equal(t, uint(16), req.MaxCompletionTokens)
			require.Zero(t, req.MaxTokens)
		})
	}
}
