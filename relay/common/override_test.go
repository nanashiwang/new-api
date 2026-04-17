package common

import (
	"encoding/json"
	"reflect"
	"testing"

	common2 "github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/setting/model_setting"
)

func TestApplyParamOverrideTrimPrefix(t *testing.T) {
	// trim_prefix example:
	// {"operations":[{"path":"model","mode":"trim_prefix","value":"openai/"}]}
	input := []byte(`{"model":"openai/gpt-4","temperature":0.7}`)
	override := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"path":  "model",
				"mode":  "trim_prefix",
				"value": "openai/",
			},
		},
	}

	out, err := ApplyParamOverride(input, override, nil)
	if err != nil {
		t.Fatalf("ApplyParamOverride returned error: %v", err)
	}
	assertJSONEqual(t, `{"model":"gpt-4","temperature":0.7}`, string(out))
}

func TestApplyParamOverrideTrimSuffix(t *testing.T) {
	// trim_suffix example:
	// {"operations":[{"path":"model","mode":"trim_suffix","value":"-latest"}]}
	input := []byte(`{"model":"gpt-4-latest","temperature":0.7}`)
	override := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"path":  "model",
				"mode":  "trim_suffix",
				"value": "-latest",
			},
		},
	}

	out, err := ApplyParamOverride(input, override, nil)
	if err != nil {
		t.Fatalf("ApplyParamOverride returned error: %v", err)
	}
	assertJSONEqual(t, `{"model":"gpt-4","temperature":0.7}`, string(out))
}

func TestApplyParamOverrideTrimNoop(t *testing.T) {
	// trim_prefix no-op example:
	// {"operations":[{"path":"model","mode":"trim_prefix","value":"openai/"}]}
	input := []byte(`{"model":"gpt-4","temperature":0.7}`)
	override := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"path":  "model",
				"mode":  "trim_prefix",
				"value": "openai/",
			},
		},
	}

	out, err := ApplyParamOverride(input, override, nil)
	if err != nil {
		t.Fatalf("ApplyParamOverride returned error: %v", err)
	}
	assertJSONEqual(t, `{"model":"gpt-4","temperature":0.7}`, string(out))
}

func TestApplyParamOverrideTrimRequiresValue(t *testing.T) {
	// trim_prefix requires value example:
	// {"operations":[{"path":"model","mode":"trim_prefix"}]}
	input := []byte(`{"model":"gpt-4"}`)
	override := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"path": "model",
				"mode": "trim_prefix",
			},
		},
	}

	_, err := ApplyParamOverride(input, override, nil)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestApplyParamOverrideReplace(t *testing.T) {
	// replace example:
	// {"operations":[{"path":"model","mode":"replace","from":"openai/","to":""}]}
	input := []byte(`{"model":"openai/gpt-4o-mini","temperature":0.7}`)
	override := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"path": "model",
				"mode": "replace",
				"from": "openai/",
				"to":   "",
			},
		},
	}

	out, err := ApplyParamOverride(input, override, nil)
	if err != nil {
		t.Fatalf("ApplyParamOverride returned error: %v", err)
	}
	assertJSONEqual(t, `{"model":"gpt-4o-mini","temperature":0.7}`, string(out))
}

func TestApplyParamOverrideRegexReplace(t *testing.T) {
	// regex_replace example:
	// {"operations":[{"path":"model","mode":"regex_replace","from":"^gpt-","to":"openai/gpt-"}]}
	input := []byte(`{"model":"gpt-4o-mini","temperature":0.7}`)
	override := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"path": "model",
				"mode": "regex_replace",
				"from": "^gpt-",
				"to":   "openai/gpt-",
			},
		},
	}

	out, err := ApplyParamOverride(input, override, nil)
	if err != nil {
		t.Fatalf("ApplyParamOverride returned error: %v", err)
	}
	assertJSONEqual(t, `{"model":"openai/gpt-4o-mini","temperature":0.7}`, string(out))
}

