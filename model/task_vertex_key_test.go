package model

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
	"strconv"
	"testing"
)

func TestTaskPersistsVertexSelectedKeyPrivately(t *testing.T) {
	truncateTables(t)
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeVertexAi, ApiKey: `{"project_id":"selected-project"}`}}
	task := InitTask(constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeVertexAi)), info)
	require.NoError(t, DB.Create(task).Error)
	var saved Task
	require.NoError(t, DB.First(&saved, task.ID).Error)
	require.Equal(t, info.ApiKey, saved.PrivateData.Key)
	payload, err := common.Marshal(saved)
	require.NoError(t, err)
	require.NotContains(t, string(payload), "selected-project")
}
