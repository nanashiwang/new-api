package crsobserver

import (
	"reflect"
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

func TestQueryCRSAccountSnapshotsFiltersHealthAndOrdersAttention(t *testing.T) {
	db := setupCRSObserverTestDB(t)

	site := &model.CRSSite{
		Name:              "health-demo",
		Host:              "health.example.com",
		Scheme:            "https",
		Username:          "admin",
		PasswordEncrypted: "enc-password",
	}
	require.NoError(t, db.Create(site).Error)

	now := int64(1_800_000_000)
	require.NoError(t, model.ReplaceCRSAccountSnapshots(site.Id, []*model.CRSAccountSnapshot{
		{
			RemoteAccountID: "acct-healthy",
			Platform:        "openai-responses",
			Name:            "Healthy",
			IsActive:        true,
			Schedulable:     true,
			QuotaUnlimited:  true,
			LastSyncedAt:    now,
		},
		{
			RemoteAccountID: "acct-limited",
			Platform:        "openai-responses",
			Name:            "Limited",
			IsActive:        true,
			Schedulable:     true,
			RateLimited:     true,
			LastSyncedAt:    now,
		},
		{
			RemoteAccountID: "acct-stale",
			Platform:        "openai-responses",
			Name:            "Stale",
			IsActive:        true,
			Schedulable:     true,
			LastSyncedAt:    now - model.CRSAccountStaleAfterSeconds - 1,
		},
		{
			RemoteAccountID: "acct-boundary",
			Platform:        "openai-responses",
			Name:            "Boundary",
			IsActive:        true,
			Schedulable:     true,
			QuotaUnlimited:  true,
			LastSyncedAt:    now - model.CRSAccountStaleAfterSeconds,
		},
		{
			RemoteAccountID: "acct-empty",
			Platform:        "openai-responses",
			Name:            "Empty",
			IsActive:        true,
			Schedulable:     true,
			QuotaTotal:      100,
			QuotaRemaining:  0,
			LastSyncedAt:    now,
		},
		{
			RemoteAccountID: "acct-error",
			Platform:        "openai-responses",
			Name:            "Error",
			IsActive:        true,
			Schedulable:     true,
			SyncError:       "refresh failed",
			LastSyncedAt:    now,
		},
	}))

	available, total, err := model.QueryCRSAccountSnapshots(model.CRSAccountSnapshotQuery{
		SiteID:      site.Id,
		HealthState: "available",
		StaleBefore: now - model.CRSAccountStaleAfterSeconds,
		Page:        1,
		PageSize:    20,
	})
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	require.ElementsMatch(t, []string{"acct-healthy", "acct-boundary"}, []string{
		available[0].RemoteAccountID,
		available[1].RemoteAccountID,
	})

	attention, total, err := model.QueryCRSAccountSnapshots(model.CRSAccountSnapshotQuery{
		SiteID:         site.Id,
		HealthState:    "attention",
		StaleBefore:    now - model.CRSAccountStaleAfterSeconds,
		AttentionFirst: true,
		Page:           1,
		PageSize:       20,
	})
	require.NoError(t, err)
	require.EqualValues(t, 4, total)
	require.Equal(t, []string{"acct-error", "acct-stale", "acct-limited", "acct-empty"}, []string{
		attention[0].RemoteAccountID,
		attention[1].RemoteAccountID,
		attention[2].RemoteAccountID,
		attention[3].RemoteAccountID,
	})

	empty, total, err := model.QueryCRSAccountSnapshots(model.CRSAccountSnapshotQuery{
		SiteID:      site.Id,
		QuotaState:  "empty",
		StaleBefore: now - model.CRSAccountStaleAfterSeconds,
		Page:        1,
		PageSize:    20,
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Equal(t, "acct-empty", empty[0].RemoteAccountID)

	defaultOrder, _, err := model.QueryCRSAccountSnapshots(model.CRSAccountSnapshotQuery{
		SiteID:      site.Id,
		StaleBefore: now - model.CRSAccountStaleAfterSeconds,
		Page:        1,
		PageSize:    20,
	})
	require.NoError(t, err)
	require.Equal(t, "acct-boundary", defaultOrder[0].RemoteAccountID)
}

func TestQueryCRSAccountSnapshotsFiltersSchedulableAndSortsServerSide(t *testing.T) {
	db := setupCRSObserverTestDB(t)
	site := &model.CRSSite{
		Name:              "sort-demo",
		Host:              "sort.example.com",
		Scheme:            "https",
		Username:          "admin",
		PasswordEncrypted: "enc-password",
	}
	require.NoError(t, db.Create(site).Error)

	require.NoError(t, model.ReplaceCRSAccountSnapshots(site.Id, []*model.CRSAccountSnapshot{
		{
			RemoteAccountID:    "acct-steady",
			Platform:           "openai",
			Name:               "Steady",
			Schedulable:        true,
			QuotaTotal:         100,
			QuotaRemaining:     20,
			UsageDailyRequests: 5,
			LastSyncedAt:       300,
		},
		{
			RemoteAccountID:    "acct-busy",
			Platform:           "openai",
			Name:               "Busy",
			Schedulable:        true,
			QuotaTotal:         100,
			QuotaRemaining:     5,
			UsageDailyRequests: 50,
			LastSyncedAt:       100,
		},
		{
			RemoteAccountID:    "acct-unlimited",
			Platform:           "claude",
			Name:               "Unlimited",
			Schedulable:        false,
			QuotaUnlimited:     true,
			UsageDailyRequests: 10,
			LastSyncedAt:       200,
		},
		{
			RemoteAccountID:    "acct-unknown-quota",
			Platform:           "openai",
			Name:               "Unknown quota",
			Schedulable:        false,
			UsageDailyRequests: 1,
			LastSyncedAt:       250,
		},
	}))

	schedulable, total, err := model.QueryCRSAccountSnapshots(model.CRSAccountSnapshotQuery{
		SiteID:      site.Id,
		HealthState: "schedulable",
		Sort:        "quota_remaining",
		Page:        1,
		PageSize:    20,
	})
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	require.Equal(t, []string{"acct-busy", "acct-steady"}, []string{
		schedulable[0].RemoteAccountID,
		schedulable[1].RemoteAccountID,
	})

	quota, _, err := model.QueryCRSAccountSnapshots(model.CRSAccountSnapshotQuery{
		SiteID:   site.Id,
		Sort:     "quota_remaining",
		Page:     1,
		PageSize: 20,
	})
	require.NoError(t, err)
	require.Equal(t, []string{"acct-busy", "acct-steady", "acct-unknown-quota", "acct-unlimited"}, []string{
		quota[0].RemoteAccountID,
		quota[1].RemoteAccountID,
		quota[2].RemoteAccountID,
		quota[3].RemoteAccountID,
	})

	daily, _, err := model.QueryCRSAccountSnapshots(model.CRSAccountSnapshotQuery{
		SiteID:   site.Id,
		Sort:     "daily_requests",
		Page:     1,
		PageSize: 20,
	})
	require.NoError(t, err)
	require.Equal(t, []string{"acct-busy", "acct-unlimited", "acct-steady", "acct-unknown-quota"}, []string{
		daily[0].RemoteAccountID,
		daily[1].RemoteAccountID,
		daily[2].RemoteAccountID,
		daily[3].RemoteAccountID,
	})

	oldest, _, err := model.QueryCRSAccountSnapshots(model.CRSAccountSnapshotQuery{
		SiteID:   site.Id,
		Sort:     "last_synced",
		Page:     1,
		PageSize: 20,
	})
	require.NoError(t, err)
	require.Equal(t, []string{"acct-busy", "acct-unlimited", "acct-unknown-quota", "acct-steady"}, []string{
		oldest[0].RemoteAccountID,
		oldest[1].RemoteAccountID,
		oldest[2].RemoteAccountID,
		oldest[3].RemoteAccountID,
	})
}

func TestCRSAccountSnapshotTextFieldsDoNotDeclareDefaultValues(t *testing.T) {
	t.Parallel()

	field, ok := reflect.TypeOf(model.CRSAccountSnapshot{}).FieldByName("UsageWindowsJSON")
	require.True(t, ok)

	gormTag := field.Tag.Get("gorm")
	require.Contains(t, gormTag, "type:text")
	require.NotContains(t, gormTag, "default:")
}

func TestReplaceCRSAccountSnapshotsBackfillsUsageWindowsJSON(t *testing.T) {
	db := setupCRSObserverTestDB(t)

	site := &model.CRSSite{
		Name:              "demo",
		Host:              "usage-window.example.com",
		Scheme:            "https",
		Username:          "admin",
		PasswordEncrypted: "enc-password",
	}
	require.NoError(t, db.Create(site).Error)

	require.NoError(t, model.ReplaceCRSAccountSnapshots(site.Id, []*model.CRSAccountSnapshot{
		{
			SiteID:          site.Id,
			RemoteAccountID: "acct-usage-window",
			Platform:        "claude",
			Name:            "Usage Window",
			LastSyncedAt:    100,
		},
	}))

	rows, err := model.ListCRSAccountSnapshotsBySite(site.Id)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "[]", rows[0].UsageWindowsJSON)
}
