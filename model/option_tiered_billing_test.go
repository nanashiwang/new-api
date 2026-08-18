package model

import (
	"sync"
	"testing"

	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestUpdateTieredBillingOptionsPersistsBothFieldsAtomically(t *testing.T) {
	savedDB := DB
	savedBundle := billing_setting.TieredBundle{
		BillingMode: billing_setting.GetBillingModeCopy(),
		BillingExpr: billing_setting.GetBillingExprCopy(),
	}
	t.Cleanup(func() {
		DB = savedDB
		billing_setting.ReplaceBundle(savedBundle)
	})

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	DB = db
	if err := DB.AutoMigrate(&Option{}); err != nil {
		t.Fatal(err)
	}

	raw := `{"billing_mode":{"tier-only":"tiered_expr"},"billing_expr":{"tier-only":"tier(\"base\", p * 2 + c * 8)"}}`
	if err := UpdateTieredBillingOptions(raw); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := DB.Model(&Option{}).
		Where("key IN ?", []string{"billing_setting.billing_mode", "billing_setting.billing_expr"}).
		Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("persisted option count = %d, want 2", count)
	}
	if billing_setting.GetBillingMode("tier-only") != billing_setting.BillingModeTieredExpr {
		t.Fatal("live tiered mode was not published")
	}
	if expr, ok := billing_setting.GetBillingExpr("tier-only"); !ok || expr == "" {
		t.Fatal("live tiered expression was not published")
	}
}

func TestConcurrentTieredBillingSavesKeepDatabaseAndRuntimeAligned(t *testing.T) {
	savedDB := DB
	savedBundle := billing_setting.TieredBundle{
		BillingMode: billing_setting.GetBillingModeCopy(),
		BillingExpr: billing_setting.GetBillingExprCopy(),
	}
	t.Cleanup(func() {
		DB = savedDB
		billing_setting.ReplaceBundle(savedBundle)
	})

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	DB = db
	if sqlDB, err := DB.DB(); err != nil {
		t.Fatal(err)
	} else {
		sqlDB.SetMaxOpenConns(1)
	}
	if err := DB.AutoMigrate(&Option{}); err != nil {
		t.Fatal(err)
	}

	raws := []string{
		`{"billing_mode":{"model-a":"tiered_expr"},"billing_expr":{"model-a":"tier(\"a\", p * 2)"}}`,
		`{"billing_mode":{"model-b":"tiered_expr"},"billing_expr":{"model-b":"tier(\"b\", c * 8)"}}`,
	}
	var wg sync.WaitGroup
	errs := make(chan error, len(raws))
	for _, raw := range raws {
		wg.Add(1)
		go func(raw string) {
			defer wg.Done()
			errs <- UpdateTieredBillingOptions(raw)
		}(raw)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}

	var stored []Option
	if err := DB.Where("key IN ?", []string{"billing_setting.billing_mode", "billing_setting.billing_expr"}).Find(&stored).Error; err != nil {
		t.Fatal(err)
	}
	values := make(map[string]string, len(stored))
	for _, option := range stored {
		values[option.Key] = option.Value
	}
	bundle, err := billing_setting.ParseAndValidateFields(
		values["billing_setting.billing_mode"],
		values["billing_setting.billing_expr"],
	)
	if err != nil {
		t.Fatal(err)
	}
	if got := billing_setting.GetBillingModeCopy(); !mapsEqual(got, bundle.BillingMode) {
		t.Fatalf("runtime modes %#v do not match stored modes %#v", got, bundle.BillingMode)
	}
	if got := billing_setting.GetBillingExprCopy(); !mapsEqual(got, bundle.BillingExpr) {
		t.Fatalf("runtime expressions %#v do not match stored expressions %#v", got, bundle.BillingExpr)
	}
}

func mapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for key, value := range a {
		if b[key] != value {
			return false
		}
	}
	return true
}

func TestLoadTieredBillingOptionsPublishesStoredPairTogether(t *testing.T) {
	savedBundle := billing_setting.TieredBundle{
		BillingMode: billing_setting.GetBillingModeCopy(),
		BillingExpr: billing_setting.GetBillingExprCopy(),
	}
	t.Cleanup(func() { billing_setting.ReplaceBundle(savedBundle) })

	loadTieredBillingOptions([]*Option{
		{Key: "billing_setting.billing_mode", Value: `{"loaded-tier":"tiered_expr"}`},
		{Key: "billing_setting.billing_expr", Value: `{"loaded-tier":"tier(\"base\", p * 2 + c * 8)"}`},
	})
	if expr, ok := billing_setting.GetTieredExpr("loaded-tier"); !ok || expr == "" {
		t.Fatal("stored tiered mode and expression were not published together")
	}
}