func TestApplyParamOverrideReplaceRequiresFrom(t *testing.T) {
	// replace requires from example:
	// {"operations":[{"path":"model","mode":"replace"}]}
	input := []byte(`{"model":"gpt-4"}`)
	override := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"path": "model",
				"mode": "replace",
			},
		},
	}

	_, err := ApplyParamOverride(input, override, nil)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestApplyParamOverrideRegexReplaceRequiresPattern(t *testing.T) {
	// regex_replace requires from(pattern) example:
	// {"operations":[{"path":"model","mode":"regex_replace"}]}
	input := []byte(`{"model":"gpt-4"}`)
	override := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"path": "model",
				"mode": "regex_replace",
			},
		},
	}

	_, err := ApplyParamOverride(input, override, nil)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestApplyParamOverrideDelete(t *testing.T) {
	input := []byte(`{"model":"gpt-4","temperature":0.7}`)
	override := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"path": "temperature",
				"mode": "delete",
			},
		},
	}

	out, err := ApplyParamOverride(input, override, nil)
	if err != nil {
		t.Fatalf("ApplyParamOverride returned error: %v", err)
	}

	var got map[string]interface{}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("failed to unmarshal output JSON: %v", err)
	}
	if _, exists := got["temperature"]; exists {
		t.Fatalf("expected temperature to be deleted")
	}
}

func TestApplyParamOverrideSet(t *testing.T) {
	input := []byte(`{"model":"gpt-4","temperature":0.7}`)
	override := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"path":  "temperature",
				"mode":  "set",
				"value": 0.1,
			},
		},
	}

	out, err := ApplyParamOverride(input, override, nil)
	if err != nil {
		t.Fatalf("ApplyParamOverride returned error: %v", err)
	}
	assertJSONEqual(t, `{"model":"gpt-4","temperature":0.1}`, string(out))
}

func TestApplyParamOverrideSetKeepOrigin(t *testing.T) {
	input := []byte(`{"model":"gpt-4","temperature":0.7}`)
	override := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"path":        "temperature",
				"mode":        "set",
				"value":       0.1,
				"keep_origin": true,
			},
		},
	}

	out, err := ApplyParamOverride(input, override, nil)
	if err != nil {
		t.Fatalf("ApplyParamOverride returned error: %v", err)
	}
	assertJSONEqual(t, `{"model":"gpt-4","temperature":0.7}`, string(out))
}

func TestApplyParamOverrideMove(t *testing.T) {
	input := []byte(`{"model":"gpt-4","meta":{"x":1}}`)
	override := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"mode": "move",
				"from": "model",
				"to":   "meta.model",
			},
		},
	}

	out, err := ApplyParamOverride(input, override, nil)
	if err != nil {
		t.Fatalf("ApplyParamOverride returned error: %v", err)
	}
	assertJSONEqual(t, `{"meta":{"x":1,"model":"gpt-4"}}`, string(out))
}

func TestApplyParamOverrideMoveMissingSource(t *testing.T) {
	input := []byte(`{"meta":{"x":1}}`)
	override := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"mode": "move",
				"from": "model",
				"to":   "meta.model",
			},
		},
	}

	_, err := ApplyParamOverride(input, override, nil)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestApplyParamOverridePrependAppendString(t *testing.T) {
	input := []byte(`{"model":"gpt-4"}`)
	override := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"path":  "model",
				"mode":  "prepend",
				"value": "openai/",
			},
			map[string]interface{}{
				"path":  "model",
				"mode":  "append",
				"value": "-latest",
			},
		},
	}

	out, err := ApplyParamOverride(input, override, nil)
	if err != nil {
		t.Fatalf("ApplyParamOverride returned error: %v", err)
	}
	assertJSONEqual(t, `{"model":"openai/gpt-4-latest"}`, string(out))
}

