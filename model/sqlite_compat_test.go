package model

import (
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestSQLiteDSNPragmasAndConcurrentReadModifyWrite(t *testing.T) {
	dsn, err := common.SQLiteDSN(filepath.Join(t.TempDir(), "compat.db"))
	require.NoError(t, err)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	conn, err := db.DB()
	require.NoError(t, err)
	defer conn.Close()
	conn.SetMaxOpenConns(8)
	var journal string
	var timeout int
	require.NoError(t, db.Raw("PRAGMA journal_mode").Scan(&journal).Error)
	require.Equal(t, "wal", journal)
	require.NoError(t, db.Raw("PRAGMA busy_timeout").Scan(&timeout).Error)
	require.Equal(t, 30000, timeout)
	type counter struct {
		ID    int `gorm:"primaryKey"`
		Value int
	}
	require.NoError(t, db.AutoMigrate(&counter{}))
	require.NoError(t, db.Create(&counter{ID: 1}).Error)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				err := db.Transaction(func(tx *gorm.DB) error {
					var row counter
					if err := tx.First(&row, 1).Error; err != nil {
						return err
					}
					return tx.Model(&row).Update("value", row.Value+1).Error
				})
				if err != nil {
					t.Error(err)
				}
			}
		}()
	}
	wg.Wait()
	var row counter
	require.NoError(t, db.First(&row, 1).Error)
	require.Equal(t, 40, row.Value)
}

func TestSQLiteDSNPreservesExplicitOptions(t *testing.T) {
	dsn, err := common.SQLiteDSN("custom.db?_pragma=busy_timeout(1234)&_pragma=journal_mode(DELETE)&_txlock=deferred&cache=shared")
	require.NoError(t, err)
	_, raw, _ := strings.Cut(dsn, "?")
	query, err := url.ParseQuery(raw)
	require.NoError(t, err)
	require.Equal(t, []string{"busy_timeout(1234)", "journal_mode(DELETE)"}, query["_pragma"])
	require.Equal(t, "deferred", query.Get("_txlock"))
	require.Equal(t, "shared", query.Get("cache"))
	dsn, err = common.SQLiteDSN("custom.db?_busy_timeout=4321")
	require.NoError(t, err)
	require.Contains(t, dsn, "busy_timeout%284321%29")
}
