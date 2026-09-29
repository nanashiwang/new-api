package helper

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestChatOutputLimitRejectsOverflowForBothAliases(t *testing.T) {
	for _, field := range []string{"max_tokens", "max_completion_tokens"} {
		for _, value := range []string{"-1", "1.5", "18446744073709551615"} {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions",
				strings.NewReader(fmt.Sprintf(`{"model":"kimi-k3","messages":[{"role":"user","content":"hi"}],"%s":%s}`, field, value)))
			c.Request.Header.Set("Content-Type", "application/json")
			_, err := GetAndValidateTextRequest(c, relayconstant.RelayModeChatCompletions)
			require.Error(t, err, field+"="+value)
		}
	}
}
