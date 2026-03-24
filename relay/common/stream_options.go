package common

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
)

// NormalizeGeneralOpenAIStreamOptions 仅在流式请求且通道声明支持时保留
// stream_options；其余情况统一移除。
func NormalizeGeneralOpenAIStreamOptions(request *dto.GeneralOpenAIRequest, supportStreamOptions bool, forceStreamOption bool) {
	if request == nil {
		return
	}

	if !supportStreamOptions || !request.Stream {
		request.StreamOptions = nil
		return
	}

	if forceStreamOption && request.StreamOptions == nil {
		request.StreamOptions = &dto.StreamOptions{
			IncludeUsage: true,
		}
	}
}

// NormalizeResponsesStreamOptions 仅在流式 Responses 请求中保留
// stream_options。
func NormalizeResponsesStreamOptions(request *dto.OpenAIResponsesRequest) {
	if request == nil {
		return
	}
	if !request.Stream {
		request.StreamOptions = nil
	}
}

// NormalizeJSONStreamOptions 在最终发往上游的 JSON 中做兜底清洗：
// 当 stream 不明确为 true 时，移除整个 stream_options，避免无效参数组合
// 被 override 或透传逻辑重新带回上游。
func NormalizeJSONStreamOptions(jsonData []byte) ([]byte, error) {
	if len(jsonData) == 0 {
		return jsonData, nil
	}

	var data map[string]any
	if err := common.Unmarshal(jsonData, &data); err != nil {
		common.SysError("NormalizeJSONStreamOptions Unmarshal error: " + err.Error())
		return jsonData, nil
	}

	if keepStreamOptionsForJSON(data["stream"]) {
		return jsonData, nil
	}

	if _, exists := data["stream_options"]; !exists {
		return jsonData, nil
	}

	delete(data, "stream_options")

	jsonDataAfter, err := common.Marshal(data)
	if err != nil {
		common.SysError("NormalizeJSONStreamOptions Marshal error: " + err.Error())
		return jsonData, nil
	}
	return jsonDataAfter, nil
}

func keepStreamOptionsForJSON(streamValue any) bool {
	stream, ok := streamValue.(bool)
	return ok && stream
}