func TestApplyParamOverridePrependAppendArray(t *testing.T) {
	input := []byte(`{"arr":[1,2]}`)
	override := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"path":  "arr",
				"mode":  "prepend",
				"value": 0,
			},
			map[string]interface{}{
				"path":  "arr",
				"mode":  "append",
				"value": []interface{}{3, 4},
			},
		},
	}

	out, err := ApplyParamOverride(input, override, nil)
	if err != nil {
		t.Fatalf("ApplyParamOverride returned error: %v", err)
	}
	assertJSONEqual(t, `{"arr":[0,1,2,3,4]}`, string(out))
}

func TestApplyParamOverrideAppendObjectMergeKeepOrigin(t *testing.T) {
	input := []byte(`{"obj":{"a":1}}`)
	override := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"path":        "obj",
				"mode":        "append",
				"keep_origin": true,
				"value": map[string]interface{}{
					"a": 2,
					"b": 3,
				},
			},
		},
	}

	out, err := ApplyParamOverride(input, override, nil)
	if err != nil {
		t.Fatalf("ApplyParamOverride returned error: %v", err)
	}
	assertJSONEqual(t, `{"obj":{"a":1,"b":3}}`, string(out))
}

func TestApplyParamOverrideAppendObjectMergeOverride(t *testing.T) {
	input := []byte(`{"obj":{"a":1}}`)
	override := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"path": "obj",
				"mode": "append",
				"value": map[string]interface{}{
					"a": 2,
					"b": 3,
				},
			},
		},
	}

	out, err := ApplyParamOverride(input, override, nil)
	if err != nil {
		t.Fatalf("ApplyParamOverride returned error: %v", err)
	}
	assertJSONEqual(t, `{"obj":{"a":2,"b":3}}`, string(out))
}

func TestApplyParamOverrideConditionORDefault(t *testing.T) {
	input := []byte(`{"model":"gpt-4","temperature":0.7}`)
	override := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"path":  "temperature",
				"mode":  "set",
				"value": 0.1,
				"conditions": []interface{}{
					map[string]interface{}{
						"path":  "model",
						"mode":  "prefix",
						"value": "gpt",
					},
					map[string]interface{}{
						"path":  "model",
						"mode":  "prefix",
						"value": "claude",
					},
				},
			},
		},
	}

	out, err := ApplyParamOverride(input, override, nil)
	if err != nil {
		t.Fatalf("ApplyParamOverride returned error: %v", err)
	}
	assertJSONEqual(t, `{"model":"gpt-4","temperature":0.1}`, string(out))
}

func TestApplyParamOverrideConditionAND(t *testing.T) {
	input := []byte(`{"model":"gpt-4","temperature":0.7}`)
	override := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"path":  "temperature",
				"mode":  "set",
				"value": 0.1,
				"logic": "AND",
				"conditions": []interface{}{
					map[string]interface{}{
						"path":  "model",
						"mode":  "prefix",
						"value": "gpt",
					},
					map[string]interface{}{
						"path":  "temperature",
						"mode":  "gt",
						"value": 0.5,
					},
				},
			},
		},
	}

	out, err := ApplyParamOverride(input, override, nil)
	if err != nil {
		t.Fatalf("ApplyParamOverride returned error: %v", err)
	}
	assertJSONEqual(t, `{"model":"gpt-4","temperature":0.1}`, string(out))
}

func TestApplyParamOverrideConditionInvert(t *testing.T) {
	input := []byte(`{"model":"gpt-4","temperature":0.7}`)
	override := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"path":  "temperature",
				"mode":  "set",
				"value": 0.1,
				"conditions": []interface{}{
					map[string]interface{}{
						"path":   "model",
						"mode":   "prefix",
						"value":  "gpt",
						"invert": true,
					},
				},
			},
		},
	}

	out, err := ApplyParamOverride(input, override, nil)
	if err != nil {
		t.Fatalf("ApplyParamOverride returned error: %v", err)
	}
	assertJSONEqual(t, `{"model":"gpt-4","temperature":0.7}`, string(out))
}

