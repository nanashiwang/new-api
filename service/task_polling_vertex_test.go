package service

import (
	"context"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"strings"
	"testing"
)

type vertexPollingAdaptor struct {
	mockAdaptor
	key string
}

func (a *vertexPollingAdaptor) FetchTask(_ string, key string, _ map[string]any, _ string) (*http.Response, error) {
	a.key = key
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"done":true,"response":{"videos":[{"bytesBase64Encoded":"dmlkZW8="}]}}`))}, nil
}
func (a *vertexPollingAdaptor) ParseTaskResult([]byte) (*relaycommon.TaskInfo, error) {
	return &relaycommon.TaskInfo{Status: model.TaskStatusSuccess, Url: "data:video/mp4;base64,dmlkZW8=", Progress: "100%"}, nil
}
func TestVertexPollingStoresProxyURLAndPreservesOriginalKey(t *testing.T) {
	truncate(t)
	task := makeTask(0, 993601, 0, 0, "wallet", 0)
	task.TaskID = "task_vertex_poll"
	task.PrivateData.Key = `{"project_id":"selected"}`
	task.PrivateData.BillingContext.PerCallBilling = true
	require.NoError(t, model.DB.Create(task).Error)
	channel := &model.Channel{Id: 993601, Type: constant.ChannelTypeVertexAi, Key: "wrong-key"}
	adaptor := &vertexPollingAdaptor{}
	require.NoError(t, updateVideoSingleTask(context.Background(), adaptor, channel, task.TaskID, map[string]*model.Task{task.TaskID: task}))
	var saved model.Task
	require.NoError(t, model.DB.First(&saved, task.ID).Error)
	require.Equal(t, model.TaskStatus(model.TaskStatusSuccess), saved.Status)
	require.Equal(t, task.PrivateData.Key, adaptor.key)
	require.Contains(t, saved.GetResultURL(), "/v1/videos/task_vertex_poll/content")
	require.NotContains(t, string(saved.Data), "dmlkZW8=")
	require.NotContains(t, saved.GetResultURL(), "base64")
}
