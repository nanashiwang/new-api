package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupModelPermissionTestDB(t *testing.T) {
	t.Helper()
	originDB := DB
	originLogDB := LOG_DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	DB = db
	LOG_DB = db
	InvalidateModelPermissionCache()
	t.Cleanup(func() {
		DB = originDB
		LOG_DB = originLogDB
		InvalidateModelPermissionCache()
	})
	if err := db.AutoMigrate(&ModelPermission{}); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
}

func TestModelPermissionScopeDefaultsAndRoles(t *testing.T) {
	setupModelPermissionTestDB(t)

	if got := NormalizeModelPermissionScope(99); got != ModelPermissionScopeAll {
		t.Fatalf("invalid scope normalized to %d, want all", got)
	}
	if got := NormalizeModelNameRule(99); got != NameRuleExact {
		t.Fatalf("invalid name rule normalized to %d, want exact", got)
	}

	if !IsModelVisibleToRole("missing-meta-model", common.RoleGuestUser) {
		t.Fatalf("model without permission should be visible by default")
	}
	if !IsModelCallableByRole("missing-meta-model", common.RoleCommonUser) {
		t.Fatalf("model without permission should be callable by default")
	}

	permission := ModelPermission{
		ModelName:       "admin-model",
		NameRule:        NameRuleExact,
		VisibilityScope: ModelPermissionScopeAdminOnly,
		CallScope:       ModelPermissionScopeAdminOnly,
	}
	if err := permission.Insert(); err != nil {
		t.Fatalf("insert permission: %v", err)
	}
	if IsModelVisibleToRole("admin-model", common.RoleCommonUser) {
		t.Fatalf("admin_only model should not be visible to common users")
	}
	if !IsModelVisibleToRole("admin-model", common.RoleRootUser) {
		t.Fatalf("admin_only model should be visible to root users")
	}
	if IsModelCallableByRole("admin-model", common.RoleCommonUser) {
		t.Fatalf("admin_only model should not be callable by common users")
	}
	if !IsModelCallableByRole("admin-model", common.RoleAdminUser) {
		t.Fatalf("admin_only model should be callable by admins")
	}
}

func TestModelPermissionCommonOnlyAndNoneScopes(t *testing.T) {
	setupModelPermissionTestDB(t)

	commonOnly := ModelPermission{
		ModelName:       "common-model",
		VisibilityScope: ModelPermissionScopeCommonOnly,
		CallScope:       ModelPermissionScopeCommonOnly,
	}
	if err := commonOnly.Insert(); err != nil {
		t.Fatalf("insert common permission: %v", err)
	}
	if !IsModelVisibleToRole("common-model", common.RoleGuestUser) {
		t.Fatalf("common_only visibility should include guests")
	}
	if !IsModelVisibleToRole("common-model", common.RoleCommonUser) {
		t.Fatalf("common_only visibility should include common users")
	}
	if IsModelVisibleToRole("common-model", common.RoleAdminUser) {
		t.Fatalf("common_only visibility should exclude admins")
	}
	if IsModelCallableByRole("common-model", common.RoleGuestUser) {
		t.Fatalf("common_only call should exclude guests")
	}
	if !IsModelCallableByRole("common-model", common.RoleCommonUser) {
		t.Fatalf("common_only call should include common users")
	}

	none := ModelPermission{
		ModelName:       "none-model",
		VisibilityScope: ModelPermissionScopeNone,
		CallScope:       ModelPermissionScopeNone,
	}
	if err := none.Insert(); err != nil {
		t.Fatalf("insert none permission: %v", err)
	}
	if IsModelVisibleToRole("none-model", common.RoleRootUser) {
		t.Fatalf("none visibility should reject root")
	}
	if IsModelCallableByRole("none-model", common.RoleRootUser) {
		t.Fatalf("none call should reject root")
	}
}

func TestModelPermissionMatchPriority(t *testing.T) {
	setupModelPermissionTestDB(t)

	rules := []ModelPermission{
		{ModelName: "gpt", NameRule: NameRuleContains, CallScope: ModelPermissionScopeNone},
		{ModelName: "turbo", NameRule: NameRuleSuffix, CallScope: ModelPermissionScopeCommonOnly},
		{ModelName: "gpt-4", NameRule: NameRulePrefix, CallScope: ModelPermissionScopeAdminOnly},
		{ModelName: "gpt-4-turbo", NameRule: NameRuleExact, CallScope: ModelPermissionScopeAll},
		{ModelName: "claude-", NameRule: NameRulePrefix, CallScope: ModelPermissionScopeAdminOnly},
		{ModelName: "claude-3-", NameRule: NameRulePrefix, CallScope: ModelPermissionScopeNone},
		{ModelName: "mini", NameRule: NameRuleContains, CallScope: ModelPermissionScopeAdminOnly},
		{ModelName: "mini", NameRule: NameRuleContains, CallScope: ModelPermissionScopeNone},
	}
	for i := range rules {
		if err := DB.Create(&rules[i]).Error; err != nil {
			t.Fatalf("insert rule %d: %v", i, err)
		}
	}
	InvalidateModelPermissionCache()

	tests := []struct {
		name      string
		modelName string
		wantScope int
	}{
		{name: "exact wins", modelName: "gpt-4-turbo", wantScope: ModelPermissionScopeAll},
		{name: "prefix before contains", modelName: "gpt-4o", wantScope: ModelPermissionScopeAdminOnly},
		{name: "suffix before contains", modelName: "my-turbo", wantScope: ModelPermissionScopeCommonOnly},
		{name: "contains fallback", modelName: "xxgptyy", wantScope: ModelPermissionScopeNone},
		{name: "longer same-rule wins", modelName: "claude-3-haiku", wantScope: ModelPermissionScopeNone},
		{name: "smaller id tie wins", modelName: "mini-model", wantScope: ModelPermissionScopeAdminOnly},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetModelAccess(tt.modelName).CallScope
			if got != tt.wantScope {
				t.Fatalf("call scope = %d, want %d", got, tt.wantScope)
			}
		})
	}
}

func TestModelPermissionVisibilityAndCallAreIndependent(t *testing.T) {
	setupModelPermissionTestDB(t)

	visibleButBlocked := ModelPermission{
		ModelName:       "visible-blocked",
		VisibilityScope: ModelPermissionScopeAll,
		CallScope:       ModelPermissionScopeNone,
	}
	hiddenButCallable := ModelPermission{
		ModelName:       "hidden-callable",
		VisibilityScope: ModelPermissionScopeNone,
		CallScope:       ModelPermissionScopeAll,
	}
	if err := visibleButBlocked.Insert(); err != nil {
		t.Fatalf("insert visible blocked: %v", err)
	}
	if err := hiddenButCallable.Insert(); err != nil {
		t.Fatalf("insert hidden callable: %v", err)
	}

	if !IsModelVisibleToRole("visible-blocked", common.RoleCommonUser) {
		t.Fatalf("visible-blocked should be visible")
	}
	if IsModelCallableByRole("visible-blocked", common.RoleCommonUser) {
		t.Fatalf("visible-blocked should not be callable")
	}
	if IsModelVisibleToRole("hidden-callable", common.RoleCommonUser) {
		t.Fatalf("hidden-callable should not be visible")
	}
	if !IsModelCallableByRole("hidden-callable", common.RoleCommonUser) {
		t.Fatalf("hidden-callable should be callable")
	}
}
