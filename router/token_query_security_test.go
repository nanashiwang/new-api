package router

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupTokenQuerySecurityTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	oldOptionMap := common.OptionMap
	oldGlobalAPIRateLimitEnable := common.GlobalApiRateLimitEnable
	oldCriticalRateLimitEnable := common.CriticalRateLimitEnable
	oldPublicTokenUsageRateLimitEnable := common.PublicTokenUsageRateLimitEnable
	oldDisplayTokenStatEnabled := common.DisplayTokenStatEnabled
	oldUsingSQLite := common.UsingSQLite
	oldUserUsableGroups := setting.UserUsableGroups2JSONString()
	oldGroupRatio := ratio_setting.GroupRatio2JSONString()

	model.DB = db
	model.LOG_DB = db
	common.RedisEnabled = false
	common.UsingSQLite = true
	common.GlobalApiRateLimitEnable = false
	common.CriticalRateLimitEnable = false
	common.PublicTokenUsageRateLimitEnable = false
	common.DisplayTokenStatEnabled = true
	common.OptionMap = map[string]string{
		"HeaderNavModules": `{"usage":true}`,
	}

	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"default":"默认分组"}`))
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1}`))
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Token{}, &model.Log{}))

	t.Cleanup(func() {
		common.OptionMap = oldOptionMap
		common.GlobalApiRateLimitEnable = oldGlobalAPIRateLimitEnable
		common.CriticalRateLimitEnable = oldCriticalRateLimitEnable
		common.PublicTokenUsageRateLimitEnable = oldPublicTokenUsageRateLimitEnable
		common.DisplayTokenStatEnabled = oldDisplayTokenStatEnabled
		common.UsingSQLite = oldUsingSQLite
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(oldUserUsableGroups))
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(oldGroupRatio))
	})

	return db
}

func newTokenQuerySecurityRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	SetApiRouter(r)
	return r
}

func seedTokenQuerySecurityUser(t *testing.T, db *gorm.DB, userID int) {
	t.Helper()

	require.NoError(t, db.Create(&model.User{
		Id:       userID,
		Username: "query-user",
		Group:    "default",
		Status:   common.UserStatusEnabled,
		Quota:    500000,
		AffCode:  "token-query-aff",
	}).Error)
}

func buildQuerySecurityToken(userID int, key string) model.Token {
	now := common.GetTimestamp()
	return model.Token{
		UserId:       userID,
		Key:          key,
		Name:         "query-token",
		Status:       common.TokenStatusEnabled,
		CreatedTime:  now,
		AccessedTime: now,
		ExpiredTime:  -1,
		RemainQuota:  300000,
		UsedQuota:    120000,
	}
}

