package setting

import (
	"sync"

	"github.com/QuantumNous/new-api/common"
)

var Chats = []map[string]string{
	//{
	//	"ChatGPT Next Web 官方示例": "https://app.nextchat.dev/#/?settings={\"key\":\"{key}\",\"url\":\"{address}\"}",
	//},
	{
		"Cherry Studio": "cherrystudio://providers/api-keys?v=1&data={cherryConfig}",
	},
	{
		"流畅阅读": "fluentread",
	},
	{
		"Lobe Chat 官方示例": "https://chat-preview.lobehub.com/?settings={\"keyVaults\":{\"openai\":{\"apiKey\":\"{key}\",\"baseURL\":\"{address}/v1\"}}}",
	},
	{
		"AI as Workspace": "https://aiaw.app/set-provider?provider={\"type\":\"openai\",\"settings\":{\"apiKey\":\"{key}\",\"baseURL\":\"{address}/v1\",\"compatibility\":\"strict\"}}",
	},
	{
		"AMA 问天": "ama://set-api-key?server={address}&key={key}",
	},
	{
		"OpenCat": "opencat://team/join?domain={address}&token={key}",
	},
}
var chatsMutex sync.RWMutex

func UpdateChatsByJsonString(jsonString string) error {
	chatsMutex.Lock()
	defer chatsMutex.Unlock()
	Chats = make([]map[string]string, 0)
	return common.Unmarshal([]byte(jsonString), &Chats)
}

func Chats2JsonString() string {
	chatsMutex.RLock()
	defer chatsMutex.RUnlock()
	jsonBytes, err := common.Marshal(Chats)
	if err != nil {
		common.SysLog("error marshalling chats: " + err.Error())
		return "[]"
	}
	return string(jsonBytes)
}

// GetChatsCopy returns a deep copy of Chats for safe external consumption.
// Prefer this over reading the Chats variable directly.
func GetChatsCopy() []map[string]string {
	chatsMutex.RLock()
	defer chatsMutex.RUnlock()
	result := make([]map[string]string, len(Chats))
	for i, entry := range Chats {
		copied := make(map[string]string, len(entry))
		for k, v := range entry {
			copied[k] = v
		}
		result[i] = copied
	}
	return result
}
