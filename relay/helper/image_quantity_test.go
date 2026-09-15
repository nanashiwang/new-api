package helper

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestImageQuantityIngress(t *testing.T) {
	for _, tc := range []struct {
		body         string
		ali, invalid bool
		count        int
	}{
		{`{"model":"image"}`, false, false, 1},
		{`{"model":"image","n":0}`, false, false, 1},
		{`{"model":"image","n":128}`, false, false, 128},
		{`{"model":"image","n":129}`, false, true, 0},
		{`{"model":"image","n":-1}`, false, true, 0},
		{`{"model":"image","n":18446744073709551615}`, false, true, 0},
		{`{"model":"image","n":2,"parameters":{"n":3}}`, true, false, 3},
		{`{"model":"image","n":2,"parameters":{}}`, true, false, 2},
		{`{"model":"image","parameters":{"n":0}}`, true, true, 0},
		{`{"model":"image","parameters":{"n":-1}}`, true, true, 0},
		{`{"model":"image","parameters":{"n":1.5}}`, true, true, 0},
		{`{"model":"image","parameters":{"n":129}}`, true, true, 0},
		{`{"model":"image","n":2,"parameters":{"n":0}}`, false, false, 2},
	} {
		t.Run(tc.body+fmtBool(tc.ali), func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("POST", "/v1/images/generations", strings.NewReader(tc.body))
			c.Request.Header.Set("Content-Type", "application/json")
			if tc.ali {
				common.SetContextKey(c, constant.ContextKeyChannelType, constant.ChannelTypeAli)
			}
			req, err := GetAndValidOpenAIImageRequest(c, relayconstant.RelayModeImagesGenerations)
			if tc.invalid {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			count, _, err := req.ImageBillingQuantity(tc.ali)
			require.NoError(t, err)
			require.Equal(t, tc.count, count)
		})
	}
}

func fmtBool(v bool) string {
	if v {
		return "_ali"
	}
	return "_openai"
}

func TestMultipartImageQuantity(t *testing.T) {
	for _, n := range []string{"-1", "129", "1.5", "invalid", "18446744073709551615", "2"} {
		t.Run(n, func(t *testing.T) {
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			require.NoError(t, writer.WriteField("model", "image"))
			require.NoError(t, writer.WriteField("n", n))
			require.NoError(t, writer.Close())
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("POST", "/v1/images/edits", &body)
			c.Request.Header.Set("Content-Type", writer.FormDataContentType())
			_, err := GetAndValidOpenAIImageRequest(c, relayconstant.RelayModeImagesEdits)
			if n == "2" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
