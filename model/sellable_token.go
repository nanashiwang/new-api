package model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	SellableTokenProductStatusEnabled  = 1
	SellableTokenProductStatusDisabled = 2

	SellableTokenOrderStatusCompleted = 1

	SellableTokenIssuanceStatusPending   = "pending"
	SellableTokenIssuanceStatusIssued    = "issued"
	SellableTokenIssuanceStatusCancelled = "cancelled"

	SellableTokenSourceTypeRedeem = "redeem"
	SellableTokenSourceTypeWallet = "wallet"
	SellableTokenSourceTypeAdmin  = "admin"

	TokenSourceTypeSellableToken = "sellable_token"
	TokenBillingModeTokenOnly    = "token_only"
)

type SellableTokenProduct struct {
	Id                   int            `json:"id"`
	Name                 string         `json:"name" gorm:"type:varchar(128);not null;index"`
	Subtitle             string         `json:"subtitle" gorm:"type:varchar(255);default:''"`
	Status               int            `json:"status" gorm:"type:int;default:1;index"`
	SortOrder            int            `json:"sort_order" gorm:"type:int;default:0"`
	PriceQuota           int            `json:"price_quota" gorm:"type:int;default:0"`
	PriceAmount          float64        `json:"price_amount" gorm:"type:decimal(10,4);default:0"`
	TotalQuota           int            `json:"total_quota" gorm:"type:int;default:0"`
	UnlimitedQuota       bool           `json:"unlimited_quota" gorm:"default:false"`
	ValiditySeconds      int64          `json:"validity_seconds" gorm:"type:bigint;default:0"`
	ModelLimitsEnabled   bool           `json:"model_limits_enabled"`
	ModelLimits          string         `json:"model_limits" gorm:"type:varchar(2048);default:''"`
	AllowedGroups        string         `json:"allowed_groups" gorm:"type:varchar(1024);default:''"`
	MaxConcurrency       int            `json:"max_concurrency" gorm:"type:int;default:0"`
	WindowRequestLimit   int            `json:"window_request_limit" gorm:"type:int;default:0"`
	WindowSeconds        int64          `json:"window_seconds" gorm:"type:bigint;default:0"`
	PackageEnabled       bool           `json:"package_enabled" gorm:"default:false"`
	PackageLimitQuota    int            `json:"package_limit_quota" gorm:"type:int;default:0"`
	PackagePeriod        string         `json:"package_period" gorm:"type:varchar(16);default:'none'"`
	PackageCustomSeconds int64          `json:"package_custom_seconds" gorm:"type:bigint;default:0"`
	PackagePeriodMode    string         `json:"package_period_mode" gorm:"type:varchar(16);default:'relative'"`
	CreatedTime          int64          `json:"created_time" gorm:"bigint"`
	UpdatedTime          int64          `json:"updated_time" gorm:"bigint"`
	DeletedAt            gorm.DeletedAt `gorm:"index"`
}

func (p *SellableTokenProduct) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	p.CreatedTime = now
	p.UpdatedTime = now
	return nil
}

func (p *SellableTokenProduct) BeforeUpdate(tx *gorm.DB) error {
	p.UpdatedTime = common.GetTimestamp()
	return nil
}

