package crsobserver

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupCRSObserverTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.CRSSite{}, &model.CRSAccountSnapshot{}))

	originDB := model.DB
	originLogDB := model.LOG_DB
	model.DB = db
	model.LOG_DB = db

	t.Cleanup(func() {
		model.DB = originDB
		model.LOG_DB = originLogDB
	})

	return db
}

func TestReplaceCRSAccountSnapshotsReplacesWholeSiteSnapshot(t *testing.T) {
	db := setupCRSObserverTestDB(t)

	site := &model.CRSSite{
		Name:              "demo",
		Host:              "example.com",
		Scheme:            "https",
		Username:          "admin",
		PasswordEncrypted: "enc-password",
	}
	require.NoError(t, db.Create(site).Error)

	require.NoError(t, model.ReplaceCRSAccountSnapshots(site.Id, []*model.CRSAccountSnapshot{
		{
			SiteID:           site.Id,
			RemoteAccountID:  "acct-1",
			Platform:         "claude",
			Name:             "A",
			IsActive:         true,
			Schedulable:      true,
			LastSyncedAt:     100,
			BalanceCurrency:  "USD",
			SubscriptionPlan: "max",
		},
		{
			SiteID:          site.Id,
			RemoteAccountID: "acct-2",
			Platform:        "claude-console",
			Name:            "B",
			LastSyncedAt:    100,
		},
	}))

	require.NoError(t, model.ReplaceCRSAccountSnapshots(site.Id, []*model.CRSAccountSnapshot{
		{
			SiteID:          site.Id,
			RemoteAccountID: "acct-2",
			Platform:        "claude-console",
			Name:            "B2",
			LastSyncedAt:    200,
		},
	}))

	rows, err := model.ListCRSAccountSnapshotsBySite(site.Id)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "acct-2", rows[0].RemoteAccountID)
	require.Equal(t, "B2", rows[0].Name)
	require.EqualValues(t, 200, rows[0].LastSyncedAt)
}

func TestQueryCRSAccountSnapshotsFiltersLowQuota(t *testing.T) {
	db := setupCRSObserverTestDB(t)

	site := &model.CRSSite{
		Name:              "demo",
		Host:              "observer.example.com",
		Scheme:            "https",
		Username:          "admin",
		PasswordEncrypted: "enc-password",
	}
	require.NoError(t, db.Create(site).Error)

	require.NoError(t, model.ReplaceCRSAccountSnapshots(site.Id, []*model.CRSAccountSnapshot{
		{
			SiteID:          site.Id,
			RemoteAccountID: "acct-low",
			Platform:        "claude-console",
			Name:            "Low",
			QuotaTotal:      20,
			QuotaRemaining:  5,
			LastSyncedAt:    100,
		},
		{
			SiteID:          site.Id,
			RemoteAccountID: "acct-ok",
			Platform:        "claude",
			Name:            "Okay",
			QuotaTotal:      20,
			QuotaRemaining:  18,
			LastSyncedAt:    100,
		},
	}))

	rows, total, err := model.QueryCRSAccountSnapshots(model.CRSAccountSnapshotQuery{
		SiteID:     site.Id,
		QuotaState: "low",
		Page:       1,
		PageSize:   20,
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, rows, 1)
	require.Equal(t, "acct-low", rows[0].RemoteAccountID)
}
