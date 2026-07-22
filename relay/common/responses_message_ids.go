package common

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"

	basecommon "github.com/QuantumNous/new-api/common"
)

func NormalizeResponsesMessageIDs(input json.RawMessage) (json.RawMessage, int, error) {
	if len(input) == 0 || basecommon.GetJsonType(input) != "array" {
		return input, 0, nil
	}

	var items []map[string]json.RawMessage
	if err := basecommon.Unmarshal(input, &items); err != nil {
		return nil, 0, err
	}

	normalized := 0
	for _, item := range items {
		if rawJSONString(item["type"]) != "message" {
			continue
		}
		id := rawJSONString(item["id"])
		if !isCCSwitchMessageID(id) {
			continue
		}
		encodedID, err := basecommon.Marshal(normalizeResponsesMessageID(id))
		if err != nil {
			return nil, 0, err
		}
		item["id"] = encodedID
		normalized++
	}
	if normalized == 0 {
		return input, 0, nil
	}

	output, err := basecommon.Marshal(items)
	if err != nil {
		return nil, 0, err
	}
	return output, normalized, nil
}

func NormalizeResponsesMessageIDsInJSON(payload []byte) ([]byte, int, error) {
	if len(payload) == 0 {
		return payload, 0, nil
	}

	var body map[string]json.RawMessage
	if err := basecommon.Unmarshal(payload, &body); err != nil {
		return nil, 0, err
	}
	input, exists := body["input"]
	if !exists {
		return payload, 0, nil
	}

	normalizedInput, count, err := NormalizeResponsesMessageIDs(input)
	if err != nil || count == 0 {
		return payload, count, err
	}
	body["input"] = normalizedInput

	output, err := basecommon.Marshal(body)
	if err != nil {
		return nil, 0, err
	}
	return output, count, nil
}

func rawJSONString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var value string
	if err := basecommon.Unmarshal(raw, &value); err != nil {
		return ""
	}
	return value
}

func isCCSwitchMessageID(id string) bool {
	if !strings.HasPrefix(id, "resp_") {
		return false
	}
	marker := strings.LastIndex(id, "_msg")
	if marker < 0 {
		return false
	}
	suffix := id[marker+len("_msg"):]
	if suffix == "" {
		return true
	}
	if !strings.HasPrefix(suffix, "_") || len(suffix) == 1 {
		return false
	}
	_, err := strconv.ParseUint(suffix[1:], 10, 32)
	return err == nil
}

func normalizeResponsesMessageID(id string) string {
	digest := sha256.Sum256([]byte(id))
	return "msg_" + hex.EncodeToString(digest[:16])
}
