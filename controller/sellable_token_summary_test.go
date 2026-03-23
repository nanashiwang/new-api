package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
)

func TestSanitizeSellableTokenSummaryItems_ProjectsUnlimitedAndPackageUsage(t *testing.T) {
	items := sanitizeSellableTokenSummaryItems([]*model.Token{
		{
			Id:                     1,
			Name:                   "unlimited-package-token",
			UnlimitedQuota:         true,
			PackageEnabled:         true,
			PackageLimitQuota:      1000,
			PackageUsedQuota:       250,
			SellableTokenProductId: 7,
		},
		nil,
	})

	if len(items) != 1 {
		t.Fatalf("expected 1 summary item, got %d", len(items))
	}
	if !items[0].UnlimitedQuota {
		t.Fatal("expected unlimited_quota to be projected")
	}
	if items[0].PackageUsedQuota != 250 {
		t.Fatalf("expected package_used_quota=250, got=%d", items[0].PackageUsedQuota)
	}
}
