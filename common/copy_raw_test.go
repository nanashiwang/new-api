package common_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/jinzhu/copier"
	"github.com/stretchr/testify/require"
)

func TestDeepCopyRawMessageIsolation(t *testing.T) {
	src := &dto.OpenAIResponsesRequest{
		Model: "test", Input: json.RawMessage(`[{"role":"user","content":"hi"}]`),
		Tools: json.RawMessage(`[{"type":"web_search"}]`),
	}
	first, err := common.DeepCopy(src)
	require.NoError(t, err)
	second, err := common.DeepCopy(src)
	require.NoError(t, err)
	require.Equal(t, src, first)
	first.Input[0], first.Tools[0] = '{', '{'
	require.Equal(t, byte('['), src.Input[0])
	require.Equal(t, src, second)
	for _, raw := range []json.RawMessage{nil, {}, []byte(`null`)} {
		original := &struct{ Raw json.RawMessage }{Raw: raw}
		cloned, err := common.DeepCopy(original)
		require.NoError(t, err)
		require.Equal(t, original, cloned)
	}
}

func BenchmarkRawMessageDeepCopy(b *testing.B) {
	src := &dto.OpenAIResponsesRequest{Input: bytes.Repeat([]byte(" "), 256*1024)}
	b.Run("legacy", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			var dst dto.OpenAIResponsesRequest
			if err := copier.CopyWithOption(&dst, src, copier.Option{DeepCopy: true, IgnoreEmpty: true}); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("bulk", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, err := common.DeepCopy(src); err != nil {
				b.Fatal(err)
			}
		}
	})
}