func (p *SellableTokenProduct) Normalize() {
	p.Name = strings.TrimSpace(p.Name)
	p.Subtitle = strings.TrimSpace(p.Subtitle)
	p.ModelLimits = normalizeCommaList(p.ModelLimits)
	p.AllowedGroups = normalizeCommaList(p.AllowedGroups)
	if p.Status != SellableTokenProductStatusDisabled {
		p.Status = SellableTokenProductStatusEnabled
	}
	if p.PriceQuota < 0 {
		p.PriceQuota = 0
	}
	if p.PriceAmount < 0 {
		p.PriceAmount = 0
	}
	if p.TotalQuota < 0 {
		p.TotalQuota = 0
	}
	if p.UnlimitedQuota {
		p.TotalQuota = 0
	}
	if p.ValiditySeconds < 0 {
		p.ValiditySeconds = 0
	}
	if p.MaxConcurrency < 0 {
		p.MaxConcurrency = 0
	}
	if p.WindowRequestLimit < 0 {
		p.WindowRequestLimit = 0
	}
	if p.WindowSeconds < 0 {
		p.WindowSeconds = 0
	}
	p.PackagePeriod = NormalizeTokenPackagePeriod(p.PackagePeriod)
	p.PackagePeriodMode = NormalizeTokenPackagePeriodMode(p.PackagePeriodMode)
	if !p.ModelLimitsEnabled {
		p.ModelLimits = ""
	}
	if !p.PackageEnabled {
		p.PackageLimitQuota = 0
		p.PackagePeriod = TokenPackagePeriodNone
		p.PackageCustomSeconds = 0
		p.PackagePeriodMode = TokenPackagePeriodModeRelative
	}
}

func ValidateSellableTokenProduct(product *SellableTokenProduct) error {
	if product == nil {
		return errors.New("product is nil")
	}
	product.Normalize()
	if product.Name == "" {
		return errors.New("可售令牌名称不能为空")
	}
	if product.TotalQuota <= 0 && !product.UnlimitedQuota {
		return errors.New("可售令牌总额度必须大于 0，或开启无限额度")
	}
	if product.PriceQuota < 0 {
		return errors.New("售价不能小于 0")
	}
	if product.WindowRequestLimit > 0 && product.WindowSeconds <= 0 {
		return errors.New("设置请求窗口限制时，窗口时长必须大于 0")
	}
	if product.WindowSeconds > 0 && product.WindowRequestLimit <= 0 {
		return errors.New("设置窗口时长时，请同时设置窗口请求上限")
	}
	tokenForValidation := &Token{
		MaxConcurrency:       product.MaxConcurrency,
		WindowRequestLimit:   product.WindowRequestLimit,
		WindowSeconds:        product.WindowSeconds,
		PackageEnabled:       product.PackageEnabled,
		PackageLimitQuota:    product.PackageLimitQuota,
		PackagePeriod:        product.PackagePeriod,
		PackageCustomSeconds: product.PackageCustomSeconds,
		PackagePeriodMode:    product.PackagePeriodMode,
		PackageUsedQuota:     0,
		PackageNextResetTime: 0,
		RemainQuota:          product.TotalQuota,
		UsedQuota:            0,
		UnlimitedQuota:       product.UnlimitedQuota,
	}
	if err := ValidateTokenPackageConfig(tokenForValidation); err != nil {
		return err
	}
	if err := ValidateTokenRuntimeLimitConfig(tokenForValidation); err != nil {
		return err
	}
	if err := ValidateTokenQuotaPackageRelation(tokenForValidation); err != nil {
		return err
	}
	return nil
}

func CreateSellableTokenProduct(product *SellableTokenProduct) error {
	if err := ValidateSellableTokenProduct(product); err != nil {
		return err
	}
	return DB.Create(product).Error
}

