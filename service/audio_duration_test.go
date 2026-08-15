package service

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

func makePCM16WAV(t *testing.T, seconds int) []byte {
	t.Helper()
	const sampleRate = 8000
	const channels = 1
	const bitsPerSample = 16
	dataSize := sampleRate * seconds * channels * (bitsPerSample / 8)
	buffer := bytes.NewBuffer(make([]byte, 0, 44+dataSize))
	write := func(value any) {
		if err := binary.Write(buffer, binary.LittleEndian, value); err != nil {
			t.Fatalf("write wav: %v", err)
		}
	}
	buffer.WriteString("RIFF")
	write(uint32(36 + dataSize))
	buffer.WriteString("WAVE")
	buffer.WriteString("fmt ")
	write(uint32(16))
	write(uint16(1))
	write(uint16(channels))
	write(uint32(sampleRate))
	write(uint32(sampleRate * channels * (bitsPerSample / 8)))
	write(uint16(channels * (bitsPerSample / 8)))
	write(uint16(bitsPerSample))
	buffer.WriteString("data")
	write(uint32(dataSize))
	buffer.Write(make([]byte, dataSize))
	return buffer.Bytes()
}

func newAudioDurationTestContext() *gin.Context {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	return ctx
}

func TestMeasureInputAudioDurationSupportsDataURIAndMultipleInputs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := newAudioDurationTestContext()
	first := base64.StdEncoding.EncodeToString(makePCM16WAV(t, 1))
	second := base64.StdEncoding.EncodeToString(makePCM16WAV(t, 2))
	meta := &types.TokenCountMeta{Files: []*types.FileMeta{
		{
			FileType: types.FileTypeAudio,
			Source:   types.NewBase64FileSource("data:audio/wav;base64,"+first, ""),
		},
		{
			FileType:    types.FileTypeAudio,
			AudioFormat: "wav",
			Source:      types.NewBase64FileSource(second, ""),
		},
	}}

	seconds, err := MeasureInputAudioDuration(ctx, meta, &relaycommon.RelayInfo{})
	if err != nil {
		t.Fatalf("MeasureInputAudioDuration returned error: %v", err)
	}
	if seconds != 3 {
		t.Fatalf("billed seconds = %d, want 3", seconds)
	}
	CleanupFileSources(ctx)
}

func TestEstimateRequestTokenMeasuresDurationWhenTokenCountingDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	originalCountToken := constant.CountToken
	originalPrices := ratio_setting.AudioDurationPrice2JSONString()
	constant.CountToken = false
	if err := ratio_setting.UpdateAudioDurationPriceByJSONString(`{"mimo-v2.5-asr":0.074}`); err != nil {
		t.Fatalf("set audio duration price: %v", err)
	}
	t.Cleanup(func() {
		constant.CountToken = originalCountToken
		if err := ratio_setting.UpdateAudioDurationPriceByJSONString(originalPrices); err != nil {
			t.Fatalf("restore audio duration price: %v", err)
		}
	})

	ctx := newAudioDurationTestContext()
	wavBase64 := base64.StdEncoding.EncodeToString(makePCM16WAV(t, 1))
	meta := &types.TokenCountMeta{Files: []*types.FileMeta{{
		FileType:    types.FileTypeAudio,
		AudioFormat: "wav",
		Source:      types.NewBase64FileSource(wavBase64, ""),
	}}}
	info := &relaycommon.RelayInfo{OriginModelName: "mimo-v2.5-asr"}

	tokens, err := EstimateRequestToken(ctx, meta, info)
	if err != nil {
		t.Fatalf("EstimateRequestToken returned error: %v", err)
	}
	if tokens != 0 {
		t.Fatalf("tokens = %d, want 0 when token counting is disabled", tokens)
	}
	if info.InputAudioDurationSeconds != 1 {
		t.Fatalf("duration seconds = %d, want 1", info.InputAudioDurationSeconds)
	}
	CleanupFileSources(ctx)
}

func TestMeasureInputAudioDurationRejectsInvalidAudio(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := newAudioDurationTestContext()
	meta := &types.TokenCountMeta{Files: []*types.FileMeta{{
		FileType:    types.FileTypeAudio,
		AudioFormat: "wav",
		Source:      types.NewBase64FileSource(base64.StdEncoding.EncodeToString([]byte("not a wav")), ""),
	}}}
	if _, err := MeasureInputAudioDuration(ctx, meta, &relaycommon.RelayInfo{}); err == nil {
		t.Fatal("invalid audio should be rejected")
	}
	CleanupFileSources(ctx)
}