func TestApplyParamOverrideConditionPassMissingKey(t *testing.T) {
	input := []byte(`{"temperature":0.7}`)
	override := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"path":  "temperature",
				"mode":  "set",
				"value": 0.1,
				"conditions": []interface{}{
					map[string]interface{}{
						"path":             "model",
						"mode":             "prefix",
						"value":            "gpt",
						"pass_missing_key": true,
					},
				},
			},
		},
	}

	out, err := ApplyParamOverride(input, override, nil)
	if err != nil {
		t.Fatalf("ApplyParamOverride returned error: %v", err)
	}
	assertJSONEqual(t, `{"temperature":0.1}`, string(out))
}

func TestApplyParamOverrideConditionFromContext(t *testing.T) {
	input := []byte(`{"temperature":0.7}`)
	override := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"path":  "temperature",
				"mode":  "set",
				"value": 0.1,
				"conditions": []interface{}{
					map[string]interface{}{
						"path":  "model",
						"mode":  "prefix",
						"value": "gpt",
					},
				},
			},
		},
	}
	ctx := map[string]interface{}{
		"model": "gpt-4",
	}

	out, err := ApplyParamOverride(input, override, ctx)
	if err != nil {
		t.Fatalf("ApplyParamOverride returned error: %v", err)
	}
	assertJSONEqual(t, `{"temperature":0.1}`, string(out))
}

func TestApplyParamOverrideNegativeIndexPath(t *testing.T) {
	input := []byte(`{"arr":[{"model":"a"},{"model":"b"}]}`)
	override := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"path":  "arr.-1.model",
				"mode":  "set",
				"value": "c",
			},
		},
	}

	out, err := ApplyParamOverride(input, override, nil)
	if err != nil {
		t.Fatalf("ApplyParamOverride returned error: %v", err)
	}
	assertJSONEqual(t, `{"arr":[{"model":"a"},{"model":"c"}]}`, string(out))
}

func TestApplyParamOverrideRegexReplaceInvalidPattern(t *testing.T) {
	// regex_replace invalid pattern example:
	// {"operations":[{"path":"model","mode":"regex_replace","from":"(","to":"x"}]}
	input := []byte(`{"model":"gpt-4"}`)
	override := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"path": "model",
				"mode": "regex_replace",
				"from": "(",
				"to":   "x",
			},
		},
	}

	_, err := ApplyParamOverride(input, override, nil)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestApplyParamOverrideCopy(t *testing.T) {
	// copy example:
	// {"operations":[{"mode":"copy","from":"model","to":"original_model"}]}
	input := []byte(`{"model":"gpt-4","temperature":0.7}`)
	override := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"mode": "copy",
				"from": "model",
				"to":   "original_model",
			},
		},
	}

	out, err := ApplyParamOverride(input, override, nil)
	if err != nil {
		t.Fatalf("ApplyParamOverride returned error: %v", err)
	}
	assertJSONEqual(t, `{"model":"gpt-4","original_model":"gpt-4","temperature":0.7}`, string(out))
}

func TestApplyParamOverrideCopyMissingSource(t *testing.T) {
	// copy missing source example:
	// {"operations":[{"mode":"copy","from":"model","to":"original_model"}]}
	input := []byte(`{"temperature":0.7}`)
	override := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"mode": "copy",
				"from": "model",
				"to":   "original_model",
			},
		},
	}

	_, err := ApplyParamOverride(input, override, nil)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestApplyParamOverrideCopyRequiresFromTo(t *testing.T) {
	// copy requires from/to example:
	// {"operations":[{"mode":"copy"}]}
	input := []byte(`{"model":"gpt-4"}`)
	override := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"mode": "copy",
			},
		},
	}

	_, err := ApplyParamOverride(input, override, nil)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestApplyParamOverrideEnsurePrefix(t *testing.T) {
	// ensure_prefix example:
	// {"operations":[{"path":"model","mode":"ensure_prefix","value":"openai/"}]}
	input := []byte(`{"model":"gpt-4"}`)
	override := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"path":  "model",
				"mode":  "ensure_prefix",
				"value": "openai/",
			},
		},
	}

	out, err := ApplyParamOverride(input, override, nil)
	if err != nil {
		t.Fatalf("ApplyParamOverride returned error: %v", err)
	}
	assertJSONEqual(t, `{"model":"openai/gpt-4"}`, string(out))
}

