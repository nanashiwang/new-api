package controller

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"

	"github.com/stretchr/testify/require"
)

func decodeMiMoTTSTestAudio(t *testing.T, request dto.Request) (*dto.GeneralOpenAIRequest, map[string]any) {
	t.Helper()
	openAIRequest, ok := request.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.NotEmpty(t, openAIRequest.Audio)

	audio := make(map[string]any)
	require.NoError(t, common.Unmarshal(openAIRequest.Audio, &audio))
	return openAIRequest, audio
}

func TestBuildMiMoTTSTestRequestUsesProviderSpecificPayloads(t *testing.T) {
	tests := []struct {
		name             string
		modelName        string
		wantKind         miMoTTSTestKind
		wantStream       bool
		wantVoice        string
		wantVoiceDataURI bool
	}{
		{
			name:       "preset voice keeps explicit stream test",
			modelName:  "mimo-v2.5-tts",
			wantKind:   miMoTTSTestPresetVoice,
			wantStream: true,
			wantVoice:  "冰糖",
		},
		{
			name:       "namespaced voice design is forced non-stream",
			modelName:  "xiaomi/mimo-v2.5-tts-voicedesign",
			wantKind:   miMoTTSTestVoiceDesign,
			wantStream: false,
		},
		{
			name:             "voice clone embeds a valid reference wav",
			modelName:        "mimo-v2.5-tts-voiceclone",
			wantKind:         miMoTTSTestVoiceClone,
			wantStream:       false,
			wantVoiceDataURI: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request, ok := buildMiMoTTSTestRequest(test.modelName, "", nil, true)
			require.True(t, ok)
			openAIRequest, audio := decodeMiMoTTSTestAudio(t, request)

			require.Equal(t, test.wantKind, classifyMiMoTTSTestModel(test.modelName))
			require.Equal(t, test.modelName, openAIRequest.Model)
			require.Equal(t, test.wantStream, openAIRequest.Stream)
			require.Zero(t, openAIRequest.MaxTokens)
			require.Zero(t, openAIRequest.MaxCompletionTokens)
			require.Nil(t, openAIRequest.StreamOptions)
			require.Len(t, openAIRequest.Messages, 2)
			require.Equal(t, "user", openAIRequest.Messages[0].Role)
			require.Equal(t, "assistant", openAIRequest.Messages[1].Role)
			require.NotEmpty(t, openAIRequest.Messages[1].Content)
			require.Equal(t, "wav", audio["format"])

			voice, _ := audio["voice"].(string)
			if test.wantVoice != "" {
				require.Equal(t, test.wantVoice, voice)
			}
			if test.wantVoiceDataURI {
				const prefix = "data:audio/wav;base64,"
				require.True(t, strings.HasPrefix(voice, prefix))
				decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(voice, prefix))
				require.NoError(t, err)
				require.Greater(t, len(decoded), 44)
				require.Equal(t, "RIFF", string(decoded[:4]))
				require.Equal(t, "WAVE", string(decoded[8:12]))
			}
		})
	}
}

func TestBuildMiMoTTSTestRequestRecognizesMappedAlias(t *testing.T) {
	mapping := `{"tts-alias":"xiaomi/mimo-v2.5-tts-voicedesign"}`
	channel := &model.Channel{ModelMapping: &mapping}

	request, ok := buildMiMoTTSTestRequest("tts-alias", string(constant.EndpointTypeOpenAI), channel, true)
	require.True(t, ok)
	openAIRequest, _ := decodeMiMoTTSTestAudio(t, request)
	require.Equal(t, "tts-alias", openAIRequest.Model)
	require.False(t, openAIRequest.Stream)
}

func TestBuildMiMoTTSTestRequestDoesNotHijackOtherModelsOrEndpoints(t *testing.T) {
	for _, modelName := range []string{
		"mimo-v2.5",
		"mimo-v2.5-tts-extra",
		"notmimo-v2.5-tts",
		"mimosa-v2.5-tts",
	} {
		request, ok := buildMiMoTTSTestRequest(modelName, "", nil, false)
		require.False(t, ok, modelName)
		require.Nil(t, request, modelName)
	}

	request, ok := buildMiMoTTSTestRequest("mimo-v2.5-tts", string(constant.EndpointTypeEmbeddings), nil, false)
	require.False(t, ok)
	require.Nil(t, request)
}

func TestResolveChannelTestMappedModelNameTerminatesOnInvalidMappings(t *testing.T) {
	malformed := `{invalid`
	require.Equal(t, "tts-alias", resolveChannelTestMappedModelName(&model.Channel{ModelMapping: &malformed}, "tts-alias"))

	cycle := `{"tts-alias":"next-alias","next-alias":"tts-alias"}`
	require.Equal(t, "next-alias", resolveChannelTestMappedModelName(&model.Channel{ModelMapping: &cycle}, "tts-alias"))
}

func TestBuildChannelStyleTestExecutionForcesOneShotMiMoTTSNonStream(t *testing.T) {
	stream := true
	for _, modelName := range []string{"mimo-v2.5-tts-voicedesign", "mimo-v2.5-tts-voiceclone"} {
		execution := buildChannelStyleTestExecution(&model.Channel{}, modelName, "", &stream)
		require.False(t, execution.isStream, modelName)
		request, ok := execution.request.(*dto.GeneralOpenAIRequest)
		require.True(t, ok)
		require.False(t, request.Stream)
	}

	execution := buildChannelStyleTestExecution(&model.Channel{}, "mimo-v2.5-tts", "", &stream)
	require.True(t, execution.isStream)
	request, ok := execution.request.(*dto.GeneralOpenAIRequest)
	require.True(t, ok)
	require.True(t, request.Stream)
}

func TestChannelTestResponseForLogOmitsMiMoTTSAudio(t *testing.T) {
	body := []byte(`{"choices":[{"message":{"audio":{"data":"sensitive-large-audio"}}}]}`)
	redacted := channelTestResponseForLog("mimo-v2.5-tts-voiceclone", body)
	require.NotContains(t, redacted, "sensitive-large-audio")
	require.Contains(t, redacted, "response omitted")
	require.Contains(t, redacted, "bytes")

	require.Equal(t, string(body), channelTestResponseForLog("mimo-v2.5", body))
}
