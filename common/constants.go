package common

import (
	"crypto/tls"
	//"os"
	//"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
)

var StartTime = time.Now().Unix() // unit: second
var Version = "v0.0.0"            // this hard coding will be replaced automatically when building, no need to manually change
var SystemName = "New API"
var Footer = ""
var Logo = ""
var TopUpLink = ""

// var ChatLink = ""
// var ChatLink2 = ""
var QuotaPerUnit = 500 * 1000.0 // $0.002 / 1K tokens
// 保留旧变量以兼容历史逻辑，实际展示由 general_setting.quota_display_type 控制
var DisplayInCurrencyEnabled = true
var DisplayTokenStatEnabled = true
var DrawingEnabled = true
var TaskEnabled = true
var DataExportEnabled = true
var DataExportInterval = 5         // unit: minute
var DataExportDefaultTime = "hour" // unit: minute
var DefaultCollapseSidebar = false // default value of collapse sidebar

// Any options with "Secret", "Token" in its key won't be return by GetOptions

var SessionSecret = uuid.New().String()
var CryptoSecret = uuid.New().String()

var OptionMap map[string]string
var OptionMapRWMutex sync.RWMutex

var ItemsPerPage = 10
var MaxRecentItems = 1000

var PasswordLoginEnabled = true
var PasswordRegisterEnabled = true
var EmailVerificationEnabled = false
var GitHubOAuthEnabled = false
var LinuxDOOAuthEnabled = false
var WeChatAuthEnabled = false
var TelegramOAuthEnabled = false
var TurnstileCheckEnabled = false
var RegisterEnabled = true

var EmailDomainRestrictionEnabled = false // 是否启用邮箱域名限制
var EmailAliasRestrictionEnabled = false  // 是否启用邮箱别名限制
var EmailDomainWhitelist = []string{
	"gmail.com",
	"163.com",
	"126.com",
	"qq.com",
	"outlook.com",
	"hotmail.com",
	"icloud.com",
	"yahoo.com",
	"foxmail.com",
}
var EmailLoginAuthServerList = []string{
	"smtp.sendcloud.net",
	"smtp.azurecomm.net",
}

var DebugEnabled bool
var MemoryCacheEnabled bool

var LogConsumeEnabled = true

var TLSInsecureSkipVerify bool
var InsecureTLSConfig = &tls.Config{InsecureSkipVerify: true}

var SMTPServer = ""
var SMTPPort = 587
var SMTPSSLEnabled = false
var SMTPAccount = ""
var SMTPFrom = ""
var SMTPToken = ""

var GitHubClientId = ""
var GitHubClientSecret = ""
var LinuxDOClientId = ""
var LinuxDOClientSecret = ""
var LinuxDOMinimumTrustLevel = 0

var WeChatServerAddress = ""
var WeChatServerToken = ""
var WeChatAccountQRCodeImageURL = ""

var TurnstileSiteKey = ""
var TurnstileSecretKey = ""

var TelegramBotToken = ""
var TelegramBotName = ""

var QuotaForNewUser = 0
var QuotaForInviter = 0
var QuotaForInvitee = 0

// InviterCommissionEnabled 是否启用“邀请充值返佣”功能。
// 默认 false：不开启时不会创建返佣台账，也不会触发日批结算发放。
var InviterCommissionEnabled = false

// InviterRechargeCommissionRate 邀请返佣比例，按“充值额度”计算。
// 取值建议 [0,1]：0.1 表示返佣 10%，0 表示不返佣。
var InviterRechargeCommissionRate = 0.05

// InviterCommissionDailyCap 邀请人单日返佣上限。
// 0 表示不限制；大于 0 时超出上限部分会被跳过或截断（见返佣风控逻辑）。
var InviterCommissionDailyCap = 0
var ChannelDisableThreshold = 5.0
var AutomaticDisableChannelEnabled = false
var AutomaticEnableChannelEnabled = false
var QuotaRemindThreshold = 1000
var PreConsumedQuota = 500

// QuotaStabilityEnabled 是否开启额度防抖。
var QuotaStabilityEnabled = true