func TestUsageTokenRoute_AllowsSuffixNormalizedKey(t *testing.T) {
	db := setupTokenQuerySecurityTestDB(t)
	seedTokenQuerySecurityUser(t, db, 1)

	token := buildQuerySecurityToken(1, "usagequerytokenkeyabcdefghijklmnopqrstuvwxyz1234")
	require.NoError(t, db.Create(&token).Error)

	router := newTokenQuerySecurityRouter()
	req := httptest.NewRequest(http.MethodGet, "/api/usage/token/", nil)
	req.Header.Set("Authorization", "Bearer sk-"+token.Key+"-channelA")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)

	var resp struct {
		Code bool `json:"code"`
		Data struct {
			Object         string `json:"object"`
			Name           string `json:"name"`
			TotalGranted   int    `json:"total_granted"`
			TotalUsed      int    `json:"total_used"`
			TotalAvailable int    `json:"total_available"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.True(t, resp.Code, recorder.Body.String())
	require.Equal(t, "token_usage", resp.Data.Object)
	require.Equal(t, token.Name, resp.Data.Name)
	require.Equal(t, token.RemainQuota+token.UsedQuota, resp.Data.TotalGranted)
	require.Equal(t, token.UsedQuota, resp.Data.TotalUsed)
	require.Equal(t, token.RemainQuota, resp.Data.TotalAvailable)
}

func TestTokenQueryRoutes_RejectUnavailableTokens(t *testing.T) {
	type routeCase struct {
		name        string
		path        string
		seedLogData bool
	}

	type tokenCase struct {
		name  string
		token func(userID int) model.Token
	}

	routeCases := []routeCase{
		{name: "usage", path: "/api/usage/token/"},
		{name: "log", path: "/api/log/token", seedLogData: true},
	}
	tokenCases := []tokenCase{
		{
			name: "disabled",
			token: func(userID int) model.Token {
				token := buildQuerySecurityToken(userID, "disabledtokenquerykeyabcdefghijklmnopqrstuvwxyz12")
				token.Status = common.TokenStatusDisabled
				return token
			},
		},
		{
			name: "expired",
			token: func(userID int) model.Token {
				token := buildQuerySecurityToken(userID, "expiredtokenquerykeyabcdefghijklmnopqrstuvwxyz123")
				token.ExpiredTime = common.GetTimestamp() - 60
				return token
			},
		},
		{
			name: "exhausted",
			token: func(userID int) model.Token {
				token := buildQuerySecurityToken(userID, "exhaustedtokenquerykeyabcdefghijklmnopqrstuvwxyz1")
				token.RemainQuota = 0
				return token
			},
		},
	}

	for _, routeCase := range routeCases {
		for _, tokenCase := range tokenCases {
			t.Run(routeCase.name+"_"+tokenCase.name, func(t *testing.T) {
				db := setupTokenQuerySecurityTestDB(t)
				seedTokenQuerySecurityUser(t, db, 1)

				token := tokenCase.token(1)
				require.NoError(t, db.Create(&token).Error)
				if routeCase.seedLogData {
					require.NoError(t, db.Create(&model.Log{
						UserId:    1,
						TokenId:   token.Id,
						Type:      model.LogTypeConsume,
						ModelName: "gpt-5.2",
						Content:   "test log",
						TokenName: token.Name,
						CreatedAt: common.GetTimestamp(),
					}).Error)
				}

				router := newTokenQuerySecurityRouter()
				req := httptest.NewRequest(http.MethodGet, routeCase.path, nil)
				req.Header.Set("Authorization", "Bearer sk-"+token.Key)
				recorder := httptest.NewRecorder()

				router.ServeHTTP(recorder, req)

				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			})
		}
	}
}

func TestPublicTokenUsageAndLogs_RejectUnavailableTokens(t *testing.T) {
	type endpointCase struct {
		name    string
		path    string
		payload func(key string) []byte
	}

	type tokenCase struct {
		name  string
		token func(userID int) model.Token
	}

	endpoints := []endpointCase{
		{
			name: "usage",
			path: "/api/usage/public_token",
			payload: func(key string) []byte {
				return []byte(`{"key":"sk-` + key + `"}`)
			},
		},
		{
			name: "logs",
			path: "/api/usage/public_token/logs",
			payload: func(key string) []byte {
				return []byte(`{"key":"sk-` + key + `","page":1,"page_size":10}`)
			},
		},
	}
	tokenCases := []tokenCase{
		{
			name: "disabled",
			token: func(userID int) model.Token {
				token := buildQuerySecurityToken(userID, "disabledpublictokenquerykeyabcdefghijklmnopqrstuvwxyz")
				token.Status = common.TokenStatusDisabled
				return token
			},
		},
		{
			name: "expired",
			token: func(userID int) model.Token {
				token := buildQuerySecurityToken(userID, "expiredpublictokenquerykeyabcdefghijklmnopqrstuvwxyz1")
				token.ExpiredTime = common.GetTimestamp() - 60
				return token
			},
		},
		{
			name: "exhausted",
			token: func(userID int) model.Token {
				token := buildQuerySecurityToken(userID, "exhaustedpublictokenquerykeyabcdefghijklmnopqrstuvwxyz")
				token.RemainQuota = 0
				return token
			},
		},
	}

	for _, endpoint := range endpoints {
		for _, tokenCase := range tokenCases {
			t.Run(endpoint.name+"_"+tokenCase.name, func(t *testing.T) {
				db := setupTokenQuerySecurityTestDB(t)
				seedTokenQuerySecurityUser(t, db, 1)

				token := tokenCase.token(1)
				require.NoError(t, db.Create(&token).Error)
				require.NoError(t, db.Create(&model.Log{
					UserId:    1,
					TokenId:   token.Id,
					Type:      model.LogTypeConsume,
					ModelName: "gpt-5.2",
					Content:   "public token log",
					TokenName: token.Name,
					CreatedAt: common.GetTimestamp(),
				}).Error)

				router := newTokenQuerySecurityRouter()
				req := httptest.NewRequest(http.MethodPost, endpoint.path, bytes.NewReader(endpoint.payload(token.Key)))
				req.Header.Set("Content-Type", "application/json")
				recorder := httptest.NewRecorder()

				router.ServeHTTP(recorder, req)

				require.Equal(t, http.StatusOK, recorder.Code)

				var resp struct {
					Success bool   `json:"success"`
					Message string `json:"message"`
				}
				require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
				require.False(t, resp.Success)
				require.Equal(t, "无效的 API Key", resp.Message)
			})
		}
	}
}

func TestPublicTokenBatchUsage_MarksUnavailableTokensAsInvalid(t *testing.T) {
	db := setupTokenQuerySecurityTestDB(t)
	seedTokenQuerySecurityUser(t, db, 1)

	enabledToken := buildQuerySecurityToken(1, "enabledpublicbatchtokenquerykeyabcdefghijklmnopqrstuvwxyz")
	disabledToken := buildQuerySecurityToken(1, "disabledpublicbatchtokenquerykeyabcdefghijklmnopqrstuvwxy")
	disabledToken.Status = common.TokenStatusDisabled

	require.NoError(t, db.Create(&enabledToken).Error)
	require.NoError(t, db.Create(&disabledToken).Error)

	router := newTokenQuerySecurityRouter()
	body := []byte(`{"keys":["sk-` + enabledToken.Key + `","sk-` + disabledToken.Key + `"]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/usage/public_token/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Tokens []struct {
				TokenID int `json:"token_id"`
			} `json:"tokens"`
			InvalidKeys []string `json:"invalid_keys"`
			Summary     struct {
				ValidKeyCount   int `json:"valid_key_count"`
				InvalidKeyCount int `json:"invalid_key_count"`
			} `json:"summary"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.True(t, resp.Success)
	require.Len(t, resp.Data.Tokens, 1)
	require.Equal(t, enabledToken.Id, resp.Data.Tokens[0].TokenID)
	require.Equal(t, []string{disabledToken.Key}, resp.Data.InvalidKeys)
	require.Equal(t, 1, resp.Data.Summary.ValidKeyCount)
	require.Equal(t, 1, resp.Data.Summary.InvalidKeyCount)
}
