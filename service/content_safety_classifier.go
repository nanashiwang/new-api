package service

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

const contentSafetyClassifierVersion = "rules-2026-07-25-v1"
const maxContentSafetyClassifierBytes = 2 << 20

type ContentSafetyClassification struct {
	OfficialMessage   string
	FineCategory      string
	ReasonSource      string
	ReasonConfidence  string
	ReasonSummary     string
	ClassifierVersion string
}

type contentSafetyRule struct {
	category string
	label    string
	patterns []*regexp.Regexp
}

var contentSafetyRules = []contentSafetyRule{
	{category: "credential_theft_phishing", label: "钓鱼或凭证窃取", patterns: compileSafetyPatterns(`(?i)phish(?:ing)?|credential harvesting|steal (?:password|cookie|token)|fake login|仿冒登录|钓鱼|窃取(?:密码|凭证|cookie|令牌)`)},
	{category: "malware", label: "恶意软件", patterns: compileSafetyPatterns(`(?i)malware|trojan|keylogger|remote access trojan|木马|键盘记录器|恶意软件`)},
	{category: "ransomware", label: "勒索软件", patterns: compileSafetyPatterns(`(?i)ransomware|encrypt (?:all )?files.*ransom|勒索软件|加密文件.*赎金`)},
	{category: "unauthorized_access", label: "未授权访问", patterns: compileSafetyPatterns(`(?i)unauthori[sz]ed access|break into (?:an? )?(?:account|server|system)|绕过认证|未授权访问|入侵(?:账号|服务器|系统)`)},
	{category: "exploit_development", label: "漏洞利用开发", patterns: compileSafetyPatterns(`(?i)exploit (?:code|chain|payload)|weaponize (?:a )?(?:cve|vulnerability)|漏洞利用(?:代码|链|载荷)|武器化.*漏洞`)},
	{category: "privilege_escalation", label: "权限提升", patterns: compileSafetyPatterns(`(?i)privilege escalation|elevate to (?:root|system)|提权|提升到(?:root|system)权限`)},
	{category: "persistence_backdoor", label: "持久化或后门", patterns: compileSafetyPatterns(`(?i)backdoor|persistence mechanism|maintain access|反弹shell|后门|持久化机制`)},
	{category: "security_evasion", label: "安全规避", patterns: compileSafetyPatterns(`(?i)bypass (?:antivirus|edr|detection)|evade detection|disable (?:defender|antivirus)|绕过(?:杀毒|EDR|检测)|免杀|规避检测`)},
	{category: "data_exfiltration", label: "数据窃取或外传", patterns: compileSafetyPatterns(`(?i)data exfiltration|exfiltrate (?:data|files|credentials)|窃取数据|数据外传|导出.*凭证`)},
	{category: "scanning_reconnaissance", label: "攻击性扫描或侦察", patterns: compileSafetyPatterns(`(?i)mass scan|scan (?:the )?(?:internet|subnet).*vulnerab|enumerate targets|批量扫描|扫描.*漏洞|目标侦察`)},
	{category: "ddos_botnet", label: "DDoS 或僵尸网络", patterns: compileSafetyPatterns(`(?i)ddos|botnet|traffic flood|僵尸网络|流量洪泛|拒绝服务攻击`)},
	{category: "automated_abuse", label: "自动化滥用", patterns: compileSafetyPatterns(`(?i)credential stuffing|spam bot|mass account creation|撞库|批量注册账号|垃圾信息机器人`)},
	{category: "account_takeover", label: "账号接管", patterns: compileSafetyPatterns(`(?i)account takeover|session hijack|steal session cookie|账号接管|会话劫持`)},
	{category: "child_sexual_content", label: "未成年人性内容", patterns: compileSafetyPatterns(`(?i)child sexual|csam|sexual.*minor|未成年人.*性|儿童色情`)},
	{category: "adult_sexual_content", label: "成人露骨性内容", patterns: compileSafetyPatterns(`(?i)explicit sexual content|graphic pornography|露骨性内容|色情描写`)},
	{category: "violence_gore", label: "暴力或血腥内容", patterns: compileSafetyPatterns(`(?i)graphic gore|graphic dismemberment|torture in graphic detail|血腥肢解|详细描述酷刑|极端暴力`)},
	{category: "self_harm", label: "自残或自杀", patterns: compileSafetyPatterns(`(?i)suicide method|how to (?:kill|hurt) myself|self[- ]harm instructions|自杀方法|如何自残|结束自己的生命`)},
	{category: "hate_discrimination", label: "仇恨或歧视", patterns: compileSafetyPatterns(`(?i)racial supremacy|exterminate (?:a )?(?:race|ethnic)|仇恨言论|种族清洗|消灭.*族群`)},
	{category: "harassment_threats", label: "骚扰或威胁", patterns: compileSafetyPatterns(`(?i)credible threat|threaten to kill|dox (?:this|the) person|死亡威胁|人肉搜索|公开.*住址`)},
	{category: "extremism", label: "极端主义", patterns: compileSafetyPatterns(`(?i)terrorist recruitment|extremist propaganda|恐怖组织招募|极端主义宣传`)},
	{category: "fraud_impersonation", label: "欺诈或冒充", patterns: compileSafetyPatterns(`(?i)impersonate (?:a )?(?:bank|government|support)|investment scam|冒充(?:银行|政府|客服)|投资诈骗`)},
	{category: "privacy_doxxing", label: "隐私侵犯或人肉", patterns: compileSafetyPatterns(`(?i)doxx?ing|find (?:their|his|her) home address|人肉搜索|查找.*家庭住址`)},
	{category: "illicit_regulated_goods", label: "非法或受管制物品", patterns: compileSafetyPatterns(`(?i)buy illegal drugs|sell firearms illegally|制毒教程|非法购买枪支|违禁品交易`)},
}

