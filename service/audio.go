package service

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"math"
	"path/filepath"
	"strings"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

func normalizeAudioExtension(format string, mimeType string) string {
	format = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(format)), ".")
	switch format {
	case "mp3", "mpeg":
		return ".mp3"
	case "wav", "wave", "x-wav":
		return ".wav"
	}

	mimeType = strings.ToLower(strings.TrimSpace(strings.Split(mimeType, ";")[0]))
	switch mimeType {
	case "audio/mpeg", "audio/mp3":
		return ".mp3"
	case "audio/wav", "audio/wave", "audio/x-wav":
		return ".wav"
	}
	return ""
}

func getAudioDurationFromFileMeta(c *gin.Context, fileMeta *types.FileMeta) (float64, error) {
	if fileMeta == nil || fileMeta.Source == nil {
		return 0, fmt.Errorf("audio source is missing")
	}
	cachedData, err := LoadFileSource(c, fileMeta.Source, "audio_duration_billing")
	if err != nil {
		return 0, err
	}
	base64Data, err := cachedData.GetBase64Data()
	if err != nil {
		return 0, err
	}
	audioBytes, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return 0, fmt.Errorf("failed to decode audio data: %w", err)
	}
	ext := normalizeAudioExtension(fileMeta.AudioFormat, cachedData.MimeType)
	if ext == "" {
		return 0, fmt.Errorf("unsupported audio format: format=%q mime=%q", fileMeta.AudioFormat, cachedData.MimeType)
	}
	duration, err := common.GetAudioDuration(c.Request.Context(), bytes.NewReader(audioBytes), ext)
	if err != nil {
		return 0, err
	}
	if duration <= 0 || math.IsNaN(duration) || math.IsInf(duration, 0) {
		return 0, fmt.Errorf("invalid audio duration: %v", duration)
	}
	return duration, nil
}

func getMultipartAudioDuration(c *gin.Context) (float64, error) {
	multiForm, err := common.ParseMultipartFormReusable(c)
	if err != nil {
		return 0, fmt.Errorf("error parsing multipart form: %w", err)
	}
	fileHeaders := multiForm.File["file"]
	if len(fileHeaders) == 0 {
		return 0, fmt.Errorf("audio file is missing")
	}
	totalDuration := 0.0
	for _, fileHeader := range fileHeaders {
		file, err := fileHeader.Open()
		if err != nil {
			return 0, fmt.Errorf("error opening audio file: %w", err)
		}
		ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
		duration, durationErr := common.GetAudioDuration(c.Request.Context(), file, ext)
		closeErr := file.Close()
		if durationErr != nil {
			return 0, fmt.Errorf("error getting audio duration: %w", durationErr)
		}
		if closeErr != nil {
			return 0, fmt.Errorf("error closing audio file: %w", closeErr)
		}
		totalDuration += duration
	}
	return totalDuration, nil
}

// MeasureInputAudioDuration returns the billable input duration rounded up to
// whole seconds, matching MiMo's documented accounting precision.
func MeasureInputAudioDuration(c *gin.Context, meta *types.TokenCountMeta, info *relaycommon.RelayInfo) (int64, error) {
	if c == nil || c.Request == nil || info == nil {
		return 0, fmt.Errorf("audio duration billing context is incomplete")
	}

	var totalDuration float64
	var err error
	if info.RelayMode == relayconstant.RelayModeAudioTranscription || info.RelayMode == relayconstant.RelayModeAudioTranslation {
		totalDuration, err = getMultipartAudioDuration(c)
	} else {
		if meta == nil {
			return 0, fmt.Errorf("audio metadata is missing")
		}
		audioCount := 0
		for _, fileMeta := range meta.Files {
			if fileMeta == nil || fileMeta.FileType != types.FileTypeAudio {
				continue
			}
			audioCount++
			duration, durationErr := getAudioDurationFromFileMeta(c, fileMeta)
			if durationErr != nil {
				return 0, durationErr
			}
			totalDuration += duration
		}
		if audioCount == 0 {
			return 0, fmt.Errorf("input audio is missing")
		}
	}
	if err != nil {
		return 0, err
	}
	billedSeconds := int64(math.Ceil(totalDuration))
	if billedSeconds <= 0 {
		return 0, fmt.Errorf("input audio duration must be greater than zero")
	}
	return billedSeconds, nil
}

func parseAudio(audioBase64 string, format string) (duration float64, err error) {
	audioData, err := base64.StdEncoding.DecodeString(audioBase64)
	if err != nil {
		return 0, fmt.Errorf("base64 decode error: %v", err)
	}

	var samplesCount int
	var sampleRate int

	switch format {
	case "pcm16":
		samplesCount = len(audioData) / 2 // 16位 = 2字节每样本
		sampleRate = 24000                // 24kHz
	case "g711_ulaw", "g711_alaw":
		samplesCount = len(audioData) // 8位 = 1字节每样本
		sampleRate = 8000             // 8kHz
	default:
		samplesCount = len(audioData) // 8位 = 1字节每样本
		sampleRate = 8000             // 8kHz
	}

	duration = float64(samplesCount) / float64(sampleRate)
	return duration, nil
}

func DecodeBase64AudioData(audioBase64 string) (string, error) {
	// 检查并移除 data:audio/xxx;base64, 前缀
	idx := strings.Index(audioBase64, ",")
	if idx != -1 {
		audioBase64 = audioBase64[idx+1:]
	}

	// 解码 Base64 数据
	_, err := base64.StdEncoding.DecodeString(audioBase64)
	if err != nil {
		return "", fmt.Errorf("base64 decode error: %v", err)
	}

	return audioBase64, nil
}
