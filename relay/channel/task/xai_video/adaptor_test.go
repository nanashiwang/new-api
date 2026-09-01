package xai_video

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestXAIVideoURL(t *testing.T) {
	require.Equal(t, "https://api.x.ai/v1/videos/generations", xAIVideoURL("https://api.x.ai", "/videos/generations"))
	require.Equal(t, "https://proxy.example/x/v1/videos/abc", xAIVideoURL("https://proxy.example/x/v1/", "/videos/abc"))
}

func TestBuildRequestBodyNormalizesAliases(t *testing.T) {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", strings.NewReader(`{"model":"grok-imagine-video","prompt":"hi","seconds":"6","size":"720p","aspectRatio":"16:9"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "grok-imagine-video-1.5"}}
	body, err := (&TaskAdaptor{}).BuildRequestBody(ctx, info)
	require.NoError(t, err)
	data, err := io.ReadAll(body)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, common.Unmarshal(data, &payload))
	require.Equal(t, "grok-imagine-video-1.5", payload["model"])
	require.EqualValues(t, 6, payload["duration"])
	require.Equal(t, "720p", payload["resolution"])
	require.Equal(t, "16:9", payload["aspect_ratio"])
	_, hasSeconds := payload["seconds"]
	_, hasSize := payload["size"]
	_, hasCamelAspect := payload["aspectRatio"]
	require.False(t, hasSeconds)
	require.False(t, hasSize)
	require.False(t, hasCamelAspect)
	common.CleanupBodyStorage(ctx)
}

func TestParseTaskResult(t *testing.T) {
	adaptor := &TaskAdaptor{}
	queued, err := adaptor.ParseTaskResult([]byte(`{"request_id":"r1","status":"pending"}`))
	require.NoError(t, err)
	require.Equal(t, model.TaskStatusInProgress, queued.Status)
	done, err := adaptor.ParseTaskResult([]byte(`{"request_id":"r1","status":"done","video":{"url":"https://vidgen.x.ai/v.mp4","duration":5}}`))
	require.NoError(t, err)
	require.Equal(t, model.TaskStatusSuccess, done.Status)
	require.Equal(t, "https://vidgen.x.ai/v.mp4", done.Url)
	failed, err := adaptor.ParseTaskResult([]byte(`{"request_id":"r1","status":"failed","error":{"message":"blocked","code":"safety"}}`))
	require.NoError(t, err)
	require.Equal(t, model.TaskStatusFailure, failed.Status)
	require.Equal(t, "blocked", failed.Reason)
	_, err = adaptor.ParseTaskResult([]byte(`{"request_id":"r1","status":"done"}`))
	require.Error(t, err)
}

func TestDoResponseRejectsNilResponse(t *testing.T) {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	_, _, taskErr := (&TaskAdaptor{}).DoResponse(ctx, nil, &relaycommon.RelayInfo{})
	require.NotNil(t, taskErr)
	require.Error(t, taskErr.Error)
	require.Equal(t, http.StatusBadGateway, taskErr.StatusCode)
}

func TestBuildRequestURL(t *testing.T) {
	url, err := (&TaskAdaptor{}).BuildRequestURL(&relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ChannelBaseUrl: "https://api.x.ai/v1"}})
	require.NoError(t, err)
	require.Equal(t, "https://api.x.ai/v1/videos/generations", url)
}
