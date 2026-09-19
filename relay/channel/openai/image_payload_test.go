package openai

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestImagePayloadCountsAndJSONStreamFallback(t *testing.T) {
	for _, test := range []struct {
		data  string
		count int
	}{
		{`{"url":"one"}`, 1},
		{`[{"url":"one","b64_json":"same"},{"revised_prompt":"text only"}]`, 1},
		{`[{"url":"one"},{"b64_json":"another"}]`, 2},
		{`[{"revised_prompt":"text only"}]`, 0}, {`[]`, 0}, {`null`, 0},
	} {
		body := `{"created":123,"data":` + test.data + `}`
		for _, stream := range []bool{false, true} {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("POST", "/v1/images/generations", nil)
			info := &relaycommon.RelayInfo{RelayFormat: types.RelayFormatOpenAIImage}
			resp := &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}
			handler := OpenaiHandlerWithUsage
			if stream {
				handler = OpenaiImageJSONAsStreamHandler
			}
			_, apiErr := handler(c, info, resp)
			if test.count == 0 {
				require.NotNil(t, apiErr)
				require.Empty(t, w.Body.String(), "do not send a success before validating image payloads")
				continue
			}
			require.Nil(t, apiErr)
			require.NotNil(t, info.ImageResponseCount)
			require.Equal(t, test.count, *info.ImageResponseCount)
			if stream {
				require.Equal(t, test.count, strings.Count(w.Body.String(), "event: image_generation.completed"))
			} else {
				require.Equal(t, body, w.Body.String())
			}
		}
	}
}
