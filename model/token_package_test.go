package model

import (
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestNormalizeTokenPackagePeriod(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"daily", TokenPackagePeriodDaily},
		{"WEEKLY", TokenPackagePeriodWeekly},
		{" monthly ", TokenPackagePeriodMonthly},
		{"custom", TokenPackagePeriodCustom},
		{"unknown", TokenPackagePeriodNone},
		{"", TokenPackagePeriodNone},
	}
	for _, c := range cases {
		if got := NormalizeTokenPackagePeriod(c.input); got != c.want {
			t.Fatalf("NormalizeTokenPackagePeriod(%q)=%q, want=%q", c.input, got, c.want)
		}
	}
}

func TestNormalizeTokenPackagePeriodMode(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"relative", TokenPackagePeriodModeRelative},
		{"natural", TokenPackagePeriodModeNatural},
		{"NATURAL", TokenPackagePeriodModeNatural},
		{" Relative ", TokenPackagePeriodModeRelative},
		{"", TokenPackagePeriodModeRelative},
		{"unknown", TokenPackagePeriodModeRelative},
	}
	for _, c := range cases {
		if got := NormalizeTokenPackagePeriodMode(c.input); got != c.want {
			t.Fatalf("NormalizeTokenPackagePeriodMode(%q)=%q, want=%q", c.input, got, c.want)
		}
	}
}

func TestCalcNextTokenPackageResetTime(t *testing.T) {
	base := time.Date(2026, 3, 4, 10, 20, 30, 0, time.UTC)

	// natural 模式：对齐到日历边界
	daily, err := calcNextTokenPackageResetTime(base, TokenPackagePeriodDaily, 0, TokenPackagePeriodModeNatural)
	if err != nil {
		t.Fatal(err)
	}
	if daily != time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC).Unix() {
		t.Fatalf("daily next reset mismatch, got=%d", daily)
	}

	weekly, err := calcNextTokenPackageResetTime(base, TokenPackagePeriodWeekly, 0, TokenPackagePeriodModeNatural)
	if err != nil {
		t.Fatal(err)
	}
	if weekly != time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC).Unix() {
		t.Fatalf("weekly next reset mismatch, got=%d", weekly)
	}

	monthly, err := calcNextTokenPackageResetTime(base, TokenPackagePeriodMonthly, 0, TokenPackagePeriodModeNatural)
	if err != nil {
		t.Fatal(err)
	}
	if monthly != time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC).Unix() {
		t.Fatalf("monthly next reset mismatch, got=%d", monthly)
	}

	custom, err := calcNextTokenPackageResetTime(base, TokenPackagePeriodCustom, 3600, TokenPackagePeriodModeNatural)
	if err != nil {
		t.Fatal(err)
	}
	if custom != base.Add(time.Hour).Unix() {
		t.Fatalf("custom next reset mismatch, got=%d", custom)
	}
}

func TestCalcNextTokenPackageResetTime_Relative(t *testing.T) {
	base := time.Date(2026, 3, 4, 10, 20, 30, 0, time.UTC)

	daily, err := calcNextTokenPackageResetTime(base, TokenPackagePeriodDaily, 0, TokenPackagePeriodModeRelative)
	if err != nil {
		t.Fatal(err)
	}
	want := base.Add(24 * time.Hour).Unix()
	if daily != want {
		t.Fatalf("relative daily: got=%d, want=%d", daily, want)
	}

	weekly, err := calcNextTokenPackageResetTime(base, TokenPackagePeriodWeekly, 0, TokenPackagePeriodModeRelative)
	if err != nil {
		t.Fatal(err)
	}
	want = base.Add(7 * 24 * time.Hour).Unix()
	if weekly != want {
		t.Fatalf("relative weekly: got=%d, want=%d", weekly, want)
	}

	monthly, err := calcNextTokenPackageResetTime(base, TokenPackagePeriodMonthly, 0, TokenPackagePeriodModeRelative)
	if err != nil {
		t.Fatal(err)
	}
	want = base.AddDate(0, 1, 0).Unix()
	if monthly != want {
		t.Fatalf("relative monthly: got=%d, want=%d", monthly, want)
	}

	// custom 不受 periodMode 影响
	custom, err := calcNextTokenPackageResetTime(base, TokenPackagePeriodCustom, 3600, TokenPackagePeriodModeRelative)
	if err != nil {
		t.Fatal(err)
	}
	want = base.Add(time.Hour).Unix()
	if custom != want {
		t.Fatalf("relative custom: got=%d, want=%d", custom, want)
	}
}

