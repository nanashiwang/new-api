package service

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
)

func TestBillingSessionShouldTrust_DisabledForPackageToken(t *testing.T) {
	originQuotaPerUnit := common.QuotaPerUnit
	t.Cleanup(func() {
		common.QuotaPerUnit = originQuotaPerUnit
	})
	common.QuotaPerUnit = 1

	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("token_quota", 100)
	common.SetContextKey(c, constant.ContextKeyTokenPackageEnabled, true)

	session := &BillingSession{
		relayInfo: &relaycommon.RelayInfo{
			UserQuota:       100,
			TokenUnlimited:  false,
			ForcePreConsume: false,
		},
		funding: &WalletFunding{userId: 1},
	}

	if session.shouldTrust(c) {
		t.Fatal("package token should not use trust bypass")
	}
}

func TestBillingSessionShouldTrust_EnabledForNormalWalletToken(t *testing.T) {
	originQuotaPerUnit := common.QuotaPerUnit
	t.Cleanup(func() {
		common.QuotaPerUnit = originQuotaPerUnit
	})
	common.QuotaPerUnit = 1

	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("token_quota", 100)
	common.SetContextKey(c, constant.ContextKeyTokenPackageEnabled, false)

	session := &BillingSession{
		relayInfo: &relaycommon.RelayInfo{
			UserQuota:       100,
			TokenUnlimited:  false,
			ForcePreConsume: false,
		},
		funding: &WalletFunding{userId: 1},
	}

	if !session.shouldTrust(c) {
		t.Fatal("normal wallet token should use trust bypass when quota is enough")
	}
}
