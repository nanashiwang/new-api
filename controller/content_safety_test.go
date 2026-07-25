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

func setupContentSafetyControllerTest(t *testing.T) (*gorm.DB, *model.User) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.ContentSafetyViolation{}, &model.ContentSafetyReviewCase{}))
	originalDB, originalLogDB := model.DB, model.LOG_DB
	originalRedisEnabled := common.RedisEnabled
	model.DB, model.LOG_DB, common.RedisEnabled = db, db, false
	t.Cleanup(func() {
		model.DB, model.LOG_DB, common.RedisEnabled = originalDB, originalLogDB, originalRedisEnabled
	})
	user := &model.User{Username: "safety-controller", Password: "password123", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, AffCode: "safety-controller-aff"}
	require.NoError(t, db.Create(user).Error)
	return db, user
}

func TestSelfContentSafetyStateAndAcknowledgement(t *testing.T) {
	db, user := setupContentSafetyControllerTest(t)
	now := time.Now().Unix()
	require.NoError(t, db.Create(&model.ContentSafetyViolation{
		UserId: user.Id, EventKey: "controller-event", ErrorCode: "cyber_policy", ErrorType: "invalid_request",
		CreatedAt: now, BurstCount: 1, WindowCount: 1, Action: model.ContentSafetyActionWarning,
		InputHash: "private-input-hash", FineCategory: "credential_theft_phishing", ReasonSource: "local_rule",
		ReasonConfidence: "medium", ReasonSummary: "本地规则推断；未保存原始请求正文。",
	}).Error)

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/content-safety/self", nil)
	c.Set("id", user.Id)
	GetSelfContentSafetyState(c)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"has_unread_warning":true`)
	require.Contains(t, recorder.Body.String(), `"fine_category":"credential_theft_phishing"`)
	require.NotContains(t, recorder.Body.String(), "private-input-hash")
	require.NotContains(t, recorder.Body.String(), "event_key")
	require.NotContains(t, recorder.Body.String(), "channel_id")
	require.NotContains(t, recorder.Body.String(), "token_id")
	require.NotContains(t, recorder.Body.String(), "request_id")

	recorder = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/content-safety/self/acknowledge", nil)
	c.Set("id", user.Id)
	AcknowledgeSelfContentSafetyWarnings(c)
	require.Equal(t, http.StatusOK, recorder.Code)
	var violation model.ContentSafetyViolation
	require.NoError(t, db.Where("user_id = ?", user.Id).First(&violation).Error)
	require.Greater(t, violation.WarningReadAt, int64(0))
}
