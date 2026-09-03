package controller

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestForumSSOConfigRequiresHTTPSAndSecrets(t *testing.T) {
	t.Setenv("PULSE_FORUM_SSO_SECRET", "forum-secret")
	t.Setenv("PULSE_FORUM_SSO_CALLBACK_URL", "https://forum.example.test/api/user-center/login/callback")
	callback, secret, err := forumSSOConfig()
	require.NoError(t, err)
	require.Equal(t, "https://forum.example.test/api/user-center/login/callback", callback)
	require.Equal(t, "forum-secret", secret)

	t.Setenv("PULSE_FORUM_SSO_CALLBACK_URL", "http://forum.example.test/callback")
	_, _, err = forumSSOConfig()
	require.Error(t, err)
	t.Setenv("PULSE_FORUM_SSO_CALLBACK_URL", "https://forum.example.test/callback#fragment")
	_, _, err = forumSSOConfig()
	require.Error(t, err)
	t.Setenv("PULSE_FORUM_SSO_SECRET", "")
	_, _, err = forumSSOConfig()
	require.Error(t, err)
}

func TestForumSSOTicketSignUsesStableHMACPayload(t *testing.T) {
	ticket := forumSSOTicket{
		UserID: "7", Username: "alice", DisplayName: "Alice", Email: "alice@example.test",
		Avatar: "", Timestamp: 1700000000, Nonce: "nonce-1",
	}
	payload := "7\nalice\nAlice\nalice@example.test\n\n1700000000\nnonce-1"
	mac := hmac.New(sha256.New, []byte("forum-secret"))
	_, err := mac.Write([]byte(payload))
	require.NoError(t, err)
	signature, err := ticket.sign("forum-secret")
	require.NoError(t, err)
	require.Equal(t, hex.EncodeToString(mac.Sum(nil)), signature)
}

func TestForumSSOTicketRejectsAmbiguousOrNonCanonicalFields(t *testing.T) {
	valid := forumSSOTicket{UserID: "7", Username: "alice", Timestamp: 1700000000, Nonce: "nonce-1"}
	for _, test := range []struct {
		name   string
		mutate func(*forumSSOTicket)
	}{
		{name: "zero user id", mutate: func(ticket *forumSSOTicket) { ticket.UserID = "0" }},
		{name: "negative user id", mutate: func(ticket *forumSSOTicket) { ticket.UserID = "-1" }},
		{name: "non-canonical user id", mutate: func(ticket *forumSSOTicket) { ticket.UserID = "007" }},
		{name: "username CRLF", mutate: func(ticket *forumSSOTicket) { ticket.Username = "alice\r\nadmin" }},
		{name: "display name LF", mutate: func(ticket *forumSSOTicket) { ticket.DisplayName = "Alice\nAdmin" }},
		{name: "email LF", mutate: func(ticket *forumSSOTicket) { ticket.Email = "alice@example.test\nadmin@example.test" }},
		{name: "avatar LF", mutate: func(ticket *forumSSOTicket) { ticket.Avatar = "https://example.test/a\nb" }},
		{name: "nonce LF", mutate: func(ticket *forumSSOTicket) { ticket.Nonce = "nonce\nshift" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			ticket := valid
			test.mutate(&ticket)
			_, err := ticket.sign("forum-secret")
			require.Error(t, err)
		})
	}
}

func TestForumSSOStartRedirectsUnauthenticatedUsersToLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("PULSE_FORUM_SSO_SECRET", "forum-secret")
	t.Setenv("PULSE_FORUM_SSO_CALLBACK_URL", "https://forum.example.test/api/user-center/login/callback")

	router := gin.New()
	router.Use(sessions.Sessions("test", cookie.NewStore([]byte("cookie-secret"))))
	router.GET(forumSSOStartPath, ForumSSOStart)

	request := httptest.NewRequest(http.MethodGet, forumSSOStartPath, nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusFound, recorder.Code)
	require.Equal(t, "/login?next=%2Fapi%2Fforum%2Fsso%2Fstart", recorder.Header().Get("Location"))
}

func TestForumSSOStartFailsClosedWhenUnconfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("PULSE_FORUM_SSO_SECRET", "")
	t.Setenv("PULSE_FORUM_SSO_CALLBACK_URL", "")

	router := gin.New()
	router.GET(forumSSOStartPath, ForumSSOStart)
	request := httptest.NewRequest(http.MethodGet, forumSSOStartPath, nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
}

func TestForumSSOStartUsesSessionIdentityAndFixedCallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("PULSE_FORUM_SSO_SECRET", "forum-secret")
	t.Setenv("PULSE_FORUM_SSO_CALLBACK_URL", "https://forum.example.test/api/user-center/login/callback?fixed=1")

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}))
	oldDB := model.DB
	model.DB = db
	t.Cleanup(func() { model.DB = oldDB })
	user := model.User{Username: "alice", DisplayName: "Alice", Email: "alice@example.test", Status: common.UserStatusEnabled, AffCode: "pulse-sso-alice"}
	require.NoError(t, db.Create(&user).Error)

	router := gin.New()
	router.Use(sessions.Sessions("test", cookie.NewStore([]byte("cookie-secret"))))
	router.GET("/seed", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Set("id", user.Id)
		session.Set("status", user.Status)
		require.NoError(t, session.Save())
		c.Status(http.StatusNoContent)
	})
	router.GET(forumSSOStartPath, ForumSSOStart)

	seedRecorder := httptest.NewRecorder()
	router.ServeHTTP(seedRecorder, httptest.NewRequest(http.MethodGet, "/seed", nil))
	require.Equal(t, http.StatusNoContent, seedRecorder.Code)
	cookies := seedRecorder.Result().Cookies()
	require.NotEmpty(t, cookies)

	request := httptest.NewRequest(http.MethodGet, forumSSOStartPath+"?user_id=999&callback=https://attacker.test", nil)
	for _, item := range cookies {
		request.AddCookie(item)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusFound, recorder.Code)

	target, err := url.Parse(recorder.Header().Get("Location"))
	require.NoError(t, err)
	require.Equal(t, "https", target.Scheme)
	require.Equal(t, "forum.example.test", target.Host)
	require.Equal(t, "/api/user-center/login/callback", target.Path)
	require.Equal(t, "1", target.Query().Get("fixed"))
	require.Equal(t, strconv.Itoa(user.Id), target.Query().Get("user_id"))
	require.NotEqual(t, "999", target.Query().Get("user_id"))
	require.Equal(t, user.Username, target.Query().Get("username"))
	require.NotEmpty(t, target.Query().Get("nonce"))
	require.NotEmpty(t, target.Query().Get("signature"))
}