func GetSellableTokenProductById(id int) (*SellableTokenProduct, error) {
	if id <= 0 {
		return nil, errors.New("无效的可售令牌商品 ID")
	}
	var product SellableTokenProduct
	if err := DB.First(&product, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &product, nil
}

func ListSellableTokenProducts(includeDisabled bool) ([]*SellableTokenProduct, error) {
	var products []*SellableTokenProduct
	query := DB.Model(&SellableTokenProduct{}).Order("sort_order desc, id desc")
	if !includeDisabled {
		query = query.Where("status = ?", SellableTokenProductStatusEnabled)
	}
	if err := query.Find(&products).Error; err != nil {
		return nil, err
	}
	return products, nil
}

func UpdateSellableTokenProductStatus(id int, status int) error {
	if id <= 0 {
		return errors.New("无效的可售令牌商品 ID")
	}
	if status != SellableTokenProductStatusEnabled && status != SellableTokenProductStatusDisabled {
		return errors.New("无效的可售令牌商品状态")
	}
	return DB.Model(&SellableTokenProduct{}).Where("id = ?", id).Update("status", status).Error
}

func DeleteSellableTokenProduct(id int) error {
	if id <= 0 {
		return errors.New("无效的可售令牌商品 ID")
	}
	return DB.Delete(&SellableTokenProduct{}, "id = ?", id).Error
}

func UpdateSellableTokenProduct(id int, product *SellableTokenProduct) error {
	if id <= 0 {
		return errors.New("无效的可售令牌商品 ID")
	}
	if err := ValidateSellableTokenProduct(product); err != nil {
		return err
	}
	updates := map[string]any{
		"name":                   product.Name,
		"subtitle":               product.Subtitle,
		"status":                 product.Status,
		"sort_order":             product.SortOrder,
		"price_quota":            product.PriceQuota,
		"price_amount":           product.PriceAmount,
		"total_quota":            product.TotalQuota,
		"unlimited_quota":        product.UnlimitedQuota,
		"validity_seconds":       product.ValiditySeconds,
		"model_limits_enabled":   product.ModelLimitsEnabled,
		"model_limits":           product.ModelLimits,
		"allowed_groups":         product.AllowedGroups,
		"max_concurrency":        product.MaxConcurrency,
		"window_request_limit":   product.WindowRequestLimit,
		"window_seconds":         product.WindowSeconds,
		"package_enabled":        product.PackageEnabled,
		"package_limit_quota":    product.PackageLimitQuota,
		"package_period":         product.PackagePeriod,
		"package_custom_seconds": product.PackageCustomSeconds,
		"package_period_mode":    product.PackagePeriodMode,
		"updated_time":           common.GetTimestamp(),
	}
	return DB.Model(&SellableTokenProduct{}).Where("id = ?", id).Updates(updates).Error
}

func (p *SellableTokenProduct) GetModelLimits() []string {
	if p == nil || p.ModelLimits == "" {
		return []string{}
	}
	return splitCommaList(p.ModelLimits)
}

func (p *SellableTokenProduct) GetAllowedGroups() []string {
	if p == nil || p.AllowedGroups == "" {
		return []string{}
	}
	return splitCommaList(p.AllowedGroups)
}

type SellableTokenOrder struct {
	Id           int            `json:"id"`
	UserId       int            `json:"user_id" gorm:"index"`
	ProductId    int            `json:"product_id" gorm:"index"`
	PriceQuota   int            `json:"price_quota" gorm:"type:int;default:0"`
	Status       int            `json:"status" gorm:"type:int;default:1"`
	CreateTime   int64          `json:"create_time" gorm:"bigint"`
	CompleteTime int64          `json:"complete_time" gorm:"bigint"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

func (o *SellableTokenOrder) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	o.CreateTime = now
	if o.CompleteTime == 0 {
		o.CompleteTime = now
	}
	if o.Status == 0 {
		o.Status = SellableTokenOrderStatusCompleted
	}
	return nil
}

type SellableTokenIssuance struct {
	Id               int                   `json:"id"`
	UserId           int                   `json:"user_id" gorm:"index"`
	ProductId        int                   `json:"product_id" gorm:"index"`
	SourceType       string                `json:"source_type" gorm:"type:varchar(32);not null;default:'redeem'"`
	SourceId         int                   `json:"source_id" gorm:"type:int;default:0"`
	Status           string                `json:"status" gorm:"type:varchar(16);not null;default:'pending';index"`
	TokenId          *int                  `json:"token_id" gorm:"type:int"`
	IssueMode        string                `json:"issue_mode" gorm:"type:varchar(16);default:''"`
	TargetTokenId    *int                  `json:"target_token_id" gorm:"type:int"`
	RequestedName    string                `json:"requested_name" gorm:"type:varchar(128);default:''"`
	RequestedGroup   string                `json:"requested_group" gorm:"type:varchar(64);default:''"`
	CreatedTime      int64                 `json:"created_time" gorm:"bigint"`
	UpdatedTime      int64                 `json:"updated_time" gorm:"bigint"`
	IssuedTime       int64                 `json:"issued_time" gorm:"bigint;default:0"`
	DeletedAt        gorm.DeletedAt        `gorm:"index"`
	Product          *SellableTokenProduct `json:"product,omitempty" gorm:"foreignKey:ProductId"`
	Token            *Token                `json:"token,omitempty" gorm:"foreignKey:TokenId"`
	RenewableTargets []*Token              `json:"renewable_targets,omitempty" gorm:"-"`
}

func (i *SellableTokenIssuance) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	i.CreatedTime = now
	i.UpdatedTime = now
	if i.Status == "" {
		i.Status = SellableTokenIssuanceStatusPending
	}
	return nil
}

func (i *SellableTokenIssuance) BeforeUpdate(tx *gorm.DB) error {
	i.UpdatedTime = common.GetTimestamp()
	return nil
}

func CreateSellableTokenIssuanceTx(tx *gorm.DB, issuance *SellableTokenIssuance) error {
	if tx == nil {
		tx = DB
	}
	if issuance == nil {
		return errors.New("issuance is nil")
	}
	if issuance.UserId <= 0 || issuance.ProductId <= 0 {
		return errors.New("待发放记录参数无效")
	}
	issuance.SourceType = strings.TrimSpace(issuance.SourceType)
	if issuance.SourceType == "" {
		issuance.SourceType = SellableTokenSourceTypeRedeem
	}
	issuance.TokenId = nil
	issuance.TargetTokenId = nil
	issuance.Status = SellableTokenIssuanceStatusPending
	return tx.Create(issuance).Error
}

func GetSellableTokenIssuanceByIdForUser(id int, userId int) (*SellableTokenIssuance, error) {
	if id <= 0 || userId <= 0 {
		return nil, errors.New("无效的待发放记录")
	}
	var issuance SellableTokenIssuance
	if err := DB.Preload("Product").Preload("Token").First(&issuance, "id = ? AND user_id = ?", id, userId).Error; err != nil {
		return nil, err
	}
	return &issuance, nil
}

func ListSellableTokenIssuancesByUser(userId int, status string) ([]*SellableTokenIssuance, error) {
	if userId <= 0 {
		return nil, errors.New("无效的用户")
	}
	var issuances []*SellableTokenIssuance
	query := DB.Preload("Product").Preload("Token").Where("user_id = ?", userId).Order("id desc")
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Find(&issuances).Error; err != nil {
		return nil, err
	}
	return issuances, nil
}

func ListRenewableSellableTokens(userId int, productId int) ([]*Token, error) {
	if userId <= 0 || productId <= 0 {
		return []*Token{}, nil
	}
	var tokens []*Token
	if err := DB.Where(
		"user_id = ? AND sellable_token_product_id = ? AND source_type = ? AND status = ?",
		userId,
		productId,
		TokenSourceTypeSellableToken,
		common.TokenStatusEnabled,
	).
		Order("id desc").Find(&tokens).Error; err != nil {
		return nil, err
	}
	return tokens, nil
}

func ResolveSellableTokenIssuanceDetails(issuance *SellableTokenIssuance) error {
	if issuance == nil {
		return errors.New("issuance is nil")
	}
	if issuance.Product == nil && issuance.ProductId > 0 {
		product, err := GetSellableTokenProductById(issuance.ProductId)
		if err != nil {
			var unscopedProduct SellableTokenProduct
			if loadErr := DB.Unscoped().First(&unscopedProduct, "id = ?", issuance.ProductId).Error; loadErr != nil {
				return err
			}
			product = &unscopedProduct
		}
		issuance.Product = product
	}
	targets, err := ListRenewableSellableTokens(issuance.UserId, issuance.ProductId)
	if err != nil {
		return err
	}
	issuance.RenewableTargets = targets
	return nil
}

func cloneTokenConfigFromSellableProduct(product *SellableTokenProduct) *Token {
	if product == nil {
		return nil
	}
	return &Token{
		RemainQuota:            product.TotalQuota,
		UnlimitedQuota:         product.UnlimitedQuota,
		ModelLimitsEnabled:     product.ModelLimitsEnabled,
		ModelLimits:            product.ModelLimits,
		BillingMode:            TokenBillingModeTokenOnly,
		SourceType:             TokenSourceTypeSellableToken,
		SellableTokenProductId: product.Id,
		MaxConcurrency:         product.MaxConcurrency,
		WindowRequestLimit:     product.WindowRequestLimit,
		WindowSeconds:          product.WindowSeconds,
		PackageEnabled:         product.PackageEnabled,
		PackageLimitQuota:      product.PackageLimitQuota,
		PackagePeriod:          product.PackagePeriod,
		PackageCustomSeconds:   product.PackageCustomSeconds,
		PackagePeriodMode:      product.PackagePeriodMode,
	}
}

func CalcSellableTokenExpiry(baseUnix int64, validitySeconds int64) int64 {
	if validitySeconds <= 0 {
		return -1
	}
	base := baseUnix
	if base <= 0 {
		base = common.GetTimestamp()
	}
	return base + validitySeconds
}

func RenewSellableTokenTx(tx *gorm.DB, token *Token, product *SellableTokenProduct) error {
	if tx == nil {
		tx = DB
	}
	if token == nil || product == nil {
		return errors.New("token 或 product 不能为空")
	}
	config := cloneTokenConfigFromSellableProduct(product)
	if config == nil {
		return errors.New("无法生成可售令牌配置")
	}
	updates := map[string]any{
		"unlimited_quota":           product.UnlimitedQuota,
		"billing_mode":              TokenBillingModeTokenOnly,
		"source_type":               TokenSourceTypeSellableToken,
		"sellable_token_product_id": product.Id,
		"max_concurrency":           product.MaxConcurrency,
		"window_request_limit":      product.WindowRequestLimit,
		"window_seconds":            product.WindowSeconds,
		"model_limits_enabled":      product.ModelLimitsEnabled,
		"model_limits":              product.ModelLimits,
		"package_enabled":           product.PackageEnabled,
		"package_limit_quota":       product.PackageLimitQuota,
		"package_period":            product.PackagePeriod,
		"package_custom_seconds":    product.PackageCustomSeconds,
		"package_period_mode":       product.PackagePeriodMode,
		"accessed_time":             common.GetTimestamp(),
	}
	if !product.UnlimitedQuota {
		updates["remain_quota"] = gorm.Expr("remain_quota + ?", product.TotalQuota)
	}
	if product.ValiditySeconds > 0 {
		baseExpiry := token.ExpiredTime
		now := common.GetTimestamp()
		if baseExpiry < now {
			baseExpiry = now
		}
		updates["expired_time"] = CalcSellableTokenExpiry(baseExpiry, product.ValiditySeconds)
	}
	return tx.Model(&Token{}).Where("id = ?", token.Id).Updates(updates).Error
}

func CreateSellableTokenTx(tx *gorm.DB, token *Token) error {
	if tx == nil {
		tx = DB
	}
	if token == nil {
		return errors.New("token 不能为空")
	}
	return tx.Create(token).Error
}

func GetRecentConsumedQuotaByToken(tokenId int, sinceUnix int64) (int, error) {
	if tokenId <= 0 {
		return 0, nil
	}
	var total int64
	err := LOG_DB.Table("logs").
		Select("COALESCE(SUM(quota), 0)").
		Where("token_id = ? AND type = ? AND created_at >= ?", tokenId, LogTypeConsume, sinceUnix).
		Scan(&total).Error
	return int(total), err
}

func normalizeCommaList(value string) string {
	return strings.Join(splitCommaList(value), ",")
}

func splitCommaList(value string) []string {
	raw := strings.Split(strings.ReplaceAll(value, "\n", ","), ",")
	result := make([]string, 0, len(raw))
	seen := make(map[string]struct{}, len(raw))
	for _, item := range raw {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

func CancelSellableTokenIssuanceTx(tx *gorm.DB, issuanceId int, userId int) (refundQuota int, err error) {
	if tx == nil {
		tx = DB
	}
	if issuanceId <= 0 || userId <= 0 {
		return 0, errors.New("无效的待发放记录")
	}
	query := tx.Where("id = ? AND user_id = ?", issuanceId, userId)
	if !common.UsingSQLite {
		query = query.Set("gorm:query_option", "FOR UPDATE")
	}
	var issuance SellableTokenIssuance
	if err := query.First(&issuance).Error; err != nil {
		return 0, err
	}
	if issuance.Status != SellableTokenIssuanceStatusPending {
		return 0, errors.New("该待发放记录已处理，无法取消")
	}
	issuance.Status = SellableTokenIssuanceStatusCancelled
	issuance.UpdatedTime = common.GetTimestamp()
	if err := tx.Save(&issuance).Error; err != nil {
		return 0, err
	}
	if issuance.SourceType == SellableTokenSourceTypeWallet {
		var order SellableTokenOrder
		if err := tx.Where("id = ? AND user_id = ?", issuance.SourceId, userId).First(&order).Error; err == nil {
			return order.PriceQuota, nil
		}
		return 0, nil
	}
	return 0, nil
}

func CancelAllPendingSellableTokenIssuancesTx(tx *gorm.DB, userId int) (cancelledCount int, totalRefund int, err error) {
	if tx == nil {
		tx = DB
	}
	if userId <= 0 {
		return 0, 0, errors.New("无效的用户")
	}
	var issuances []SellableTokenIssuance
	if err := tx.Where("user_id = ? AND status = ?", userId, SellableTokenIssuanceStatusPending).Find(&issuances).Error; err != nil {
		return 0, 0, err
	}
	if len(issuances) == 0 {
		return 0, 0, nil
	}
	now := common.GetTimestamp()
	for i := range issuances {
		issuances[i].Status = SellableTokenIssuanceStatusCancelled
		issuances[i].UpdatedTime = now
		if err := tx.Save(&issuances[i]).Error; err != nil {
			return cancelledCount, totalRefund, err
		}
		cancelledCount++
		if issuances[i].SourceType == SellableTokenSourceTypeWallet && issuances[i].SourceId > 0 {
			var order SellableTokenOrder
			if err := tx.Where("id = ? AND user_id = ?", issuances[i].SourceId, userId).First(&order).Error; err == nil {
				totalRefund += order.PriceQuota
			}
		}
	}
	return cancelledCount, totalRefund, nil
}

func ValidateSellableTokenProductAvailability(product *SellableTokenProduct) error {
	if product == nil {
		return errors.New("可售令牌商品不存在")
	}
	if product.Status != SellableTokenProductStatusEnabled {
		return errors.New("可售令牌商品已下架")
	}
	return nil
}

func MustGetSellableTokenProductAvailableTx(tx *gorm.DB, productId int) (*SellableTokenProduct, error) {
	if productId <= 0 {
		return nil, errors.New("无效的可售令牌商品 ID")
	}
	var product SellableTokenProduct
	query := tx.Where("id = ?", productId)
	if !common.UsingSQLite {
		query = query.Set("gorm:query_option", "FOR UPDATE")
	}
	if err := query.First(&product).Error; err != nil {
		return nil, err
	}
	if err := ValidateSellableTokenProductAvailability(&product); err != nil {
		return nil, err
	}
	return &product, nil
}

func BuildSellableTokenName(product *SellableTokenProduct, requestedName string) string {
	requestedName = strings.TrimSpace(requestedName)
	if requestedName != "" {
		return requestedName
	}
	if product == nil {
		return "可售令牌"
	}
	return product.Name
}

func ValidateSellableTokenGroupChoice(group string, allowedGroups []string) error {
	group = strings.TrimSpace(group)
	if group == "" {
		return errors.New("请选择分组")
	}
	if len(allowedGroups) == 0 {
		return nil
	}
	for _, item := range allowedGroups {
		if item == group {
			return nil
		}
	}
	return fmt.Errorf("分组 %s 不在可选范围内", group)
}
