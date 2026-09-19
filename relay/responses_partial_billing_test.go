package relay

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	openaichannel "github.com/QuantumNous/new-api/relay/channel/openai"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestResponsesPartialTokenOnlySettlementCannotBeRefunded(t *testing.T) {
	truncateRelayTables(t)
	seedRelayUser(t, 5111, 0)
	seedRelayPackageToken(t, 6111, 5111, "partial-token-test", 100, 100)
	seedRelayChannel(t, 7111)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/responses", nil)
	common.SetContextKey(c, constant.ContextKeyTokenPackageEnabled, true)
	common.SetContextKey(c, constant.ContextKeyTokenBillingMode, model.TokenBillingModeTokenOnly)
	info := &relaycommon.RelayInfo{UserId: 5111, TokenId: 6111, TokenKey: "partial-token-test", OriginModelName: "test-model", UsingGroup: "default", StartTime: time.Now(), IsStream: true,
		ChannelMeta: &relaycommon.ChannelMeta{ChannelId: 7111, UpstreamModelName: "test-model"},
		PriceData:   types.PriceData{ModelRatio: 1, CompletionRatio: 1, GroupRatioInfo: types.GroupRatioInfo{GroupRatio: 1}},
	}
	require.Nil(t, service.PreConsumeBilling(c, 20, info))
	body := "data: {\"type\":\"response.output_text.delta\",\"delta\":\"hello\"}\n" +
		`data: {"type":"response.incomplete","response":{"status":"incomplete","usage":{"input_tokens":5,"output_tokens":8,"total_tokens":13}}}` + "\n"
	usage, apiErr := openaichannel.OaiResponsesStreamHandler(c, info, &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: io.NopCloser(strings.NewReader(body))})
	require.NotNil(t, apiErr)
	require.NotNil(t, usage)
	require.True(t, usage.InterruptedOutput)
	postConsumeQuota(c, info, usage, "流式响应中断，按已交付用量结算")
	require.False(t, info.Billing.NeedsRefund())
	info.Billing.Refund(c) // mirrors the controller error defer
	var token model.Token
	require.NoError(t, model.DB.First(&token, 6111).Error)
	require.Equal(t, 87, token.RemainQuota)
	require.Equal(t, 13, getLastRelayLog(t).Quota)
	require.False(t, info.StreamStatus.IsSuccessful())
}
