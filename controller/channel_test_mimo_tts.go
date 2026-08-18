package controller

import (
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
)

type miMoTTSTestKind int

const (
	miMoTTSTestNone miMoTTSTestKind = iota
	miMoTTSTestPresetVoice
	miMoTTSTestVoiceDesign
	miMoTTSTestVoiceClone
)

// The reference clip is Xiaomi's public voice-cloning example from its MiMo
// documentation:
// https://example-files.cnbj1.mi-fds.com/example-files/audio/audio_example.wav
// Embedding it keeps manual channel tests deterministic and avoids an extra
// runtime download that could fail independently of the channel.
//
//go:embed testdata/mimo_voiceclone_reference.wav
var miMoVoiceCloneTestReference []byte

var miMoVoiceCloneTestReferenceDataURI = "data:audio/wav;base64," + base64.StdEncoding.EncodeToString(miMoVoiceCloneTestReference)

func normalizeMiMoTTSTestModelName(modelName string) string {
	normalized := strings.ToLower(strings.TrimSpace(modelName))
	if slash := strings.LastIndex(normalized, "/"); slash >= 0 {
		normalized = normalized[slash+1:]
	}
	return normalized
}

func classifyMiMoTTSTestModel(modelName string) miMoTTSTestKind {
	switch normalizeMiMoTTSTestModelName(modelName) {
	case "mimo-v2.5-tts":
		return miMoTTSTestPresetVoice
	case "mimo-v2.5-tts-voicedesign":
		return miMoTTSTestVoiceDesign
	case "mimo-v2.5-tts-voiceclone":
		return miMoTTSTestVoiceClone
	default:
		return miMoTTSTestNone
	}
}

func resolveChannelTestMappedModelName(channel *model.Channel, modelName string) string {
	if channel == nil {
		return modelName
	}
	mappingJSON := strings.TrimSpace(channel.GetModelMapping())
	if mappingJSON == "" || mappingJSON == "{}" {
		return modelName
	}

	mapping := make(map[string]string)
	if err := common.UnmarshalJsonStr(mappingJSON, &mapping); err != nil {
		return modelName
	}

	current := modelName
	visited := map[string]struct{}{current: {}}
	for {
		next, ok := mapping[current]
		if !ok || next == "" {
			return current
		}
		if _, exists := visited[next]; exists {
			// The relay's model-mapping validation will report the cycle later.
			return current
		}
		visited[next] = struct{}{}
		current = next
	}
}

func classifyChannelMiMoTTSTestModel(channel *model.Channel, modelName string) miMoTTSTestKind {
	if kind := classifyMiMoTTSTestModel(modelName); kind != miMoTTSTestNone {
		return kind
	}
	return classifyMiMoTTSTestModel(resolveChannelTestMappedModelName(channel, modelName))
}

func isMiMoTTSChatEndpoint(endpointType string) bool {
	return endpointType == "" || constant.EndpointType(endpointType) == constant.EndpointTypeOpenAI
}

func marshalMiMoTTSTestAudio(value any) json.RawMessage {
	data, err := common.Marshal(value)
	if err != nil {
		// These payloads contain only primitive values, so this is defensive.
		return json.RawMessage(`{}`)
	}
	return data
}

func buildMiMoTTSTestRequest(modelName string, endpointType string, channel *model.Channel, isStream bool) (dto.Request, bool) {
	if !isMiMoTTSChatEndpoint(endpointType) {
		return nil, false
	}

	kind := classifyChannelMiMoTTSTestModel(channel, modelName)
	if kind == miMoTTSTestNone {
		return nil, false
	}

	request := &dto.GeneralOpenAIRequest{
		Model:  modelName,
		Stream: isStream,
	}

	switch kind {
	case miMoTTSTestPresetVoice:
		request.Messages = []dto.Message{
			{Role: "user", Content: "请使用自然、清晰的中文女声，语速适中。"},
			{Role: "assistant", Content: "你好，这是语音测试。"},
		}
		request.Audio = marshalMiMoTTSTestAudio(map[string]any{
			"format": "wav",
			"voice":  "冰糖",
		})
	case miMoTTSTestVoiceDesign:
		request.Stream = false
		request.Messages = []dto.Message{
			{Role: "user", Content: "年轻女性声音，温柔自然、清晰有亲和力，语速适中。"},
			{Role: "assistant", Content: "你好，这是音色设计测试。"},
		}
		request.Audio = marshalMiMoTTSTestAudio(map[string]any{
			"format": "wav",
		})
	case miMoTTSTestVoiceClone:
		request.Stream = false
		request.Messages = []dto.Message{
			{Role: "user", Content: "保持参考声音的自然语气和正常语速。"},
			{Role: "assistant", Content: "你好，这是音色克隆测试。"},
		}
		request.Audio = marshalMiMoTTSTestAudio(map[string]any{
			"format": "wav",
			"voice":  miMoVoiceCloneTestReferenceDataURI,
		})
	}

	return request, true
}

func forceNonStreamMiMoTTSTest(channel *model.Channel, modelName string) bool {
	switch classifyChannelMiMoTTSTestModel(channel, modelName) {
	case miMoTTSTestVoiceDesign, miMoTTSTestVoiceClone:
		return true
	default:
		return false
	}
}

func channelTestResponseForLog(modelName string, responseBody []byte) string {
	if classifyMiMoTTSTestModel(modelName) != miMoTTSTestNone {
		return fmt.Sprintf("<MiMo TTS response omitted: %d bytes>", len(responseBody))
	}
	return string(responseBody)
}
