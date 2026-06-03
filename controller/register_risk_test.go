package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupRegisterRiskControllerTestDB(t *testing.T) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	originDB := model.DB
	originUsingSQLite := common.UsingSQLite
	originUsingMySQL := common.UsingMySQL
	originUsingPostgreSQL := common.UsingPostgreSQL

	model.DB = db
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false

	t.Cleanup(func() {
		model.DB = originDB
		common.UsingSQLite = originUsingSQLite
		common.UsingMySQL = originUsingMySQL
		common.UsingPostgreSQL = originUsingPostgreSQL
	})

	require.NoError(t, db.AutoMigrate(&model.User{}))
}

func newRegisterRiskTestContext(t *testing.T, ip string, userAgent string) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodPost, "/api/user/register", nil)
	req.RemoteAddr = ip + ":12345"
	req.Header.Set("User-Agent", userAgent)
	ctx.Request = req
	return ctx
}

func createRegisterRiskUsers(t *testing.T, users ...model.User) {
	t.Helper()

	now := time.Now().Unix()
	for i := range users {
		if users[i].Password == "" {
			users[i].Password = "test-password"
		}
		if users[i].Role == 0 {
			users[i].Role = common.RoleCommonUser
		}
		if users[i].Status == 0 {
			users[i].Status = common.UserStatusEnabled
		}
		if users[i].CreatedAt == 0 {
			users[i].CreatedAt = now
		}
	}
	require.NoError(t, model.DB.Create(&users).Error)
}

func TestEnforceUserRegisterRiskBlocksThirdSameIP(t *testing.T) {
	setupRegisterRiskControllerTestDB(t)

	createRegisterRiskUsers(t,
		model.User{Username: "same-ip-1", Email: "a@example.com", AffCode: "same-ip-1", RegisterIP: "198.51.100.10"},
		model.User{Username: "same-ip-2", Email: "b@example.com", AffCode: "same-ip-2", RegisterIP: "198.51.100.10"},
	)

	ctx := newRegisterRiskTestContext(t, "198.51.100.10", "Mozilla/5.0")
	err := enforceUserRegisterRisk(ctx, &model.User{Email: "c@example.com"})

	require.ErrorIs(t, err, errRegisterRiskBlocked)
}

func TestEnforceUserRegisterRiskBlocksSameIPv4C24(t *testing.T) {
	setupRegisterRiskControllerTestDB(t)

	createRegisterRiskUsers(t,
		model.User{Username: "c24-1", Email: "a@example.com", AffCode: "c24-1", RegisterIP: "203.0.113.1"},
		model.User{Username: "c24-2", Email: "b@example.com", AffCode: "c24-2", RegisterIP: "203.0.113.2"},
		model.User{Username: "c24-3", Email: "c@example.com", AffCode: "c24-3", RegisterIP: "203.0.113.3"},
		model.User{Username: "c24-4", Email: "d@example.com", AffCode: "c24-4", RegisterIP: "203.0.113.4"},
		model.User{Username: "c24-5", Email: "e@example.com", AffCode: "c24-5", RegisterIP: "203.0.113.5"},
	)

	ctx := newRegisterRiskTestContext(t, "203.0.113.88", "Mozilla/5.0")
	err := enforceUserRegisterRisk(ctx, &model.User{Email: "f@example.com"})

	require.ErrorIs(t, err, errRegisterRiskBlocked)
}

func TestEnforceUserRegisterRiskBlocksSameUAAndEmailDomain(t *testing.T) {
	setupRegisterRiskControllerTestDB(t)

	userAgent := "Mozilla/5.0 Chrome/148.0.0.0"
	createRegisterRiskUsers(t,
		model.User{Username: "ua-domain-1", Email: "a@outlook.com", AffCode: "ua-domain-1", RegisterIP: "198.51.100.1", RegisterUserAgent: userAgent},
		model.User{Username: "ua-domain-2", Email: "b@outlook.com", AffCode: "ua-domain-2", RegisterIP: "198.51.100.2", RegisterUserAgent: userAgent},
		model.User{Username: "ua-domain-3", Email: "c@outlook.com", AffCode: "ua-domain-3", RegisterIP: "198.51.100.3", RegisterUserAgent: userAgent},
	)

	ctx := newRegisterRiskTestContext(t, "198.51.100.88", userAgent)
	err := enforceUserRegisterRisk(ctx, &model.User{Email: "d@outlook.com"})

	require.ErrorIs(t, err, errRegisterRiskBlocked)
}

func TestEnforceUserRegisterRiskAllowsLowRiskRegister(t *testing.T) {
	setupRegisterRiskControllerTestDB(t)

	createRegisterRiskUsers(t,
		model.User{Username: "low-risk-1", Email: "a@example.com", AffCode: "low-risk-1", RegisterIP: "198.51.100.1", RegisterUserAgent: "Mozilla/5.0"},
	)

	ctx := newRegisterRiskTestContext(t, "203.0.113.88", "Mozilla/5.0 Other")
	err := enforceUserRegisterRisk(ctx, &model.User{Email: "b@example.com"})

	require.NoError(t, err)
}
