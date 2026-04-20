package crssiteregression

import (
	"reflect"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupCRSSiteTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.CRSSite{}))

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

func TestCRSSiteTextFieldsDoNotDeclareDefaultValues(t *testing.T) {
	t.Parallel()

	textFields := []string{
		"PasswordEncrypted",
		"TokenEncrypted",
		"LastSyncError",
		"CachedStats",
	}

	siteType := reflect.TypeOf(model.CRSSite{})
	for _, fieldName := range textFields {
		field, ok := siteType.FieldByName(fieldName)
		require.True(t, ok, "field %s should exist", fieldName)

		gormTag := field.Tag.Get("gorm")
		require.Contains(t, gormTag, "type:text", "field %s should stay as text", fieldName)
		require.NotContains(t, gormTag, "default:", "field %s must not declare a DB default", fieldName)
	}
}

func TestPersistCRSSiteStatsPreservesCachedStatsWhenStatsNotProvided(t *testing.T) {
	db := setupCRSSiteTestDB(t)

	site := &model.CRSSite{
		Name:              "demo",
		Host:              "example.com",
		Scheme:            "https",
		Username:          "admin",
		PasswordEncrypted: "enc-password",
		CachedStats:       `{"cached":true}`,
		Status:            model.CRSSiteStatusSynced,
	}
	require.NoError(t, db.Create(site).Error)

	require.NoError(t, model.PersistCRSSiteStats(site.Id, "enc-token", 321, "", model.CRSSiteStatusError, "refresh failed"))

	var stored model.CRSSite
	require.NoError(t, db.First(&stored, "id = ?", site.Id).Error)
	require.Equal(t, `{"cached":true}`, stored.CachedStats)
	require.Equal(t, "enc-token", stored.TokenEncrypted)
	require.EqualValues(t, 321, stored.TokenExpiresAt)
	require.Equal(t, model.CRSSiteStatusError, stored.Status)
	require.Equal(t, "refresh failed", stored.LastSyncError)
	require.NotZero(t, stored.LastSyncedAt)
}

func TestPersistCRSSiteStatsUpdatesCachedStatsWhenProvided(t *testing.T) {
	db := setupCRSSiteTestDB(t)

	site := &model.CRSSite{
		Name:              "demo",
		Host:              "example.org",
		Scheme:            "https",
		Username:          "admin",
		PasswordEncrypted: "enc-password",
		CachedStats:       `{"cached":true}`,
		Status:            model.CRSSiteStatusPending,
	}
	require.NoError(t, db.Create(site).Error)

	require.NoError(t, model.PersistCRSSiteStats(site.Id, "", 0, `{"cached":false}`, model.CRSSiteStatusSynced, ""))

	var stored model.CRSSite
	require.NoError(t, db.First(&stored, "id = ?", site.Id).Error)
	require.Equal(t, `{"cached":false}`, strings.TrimSpace(stored.CachedStats))
	require.Equal(t, model.CRSSiteStatusSynced, stored.Status)
	require.Empty(t, stored.LastSyncError)
}
