package model

import (
	"os"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestPostgreSQLDisablesExplicitPreparedStatements(t *testing.T) {
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN is not configured")
	}
	oldPG, oldSQLite, oldMySQL := common.UsingPostgreSQL, common.UsingSQLite, common.UsingMySQL
	t.Cleanup(func() {
		common.UsingPostgreSQL, common.UsingSQLite, common.UsingMySQL = oldPG, oldSQLite, oldMySQL
		initCol()
	})
	t.Setenv("POOLER_TEST_DSN", dsn)
	db, err := chooseDB("POOLER_TEST_DSN", false)
	require.NoError(t, err)
	conn, err := db.DB()
	require.NoError(t, err)
	defer conn.Close()
	conn.SetMaxOpenConns(1)
	require.False(t, db.PrepareStmt)
	for i := 0; i < 3; i++ {
		require.Error(t, db.Exec("SELECT definitely_missing_compat_column").Error)
		var one int
		require.NoError(t, db.Raw("SELECT ?", 1).Scan(&one).Error)
		require.Equal(t, 1, one)
	}
	var prepared int64
	require.NoError(t, db.Raw("SELECT count(*) FROM pg_prepared_statements").Scan(&prepared).Error)
	require.Zero(t, prepared)
}
