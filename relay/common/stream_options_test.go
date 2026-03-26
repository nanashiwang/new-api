package common

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
)

func TestNormalizeGeneralOpenAIStreamOptionsRemovesForNonStream(t *testing.T) {
	req := &dto.GeneralOpenAIRequest{
		Stream:        false,
		StreamOptions: &dto.StreamOptions{IncludeUsage: true},
	}

	NormalizeGeneralOpenAIStreamOptions(req, true, true)

	if req.StreamOptions != nil {
		t.Fatalf("expected stream_options to be removed for non-stream request")
	}
}

func TestNormalizeGeneralOpenAIStreamOptionsPreservesExistingValue(t *testing.T) {
	req := &dto.GeneralOpenAIRequest{
		Stream:        true,
		StreamOptions: &dto.StreamOptions{IncludeUsage: false},
	}

	NormalizeGeneralOpenAIStreamOptions(req, true, true)

	if req.StreamOptions == nil {
		t.Fatalf("expected stream_options to remain present")
	}
	if req.StreamOptions.IncludeUsage {
		t.Fatalf("expected include_usage=false to be preserved")
	}
}

func TestNormalizeGeneralOpenAIStreamOptionsAddsDefaultOnlyWhenMissing(t *testing.T) {
	req := &dto.GeneralOpenAIRequest{
		Stream: true,
	}

	NormalizeGeneralOpenAIStreamOptions(req, true, true)

	if req.StreamOptions == nil {
		t.Fatalf("expected default stream_options to be added")
	}
	if !req.StreamOptions.IncludeUsage {
		t.Fatalf("expected default include_usage=true")
	}
}

func TestNormalizeResponsesStreamOptionsRemovesForNonStream(t *testing.T) {
	req := &dto.OpenAIResponsesRequest{
		Stream:        false,
		StreamOptions: &dto.StreamOptions{IncludeUsage: true},
	}

	NormalizeResponsesStreamOptions(req, true)

	if req.StreamOptions != nil {
		t.Fatalf("expected stream_options to be removed for non-stream responses request")
	}
}

func TestNormalizeResponsesStreamOptionsDropsIncludeUsage(t *testing.T) {
	req := &dto.OpenAIResponsesRequest{
		Stream:        true,
		StreamOptions: &dto.StreamOptions{IncludeUsage: true},
	}

	NormalizeResponsesStreamOptions(req, true)

	if req.StreamOptions != nil {
		t.Fatalf("expected responses stream_options to be removed when only include_usage was provided")
	}
}

func TestNormalizeResponsesStreamOptionsKeepsIncludeObfuscationOnly(t *testing.T) {
	req := &dto.OpenAIResponsesRequest{
		Stream: true,
		StreamOptions: &dto.StreamOptions{
			IncludeUsage:       true,
			IncludeObfuscation: true,
		},
	}

	NormalizeResponsesStreamOptions(req, true)

	if req.StreamOptions == nil {
		t.Fatalf("expected responses stream_options to remain present")
	}
	if req.StreamOptions.IncludeUsage {
		t.Fatalf("expected include_usage to be stripped for responses stream_options")
	}
	if !req.StreamOptions.IncludeObfuscation {
		t.Fatalf("expected include_obfuscation to be preserved")
	}
}

func TestNormalizeResponsesStreamOptionsRemovesWhenUnsupported(t *testing.T) {
	req := &dto.OpenAIResponsesRequest{
		Stream:        true,
		StreamOptions: &dto.StreamOptions{IncludeUsage: true},
	}

	NormalizeResponsesStreamOptions(req, false)

	if req.StreamOptions != nil {
		t.Fatalf("expected stream_options to be removed when responses stream options unsupported")
	}
}

func TestNormalizeJSONStreamOptionsRemovesWhenStreamFalse(t *testing.T) {
	input := []byte(`{"stream":false,"stream_options":{"include_usage":true},"model":"gpt-4o"}`)

	out, err := NormalizeJSONStreamOptions(input)
	if err != nil {
		t.Fatalf("NormalizeJSONStreamOptions returned error: %v", err)
	}

	assertJSONEqual(t, `{"stream":false,"model":"gpt-4o"}`, string(out))
}

func TestNormalizeJSONStreamOptionsPreservesWhenStreamTrue(t *testing.T) {
	input := []byte(`{"stream":true,"stream_options":{"include_usage":false},"model":"gpt-4o"}`)

	out, err := NormalizeJSONStreamOptions(input)
	if err != nil {
		t.Fatalf("NormalizeJSONStreamOptions returned error: %v", err)
	}

	assertJSONEqual(t, `{"stream":true,"stream_options":{"include_usage":false},"model":"gpt-4o"}`, string(out))
}

func TestRemoveJSONStreamOptionsRemovesRegardlessOfStream(t *testing.T) {
	input := []byte(`{"stream":true,"stream_options":{"include_usage":true},"model":"gpt-4o"}`)

	out, err := RemoveJSONStreamOptions(input)
	if err != nil {
		t.Fatalf("RemoveJSONStreamOptions returned error: %v", err)
	}

	assertJSONEqual(t, `{"stream":true,"model":"gpt-4o"}`, string(out))
}