func compileSafetyPatterns(patterns ...string) []*regexp.Regexp {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		compiled = append(compiled, regexp.MustCompile(pattern))
	}
	return compiled
}

func classifyContentSafetyViolation(c *gin.Context, err *types.NewAPIError, errorCode string) ContentSafetyClassification {
	oai := err.ToOpenAIError()
	officialMessage := sanitizeContentSafetyAuditText(oai.Message, 512)
	requestText := extractContentSafetyRequestText(c)
	searchText := strings.ToLower(strings.Join([]string{oai.Message, requestText}, "\n"))

	for _, rule := range contentSafetyRules {
		matched := make([]string, 0, len(rule.patterns))
		for _, pattern := range rule.patterns {
			if pattern.MatchString(searchText) {
				matched = append(matched, rule.label)
			}
		}
		if len(matched) == 0 {
			continue
		}
		sort.Strings(matched)
		matched = compactSafetySignals(matched)
		return ContentSafetyClassification{
			OfficialMessage: officialMessage, FineCategory: rule.category,
			ReasonSource: "local_rule", ReasonConfidence: "medium",
			ReasonSummary:     fmt.Sprintf("官方 %s 拒绝；本地规则识别到%s风险信号。该分类为本地推断，不代表上游提供了同名子类型；未保存原始请求正文。", errorCode, strings.Join(matched, "、")),
			ClassifierVersion: contentSafetyClassifierVersion,
		}
	}

	category, label := "safety_policy_other", "其他内容安全"
	if errorCode == "cyber_policy" {
		category, label = "cyber_policy_other", "其他网络安全高风险"
	}
	return ContentSafetyClassification{
		OfficialMessage: officialMessage, FineCategory: category,
		ReasonSource: "local_rule", ReasonConfidence: "low",
		ReasonSummary:     fmt.Sprintf("官方 %s 拒绝；本地规则未识别到足够明确的细分类信号，因此归入“%s”。该分类为本地推断；未保存原始请求正文。", errorCode, label),
		ClassifierVersion: contentSafetyClassifierVersion,
	}
}

func extractContentSafetyRequestText(c *gin.Context) string {
	if c == nil {
		return ""
	}
	storage, err := common.GetBodyStorage(c)
	if err != nil || storage.Size() <= 0 || storage.Size() > maxContentSafetyClassifierBytes {
		return ""
	}
	body, err := storage.Bytes()
	if err != nil {
		return ""
	}
	var payload any
	if common.Unmarshal(body, &payload) != nil {
		return ""
	}
	parts := make([]string, 0, 16)
	collectContentSafetyStrings(payload, "", &parts, 0)
	return strings.Join(parts, "\n")
}

func collectContentSafetyStrings(value any, key string, parts *[]string, total int) int {
	if total >= 200000 {
		return total
	}
	switch typed := value.(type) {
	case string:
		if isContentSafetyTextKey(key) {
			remaining := 200000 - total
			runes := []rune(typed)
			if len(runes) > remaining {
				runes = runes[:remaining]
			}
			*parts = append(*parts, string(runes))
			return total + len(runes)
		}
	case []any:
		for _, item := range typed {
			total = collectContentSafetyStrings(item, key, parts, total)
		}
	case map[string]any:
		for childKey, item := range typed {
			total = collectContentSafetyStrings(item, strings.ToLower(childKey), parts, total)
		}
	}
	return total
}

func isContentSafetyTextKey(key string) bool {
	switch key {
	case "input", "content", "text", "prompt", "instructions", "message", "messages", "query":
		return true
	default:
		return false
	}
}

func compactSafetySignals(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if len(result) == 0 || result[len(result)-1] != value {
			result = append(result, value)
		}
	}
	return result
}

var (
	contentSafetyEmailPattern  = regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`)
	contentSafetyURLPattern    = regexp.MustCompile(`(?i)https?://[^\s]+`)
	contentSafetySecretPattern = regexp.MustCompile(`(?i)(?:sk-[a-z0-9_\-]{12,}|(?:api[_ -]?key|password|passwd|secret|token)\s*[:=]\s*\S+)`)
	contentSafetyPhonePattern  = regexp.MustCompile(`(?:\+?\d[\d\s()\-]{8,}\d)`)
)

func sanitizeContentSafetyAuditText(value string, max int) string {
	value = common.MaskSensitiveInfo(strings.TrimSpace(value))
	value = contentSafetyEmailPattern.ReplaceAllString(value, "[redacted-email]")
	value = contentSafetyURLPattern.ReplaceAllStringFunc(value, func(raw string) string {
		if index := strings.IndexAny(raw, "?#"); index >= 0 {
			return raw[:index] + "?[redacted]"
		}
		return raw
	})
	value = contentSafetySecretPattern.ReplaceAllString(value, "[redacted-secret]")
	value = contentSafetyPhonePattern.ReplaceAllString(value, "[redacted-phone]")
	return truncateContentSafetyText(value, max)
}

func truncateContentSafetyText(value string, max int) string {
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max])
}