func TestApplyParamOverrideEnsurePrefixNoop(t *testing.T) {
	// ensure_prefix no-op example:
	// {"operations":[{"path":"model","mode":"ensure_prefix","value":"openai/"}]}
	input := []byte(`{"model":"openai/gpt-4"}`)
	override := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"path":  "model",
				"mode":  "ensure_prefix",
				"value": "openai/",
			},
		},
	}

	out, err := ApplyParamOverride(input, override, nil)
	if err != nil {
		t.Fatalf("ApplyParamOverride returned error: %v", err)
	}
	assertJSONEqual(t, `{"model":"openai/gpt-4"}`, string(out))
}

func TestApplyParamOverrideEnsureSuffix(t *testing.T) {
	// ensure_suffix example:
	// {"operations":[{"path":"model","mode":"ensure_suffix","value":"-latest"}]}
	input := []byte(`{"model":"gpt-4"}`)
	override := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"path":  "model",
				"mode":  "ensure_suffix",
				"value": "-latest",
			},
		},
	}

	out, err := ApplyParamOverride(input, override, nil)
	if err != nil {
		t.Fatalf("ApplyParamOverride returned error: %v", err)
	}
	assertJSONEqual(t, `{"model":"gpt-4-latest"}`, string(out))
}

func TestApplyParamOverrideEnsureSuffixNoop(t *testing.T) {
	// ensure_suffix no-op example:
	// {"operations":[{"path":"model","mode":"ensure_suffix","value":"-latest"}]}
	input := []byte(`{"model":"gpt-4-latest"}`)
	override := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"path":  "model",
				"mode":  "ensure_suffix",
				"value": "-latest",
			},
		},
	}

	out, err := ApplyParamOverride(input, override, nil)
	if err != nil {
		t.Fatalf("ApplyParamOverride returned error: %v", err)
	}
	assertJSONEqual(t, `{"model":"gpt-4-latest"}`, string(out))
}

func TestApplyParamOverrideEnsureRequiresValue(t *testing.T) {
	// ensure_prefix requires value example:
	// {"operations":[{"path":"model","mode":"ensure_prefix"}]}
	input := []byte(`{"model":"gpt-4"}`)
	override := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"path": "model",
				"mode": "ensure_prefix",
			},
		},
	}

	_, err := ApplyParamOverride(input, override, nil)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestApplyParamOverrideTrimSpace(t *testing.T) {
	// trim_space example:
	// {"operations":[{"path":"model","mode":"trim_space"}]}
	input := []byte("{\"model\":\"  gpt-4 \\n\"}")
	override := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"path": "model",
				"mode": "trim_space",
			},
		},
	}

	out, err := ApplyParamOverride(input, override, nil)
	if err != nil {
		t.Fatalf("ApplyParamOverride returned error: %v", err)
	}
	assertJSONEqual(t, `{"model":"gpt-4"}`, string(out))
}

func TestApplyParamOverrideToLower(t *testing.T) {
	// to_lower example:
	// {"operations":[{"path":"model","mode":"to_lower"}]}
	input := []byte(`{"model":"GPT-4"}`)
	override := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"path": "model",
				"mode": "to_lower",
			},
		},
	}

	out, err := ApplyParamOverride(input, override, nil)
	if err != nil {
		t.Fatalf("ApplyParamOverride returned error: %v", err)
	}
	assertJSONEqual(t, `{"model":"gpt-4"}`, string(out))
}