func TestCalcNextTokenPackageResetTime_DefaultIsRelative(t *testing.T) {
	base := time.Date(2026, 6, 15, 14, 30, 0, 0, time.UTC)

	// 空字符串应默认为 relative
	monthly, err := calcNextTokenPackageResetTime(base, TokenPackagePeriodMonthly, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	want := base.AddDate(0, 1, 0).Unix()
	if monthly != want {
		t.Fatalf("empty mode should default to relative: got=%d, want=%d", monthly, want)
	}
}

func TestMaybeResetTokenPackageState_DailyReset(t *testing.T) {
	now := time.Date(2026, 3, 4, 10, 0, 0, 0, time.UTC).Unix()
	token := &Token{
		PackageEnabled:       true,
		PackageLimitQuota:    1000,
		PackagePeriod:        TokenPackagePeriodDaily,
		PackagePeriodMode:    TokenPackagePeriodModeNatural,
		RemainQuota:          1000,
		PackageUsedQuota:     300,
		PackageNextResetTime: now - 10,
	}
	changed, err := MaybeResetTokenPackageState(token, now)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected changed=true")
	}
	if token.PackageUsedQuota != 0 {
		t.Fatalf("expected package_used_quota=0, got=%d", token.PackageUsedQuota)
	}
	if token.PackageNextResetTime <= now {
		t.Fatalf("expected next_reset_time > now, got=%d", token.PackageNextResetTime)
	}
}

func TestMaybeResetTokenPackageState_RelativeMonthly(t *testing.T) {
	base := time.Date(2026, 3, 15, 14, 30, 0, 0, time.UTC)
	// 上次重置时间刚好过期
	lastReset := base.Unix() - 1
	token := &Token{
		PackageEnabled:       true,
		PackageLimitQuota:    5000,
		PackagePeriod:        TokenPackagePeriodMonthly,
		PackagePeriodMode:    TokenPackagePeriodModeRelative,
		RemainQuota:          5000,
		PackageUsedQuota:     2000,
		PackageNextResetTime: lastReset,
	}
	changed, err := MaybeResetTokenPackageState(token, base.Unix())
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected changed=true")
	}
	if token.PackageUsedQuota != 0 {
		t.Fatalf("expected package_used_quota=0, got=%d", token.PackageUsedQuota)
	}
	// relative monthly: 从上次重置点 +1 个月，而非下月 1 号
	wantNext := time.Unix(lastReset, 0).AddDate(0, 1, 0).Unix()
	if token.PackageNextResetTime != wantNext {
		t.Fatalf("relative monthly next_reset: got=%d, want=%d", token.PackageNextResetTime, wantNext)
	}
}

func TestMaybeResetTokenPackageState_RelativeInitialize(t *testing.T) {
	// PackageNextResetTime=0 表示首次初始化
	now := time.Date(2026, 5, 20, 8, 0, 0, 0, time.UTC)
	token := &Token{
		PackageEnabled:       true,
		PackageLimitQuota:    1000,
		PackagePeriod:        TokenPackagePeriodWeekly,
		PackagePeriodMode:    TokenPackagePeriodModeRelative,
		RemainQuota:          1000,
		PackageNextResetTime: 0,
	}
	changed, err := MaybeResetTokenPackageState(token, now.Unix())
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected changed=true on first init")
	}
	// relative weekly: now + 7 天
	want := now.Add(7 * 24 * time.Hour).Unix()
	if token.PackageNextResetTime != want {
		t.Fatalf("relative weekly init: got=%d, want=%d", token.PackageNextResetTime, want)
	}
}

func TestMaybeResetTokenPackageState_DisabledNormalize(t *testing.T) {
	token := &Token{
		PackageEnabled:       false,
		PackagePeriod:        TokenPackagePeriodDaily,
		PackageLimitQuota:    100,
		PackageCustomSeconds: 123,
		PackageUsedQuota:     8,
		PackageNextResetTime: 999,
	}
	changed, err := MaybeResetTokenPackageState(token, time.Now().Unix())
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected changed=true")
	}
	if token.PackagePeriod != TokenPackagePeriodNone || token.PackageCustomSeconds != 0 ||
		token.PackageUsedQuota != 0 || token.PackageNextResetTime != 0 {
		t.Fatalf("disabled token package state not normalized: %+v", token)
	}
}

func TestValidateTokenQuotaPackageRelation_RejectWhenRemainQuotaLessThanPackageLimit(t *testing.T) {
	token := &Token{
		PackageEnabled:    true,
		PackagePeriod:     TokenPackagePeriodDaily,
		PackageLimitQuota: 100,
		RemainQuota:       99,
		UnlimitedQuota:    false,
	}
	if err := ValidateTokenQuotaPackageRelation(token); err == nil {
		t.Fatal("expected error when remain_quota < package_limit_quota")
	}
}