// QuotaInsufficientDBRecheckEnabled 报额度不足前，是否再查一次数据库。
var QuotaInsufficientDBRecheckEnabled = true

// MultiKeyCooldownSeconds Key 冷却时长（秒）。
var MultiKeyCooldownSeconds = 60

// MultiKeyStickySeconds Key 粘性时长（秒）。
var MultiKeyStickySeconds = 60

// MultiKeySameChannelFailoverOnce 额度错误时，是否同渠道立即重试一次。
var MultiKeySameChannelFailoverOnce = true

// SubscriptionRenewRespectPurchaseLimit 是否让“续费”也受每用户购买上限约束。
// false: 仅新购受限；true: 新购与续费都受限。
var SubscriptionRenewRespectPurchaseLimit = false

var RetryTimes = 0

//var RootUserEmail = ""

var IsMasterNode bool

var requestInterval int
var RequestInterval time.Duration

var SyncFrequency int // unit is second

var BatchUpdateEnabled = false
var BatchUpdateInterval int

var RelayTimeout int // unit is second

var RelayMaxIdleConns int
var RelayMaxIdleConnsPerHost int

var GeminiSafetySetting string

// https://docs.cohere.com/docs/safety-modes Type; NONE/CONTEXTUAL/STRICT
var CohereSafetySetting string

const (
	RequestIdKey = "X-Oneapi-Request-Id"
)

const (
	RoleGuestUser  = 0
	RoleCommonUser = 1
	RoleAdminUser  = 10
	RoleRootUser   = 100
)

func IsValidateRole(role int) bool {
	return role == RoleGuestUser || role == RoleCommonUser || role == RoleAdminUser || role == RoleRootUser
}

var (
	FileUploadPermission    = RoleGuestUser
	FileDownloadPermission  = RoleGuestUser
	ImageUploadPermission   = RoleGuestUser
	ImageDownloadPermission = RoleGuestUser
)

// All duration's unit is seconds
// Shouldn't larger then RateLimitKeyExpirationDuration
var (
	GlobalApiRateLimitEnable   bool
	GlobalApiRateLimitNum      int
	GlobalApiRateLimitDuration int64

	GlobalWebRateLimitEnable   bool
	GlobalWebRateLimitNum      int
	GlobalWebRateLimitDuration int64

	PublicTokenUsageRateLimitEnable   bool
	PublicTokenUsageRateLimitNum      int
	PublicTokenUsageRateLimitDuration int64

	CriticalRateLimitEnable   bool
	CriticalRateLimitNum            = 20
	CriticalRateLimitDuration int64 = 20 * 60

	UploadRateLimitNum            = 10
	UploadRateLimitDuration int64 = 60

	DownloadRateLimitNum            = 10
	DownloadRateLimitDuration int64 = 60

	// Per-user search rate limit (applies after authentication, keyed by user ID)
	SearchRateLimitNum            = 10
	SearchRateLimitDuration int64 = 60
)

var RateLimitKeyExpirationDuration = 20 * time.Minute

const (
	UserStatusEnabled  = 1 // don't use 0, 0 is the default value!
	UserStatusDisabled = 2 // also don't use 0
)

const (
	TokenStatusEnabled   = 1 // don't use 0, 0 is the default value!
	TokenStatusDisabled  = 2 // also don't use 0
	TokenStatusExpired   = 3
	TokenStatusExhausted = 4
)

const (
	RedemptionCodeStatusEnabled  = 1 // don't use 0, 0 is the default value!
	RedemptionCodeStatusDisabled = 2 // also don't use 0
	RedemptionCodeStatusUsed     = 3 // also don't use 0
)

const (
	ChannelStatusUnknown          = 0
	ChannelStatusEnabled          = 1 // don't use 0, 0 is the default value!
	ChannelStatusManuallyDisabled = 2 // also don't use 0
	ChannelStatusAutoDisabled     = 3
)

const (
	TopUpStatusPending = "pending"
	TopUpStatusSuccess = "success"
	TopUpStatusExpired = "expired"
)