func TestApplyParamOverrideToUpper(t *testing.T) {
	// to_upper example:
	// {"operations":[{"path":"model","mode":"to_upper"}]}
	input := []byte(`{"model":"gpt-4"}`)
	override := map[string]interface{}{
		"operations": []interface{}{
			map[string]interface{}{
				"path": "model",
				"mode": "to_upper",
			},
		},
	}

	out, err := ApplyParamOverride(input, override, nil)
	if err != nil {
		t.Fatalf("ApplyParamOverride returned error: %v", err)
	}
	assertJSONEqual(t, `{"model":"GPT-4"}`, string(out))
}

func TestRemoveDisabledFieldsSkipWhenChannelPassThroughEnabled(t *testing.T) {
	input := `{
		"service_tier":"flex",
		"safety_identifier":"user-123",
		"store":true,
		"stream_options":{"include_obfuscation":false}
	}`
	settings := dto.ChannelOtherSettings{}

	out, err := RemoveDisabledFields([]byte(input), settings, true)
	if err != nil {
		t.Fatalf("RemoveDisabledFields returned error: %v", err)
	}
	assertJSONEqual(t, input, string(out))
}

func TestRemoveDisabledFieldsSkipWhenGlobalPassThroughEnabled(t *testing.T) {
	original := model_setting.GetGlobalSettings().PassThroughRequestEnabled
	model_setting.GetGlobalSettings().PassThroughRequestEnabled = true
	t.Cleanup(func() {
		model_setting.GetGlobalSettings().PassThroughRequestEnabled = original
	})

	input := `{
		"service_tier":"flex",
		"safety_identifier":"user-123",
		"stream_options":{"include_obfuscation":false}
	}`
	settings := dto.ChannelOtherSettings{}

	out, err := RemoveDisabledFields([]byte(input), settings, false)
	if err != nil {
		t.Fatalf("RemoveDisabledFields returned error: %v", err)
	}
	assertJSONEqual(t, input, string(out))
}

func TestRemoveDisabledFieldsDefaultFiltering(t *testing.T) {
	input := `{
		"service_tier":"flex",
		"inference_geo":"eu",
		"safety_identifier":"user-123",
		"store":true,
		"stream_options":{"include_obfuscation":false}
	}`
	settings := dto.ChannelOtherSettings{}

	out, err := RemoveDisabledFields([]byte(input), settings, false)
	if err != nil {
		t.Fatalf("RemoveDisabledFields returned error: %v", err)
	}
	assertJSONEqual(t, `{"store":true}`, string(out))
}

func TestRemoveDisabledFieldsAllowInferenceGeo(t *testing.T) {
	input := `{
		"inference_geo":"eu",
		"store":true
	}`
	settings := dto.ChannelOtherSettings{
		AllowInferenceGeo: true,
	}

	out, err := RemoveDisabledFields([]byte(input), settings, false)
	if err != nil {
		t.Fatalf("RemoveDisabledFields returned error: %v", err)
	}
	assertJSONEqual(t, `{"inference_geo":"eu","store":true}`, string(out))
}

func TestApplyParamOverrideWithRelayInfoRecordsOperationAuditInDebugMode(t *testing.T) {
	originalDebugEnabled := common2.DebugEnabled
	common2.DebugEnabled = true
	t.Cleanup(func() {
		common2.DebugEnabled = originalDebugEnabled
	})

	info := &RelayInfo{
		ChannelMeta: &ChannelMeta{
			ParamOverride: map[string]interface{}{
				"operations": []interface{}{
					map[string]interface{}{
						"mode": "copy",
						"from": "metadata.target_model",
						"to":   "model",
					},
					map[string]interface{}{
						"mode":  "set",
						"path":  "service_tier",
						"value": "flex",
					},
					map[string]interface{}{
						"mode":  "set",
						"path":  "temperature",
						"value": 0.1,
					},
				},
			},
		},
	}

	out, err := ApplyParamOverrideWithRelayInfo([]byte(`{
		"model":"gpt-4.1",
		"temperature":0.7,
		"metadata":{"target_model":"gpt-4.1-mini"}
	}`), info)
	if err != nil {
		t.Fatalf("ApplyParamOverrideWithRelayInfo returned error: %v", err)
	}
	assertJSONEqual(t, `{
		"model":"gpt-4.1-mini",
		"temperature":0.1,
		"service_tier":"flex",
		"metadata":{"target_model":"gpt-4.1-mini"}
	}`, string(out))

	expected := []string{
		"copy metadata.target_model -> model",
		"set service_tier = flex",
		"set temperature = 0.1",
	}
	if !reflect.DeepEqual(info.ParamOverrideAudit, expected) {
		t.Fatalf("unexpected param override audit, got %#v", info.ParamOverrideAudit)
	}
}

