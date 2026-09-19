package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	passkeysvc "github.com/QuantumNous/new-api/service/passkey"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestPasskeySecureVerificationRejectsBypassAndCookieReplay(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	oldDB, oldLogDB := model.DB, model.LOG_DB
	oldRedis := common.RedisEnabled
	common.RedisEnabled = false
	model.DB, model.LOG_DB = db, db
	t.Cleanup(func() { model.DB, model.LOG_DB = oldDB, oldLogDB; common.RedisEnabled = oldRedis; _ = sqlDB.Close() })
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.TwoFA{}, &model.PasskeyCredential{}, &model.VerificationProof{}, &model.Log{}))
	user := model.User{Username: "secure-passkey", Status: common.UserStatusEnabled}
	require.NoError(t, db.Create(&user).Error)
	require.NoError(t, db.Create(&model.PasskeyCredential{UserID: user.Id, CredentialID: "test-credential"}).Error)
	r := gin.New()
	r.Use(sessions.Sessions("test", cookie.NewStore([]byte("verification-test-secret"))))
	r.Use(func(c *gin.Context) { c.Set("id", user.Id); c.Next() })
	r.GET("/seed/:kind", func(c *gin.Context) {
		s := sessions.Default(c)
		s.Set("id", user.Id)
		s.Set(PasskeyReadySessionKey, time.Now().Unix())
		if c.Param("kind") != "legacy" {
			proofUser := user.Id
			if c.Param("kind") == "other" {
				proofUser++
			}
			proof, err := model.IssueVerificationProof(proofUser, passkeyReadyProofPurpose, time.Minute)
			require.NoError(t, err)
			s.Set(passkeyReadyProofSessionKey, proof)
		}
		require.NoError(t, s.Save())
	})
	r.POST("/verify", UniversalVerify)
	r.GET("/challenge", func(c *gin.Context) {
		require.NoError(t, passkeysvc.SaveSessionData(c, passkeysvc.VerifySessionKey, &webauthn.SessionData{Challenge: "test"}))
	})
	r.POST("/consume", func(c *gin.Context) {
		_, err := passkeysvc.PopSessionData(c, passkeysvc.VerifySessionKey)
		c.JSON(200, gin.H{"success": err == nil})
	})
	call := func(method, path string, cookies []*http.Cookie) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(`{"method":"passkey"}`))
		req.Header.Set("Content-Type", "application/json")
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}
	require.Contains(t, call("POST", "/verify", nil).Body.String(), `"success":false`)
	for _, kind := range []string{"legacy", "other"} {
		seed := call("GET", "/seed/"+kind, nil)
		require.Contains(t, call("POST", "/verify", seed.Result().Cookies()).Body.String(), `"success":false`)
	}
	seed := call("GET", "/seed/valid", nil)
	cookies := seed.Result().Cookies()
	require.Contains(t, call("POST", "/verify", cookies).Body.String(), `"success":true`)
	require.Contains(t, call("POST", "/verify", cookies).Body.String(), `"success":false`, "old signed cookie must not reissue authorization")
	challenge := call("GET", "/challenge", nil)
	cookies = challenge.Result().Cookies()
	require.Contains(t, call("POST", "/consume", cookies).Body.String(), `"success":true`)
	require.Contains(t, call("POST", "/consume", cookies).Body.String(), `"success":false`)
}