func TestValidateTokenQuotaPackageRelation_AllowUnlimitedQuotaLessThanPackageLimit(t *testing.T) {
	token := &Token{
		PackageEnabled:    true,
		PackagePeriod:     TokenPackagePeriodDaily,
		PackageLimitQuota: 100,
		RemainQuota:       0,
		UnlimitedQuota:    true,
	}
	if err := ValidateTokenQuotaPackageRelation(token); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateTokenQuotaPackageRelation_AllowWhenTotalQuotaMeetsPackageLimit(t *testing.T) {
	token := &Token{
		PackageEnabled:    true,
		PackagePeriod:     TokenPackagePeriodDaily,
		PackageLimitQuota: 100,
		RemainQuota:       90,
		UsedQuota:         10,
		UnlimitedQuota:    false,
	}
	if err := ValidateTokenQuotaPackageRelation(token); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNormalizeTokenPackageStateForRead_PersistsExpiredCycleReset(t *testing.T) {
	cleanupTokenPackageIntegrationData(t)
	now := common.GetTimestamp()
	token := &Token{
		UserId:               1,
		Key:                  "token_package_read_reset_key_00000000000000000001",
		Status:               common.TokenStatusEnabled,
		Name:                 "package-read-reset-test",
		CreatedTime:          now,
		AccessedTime:         now,
		ExpiredTime:          -1,
		RemainQuota:          50000,
		PackageEnabled:       true,
		PackageLimitQuota:    20000,
		PackageUsedQuota:     18000,
		PackagePeriod:        TokenPackagePeriodDaily,
		PackagePeriodMode:    TokenPackagePeriodModeNatural,
		PackageNextResetTime: now - 60,
	}
	require.NoError(t, token.Insert())

	stored, err := GetTokenById(token.Id)
	require.NoError(t, err)
	require.Equal(t, 0, stored.PackageUsedQuota)
	require.Greater(t, stored.PackageNextResetTime, now)

	var persisted Token
	require.NoError(t, DB.First(&persisted, "id = ?", token.Id).Error)
	require.Equal(t, 0, persisted.PackageUsedQuota)
	require.Greater(t, persisted.PackageNextResetTime, now)
}

func TestValidateTokenCanEnable_RejectsPackageExhaustedToken(t *testing.T) {
	cleanupTokenPackageIntegrationData(t)
	now := common.GetTimestamp()
	token := &Token{
		UserId:               1,
		Key:                  "token_package_enable_block_key_0000000000000000001",
		Status:               common.TokenStatusDisabled,
		Name:                 "package-enable-block-test",
		CreatedTime:          now,
		AccessedTime:         now,
		ExpiredTime:          -1,
		RemainQuota:          50000,
		PackageEnabled:       true,
		PackageLimitQuota:    20000,
		PackageUsedQuota:     20000,
		PackagePeriod:        TokenPackagePeriodDaily,
		PackagePeriodMode:    TokenPackagePeriodModeRelative,
		PackageNextResetTime: now + 3600,
	}
	require.NoError(t, token.Insert())

	normalized, err := ValidateTokenCanEnable(token)
	require.ErrorIs(t, err, ErrTokenCannotEnablePackageExhausted)
	require.NotNil(t, normalized)
	require.Equal(t, 20000, normalized.PackageUsedQuota)
}

func cleanupTokenPackageIntegrationData(t *testing.T) {
	t.Helper()
	if DB != nil {
		_ = DB.Exec("DELETE FROM tokens").Error
	}
}

func TestDecreaseTokenQuotaWithPackage_UpdatesCycleUsage(t *testing.T) {
	cleanupTokenPackageIntegrationData(t)
	token := &Token{
		UserId:            1,
		Key:               "token_package_usage_test_key_000000000000000000001",
		Status:            common.TokenStatusEnabled,
		Name:              "package-test",
		CreatedTime:       common.GetTimestamp(),
		AccessedTime:      common.GetTimestamp(),
		ExpiredTime:       -1,
		RemainQuota:       100000,
		UsedQuota:         0,
		UnlimitedQuota:    false,
		PackageEnabled:    true,
		PackageLimitQuota: 20000,
		PackagePeriod:     TokenPackagePeriodDaily,
	}
	if err := token.Insert(); err != nil {
		t.Fatalf("insert token failed: %v", err)
	}

	const consumeQuota = 1200
	if err := DecreaseTokenQuota(token.Id, token.Key, consumeQuota); err != nil {
		t.Fatalf("decrease token quota failed: %v", err)
	}

	updated, err := GetTokenById(token.Id)
	if err != nil {
		t.Fatalf("reload token failed: %v", err)
	}
	if updated.PackageUsedQuota != consumeQuota {
		t.Fatalf("expected package_used_quota=%d, got=%d", consumeQuota, updated.PackageUsedQuota)
	}
	if updated.PackageNextResetTime <= common.GetTimestamp() {
		t.Fatalf("expected package_next_reset_time initialized, got=%d", updated.PackageNextResetTime)
	}
}

func TestDecreaseTokenQuotaWithPackageUnlimited_UpdatesCycleUsageWithoutChangingRemainQuota(t *testing.T) {
	cleanupTokenPackageIntegrationData(t)
	token := &Token{
		UserId:            1,
		Key:               "token_package_unlimited_usage_test_key_0000000000001",
		Status:            common.TokenStatusEnabled,
		Name:              "package-unlimited-test",
		CreatedTime:       common.GetTimestamp(),
		AccessedTime:      common.GetTimestamp(),
		ExpiredTime:       -1,
		RemainQuota:       0,
		UsedQuota:         0,
		UnlimitedQuota:    true,
		PackageEnabled:    true,
		PackageLimitQuota: 20000,
		PackagePeriod:     TokenPackagePeriodDaily,
	}
	if err := token.Insert(); err != nil {
		t.Fatalf("insert token failed: %v", err)
	}

	const consumeQuota = 1200
	if err := DecreaseTokenQuota(token.Id, token.Key, consumeQuota); err != nil {
		t.Fatalf("decrease token quota failed: %v", err)
	}

	updated, err := GetTokenById(token.Id)
	if err != nil {
		t.Fatalf("reload token failed: %v", err)
	}
	if updated.PackageUsedQuota != consumeQuota {
		t.Fatalf("expected package_used_quota=%d, got=%d", consumeQuota, updated.PackageUsedQuota)
	}
	if updated.UsedQuota != consumeQuota {
		t.Fatalf("expected used_quota=%d, got=%d", consumeQuota, updated.UsedQuota)
	}
	if updated.RemainQuota != 0 {
		t.Fatalf("expected remain_quota unchanged for unlimited token, got=%d", updated.RemainQuota)
	}
	if updated.PackageNextResetTime <= common.GetTimestamp() {
		t.Fatalf("expected package_next_reset_time initialized, got=%d", updated.PackageNextResetTime)
	}
}

func TestValidateUserToken_RejectsExhaustedPackageQuota(t *testing.T) {
	cleanupTokenPackageIntegrationData(t)
	token := &Token{
		UserId:               1,
		Key:                  "token_package_block_test_key_00000000000000000001",
		Status:               common.TokenStatusEnabled,
		Name:                 "package-block-test",
		CreatedTime:          common.GetTimestamp(),
		AccessedTime:         common.GetTimestamp(),
		ExpiredTime:          -1,
		RemainQuota:          100000,
		PackageEnabled:       true,
		PackageLimitQuota:    20000,
		PackageUsedQuota:     20000,
		PackagePeriod:        TokenPackagePeriodDaily,
		PackagePeriodMode:    TokenPackagePeriodModeRelative,
		PackageNextResetTime: common.GetTimestamp() + 3600,
	}
	if err := token.Insert(); err != nil {
		t.Fatalf("insert token failed: %v", err)
	}

	validated, err := ValidateUserToken(token.Key)
	if err == nil {
		t.Fatal("expected package quota error")
	}
	if validated == nil {
		t.Fatal("expected validated token returned")
	}
	if !strings.Contains(err.Error(), "套餐周期额度已用尽") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateUserToken_AllowsAfterPackageReset(t *testing.T) {
	cleanupTokenPackageIntegrationData(t)
	now := common.GetTimestamp()
	token := &Token{
		UserId:               1,
		Key:                  "token_package_reset_test_key_00000000000000000001",
		Status:               common.TokenStatusEnabled,
		Name:                 "package-reset-test",
		CreatedTime:          now,
		AccessedTime:         now,
		ExpiredTime:          -1,
		RemainQuota:          100000,
		PackageEnabled:       true,
		PackageLimitQuota:    20000,
		PackageUsedQuota:     20000,
		PackagePeriod:        TokenPackagePeriodDaily,
		PackagePeriodMode:    TokenPackagePeriodModeRelative,
		PackageNextResetTime: now - 60,
	}
	if err := token.Insert(); err != nil {
		t.Fatalf("insert token failed: %v", err)
	}

	validated, err := ValidateUserToken(token.Key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if validated.PackageUsedQuota != 0 {
		t.Fatalf("expected reset package usage, got=%d", validated.PackageUsedQuota)
	}
	if validated.PackageNextResetTime <= now {
		t.Fatalf("expected next reset in future, got=%d", validated.PackageNextResetTime)
	}
}