func TestApplyParamOverrideWithRelayInfoRecordsOnlyKeyOperationsWhenDebugDisabled(t *testing.T) {
	originalDebugEnabled := common2.DebugEnabled
	common2.DebugEnabled = false
	t.Cleanup(func() {
		common2.DebugEnabled = originalDebugEnabled
	})

	info := &RelayInfo{
		ChannelMeta: &ChannelMeta{
			ParamOverride: map[string]interface{}{
				"operations": []interface{}{
					map[string]interface{}{
						"mode": "copy",
						"from": "metadata.target_model",
						"to":   "model",
					},
					map[string]interface{}{
						"mode":  "set",
						"path":  "temperature",
						"value": 0.1,
					},
				},
			},
		},
	}

	_, err := ApplyParamOverrideWithRelayInfo([]byte(`{
		"model":"gpt-4.1",
		"temperature":0.7,
		"metadata":{"target_model":"gpt-4.1-mini"}
	}`), info)
	if err != nil {
		t.Fatalf("ApplyParamOverrideWithRelayInfo returned error: %v", err)
	}

	expected := []string{
		"copy metadata.target_model -> model",
	}
	if !reflect.DeepEqual(info.ParamOverrideAudit, expected) {
		t.Fatalf("unexpected param override audit, got %#v", info.ParamOverrideAudit)
	}
}

func TestApplyParamOverrideWithRelayInfoSetHeaderAppendsAnthropicBetaTokens(t *testing.T) {
	info := &RelayInfo{
		RequestHeaders: map[string]string{
			"Anthropic-Beta": "computer-use-2025-01-24",
		},
		ChannelMeta: &ChannelMeta{
			ParamOverride: map[string]interface{}{
				"operations": []interface{}{
					map[string]interface{}{
						"mode": "set_header",
						"path": "anthropic-beta",
						"value": map[string]interface{}{
							"$append": []interface{}{"context-1m-2025-08-07", "computer-use-2025-01-24"},
						},
					},
				},
			},
		},
	}

	out, err := ApplyParamOverrideWithRelayInfo([]byte(`{"messages":[{"role":"user","content":"hello"}]}`), info)
	if err != nil {
		t.Fatalf("ApplyParamOverrideWithRelayInfo returned error: %v", err)
	}

	assertJSONEqual(t, `{"messages":[{"role":"user","content":"hello"}]}`, string(out))
	if !info.UseRuntimeHeadersOverride {
		t.Fatal("expected runtime header override to be enabled")
	}

	got, ok := info.RuntimeHeadersOverride["anthropic-beta"]
	if !ok {
		t.Fatal("expected anthropic-beta runtime header override to be populated")
	}

	want := "computer-use-2025-01-24,context-1m-2025-08-07"
	if got != want {
		t.Fatalf("unexpected anthropic-beta runtime header override, want %q, got %v", want, got)
	}
}

func assertJSONEqual(t *testing.T, want, got string) {
	t.Helper()

	var wantObj interface{}
	var gotObj interface{}

	if err := json.Unmarshal([]byte(want), &wantObj); err != nil {
		t.Fatalf("failed to unmarshal want JSON: %v", err)
	}
	if err := json.Unmarshal([]byte(got), &gotObj); err != nil {
		t.Fatalf("failed to unmarshal got JSON: %v", err)
	}

	if !reflect.DeepEqual(wantObj, gotObj) {
		t.Fatalf("json not equal\nwant: %s\ngot:  %s", want, got)
	}
}
