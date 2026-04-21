package model

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	mysqlgorm "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type saveWithoutKeyFakeConn struct {
	t       *testing.T
	queries []string
}

func (c *saveWithoutKeyFakeConn) PrepareContext(_ context.Context, query string) (*sql.Stmt, error) {
	return nil, fmt.Errorf("unexpected prepare: %s", query)
}

func (c *saveWithoutKeyFakeConn) ExecContext(_ context.Context, query string, _ ...interface{}) (sql.Result, error) {
	c.queries = append(c.queries, strings.TrimSpace(query))
	if len(c.queries) == 1 {
		// Simulate MySQL's no-op UPDATE result so we can catch Save's fallback INSERT.
		return saveWithoutKeyFakeResult(0), nil
	}
	return nil, fmt.Errorf("unexpected extra exec: %s", query)
}

func (c *saveWithoutKeyFakeConn) QueryContext(_ context.Context, query string, _ ...interface{}) (*sql.Rows, error) {
	return nil, fmt.Errorf("unexpected query: %s", query)
}

func (c *saveWithoutKeyFakeConn) QueryRowContext(_ context.Context, _ string, _ ...interface{}) *sql.Row {
	return &sql.Row{}
}

type saveWithoutKeyFakeResult int64

func (r saveWithoutKeyFakeResult) LastInsertId() (int64, error) {
	return 0, nil
}

func (r saveWithoutKeyFakeResult) RowsAffected() (int64, error) {
	return int64(r), nil
}

func TestSaveWithoutKey_NoFallbackCreateOnNoopUpdate(t *testing.T) {
	conn := &saveWithoutKeyFakeConn{t: t}
	db, err := gorm.Open(mysqlgorm.New(mysqlgorm.Config{
		Conn:                      conn,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open gorm db: %v", err)
	}

	oldDB := DB
	DB = db
	defer func() {
		DB = oldDB
	}()

	channel := &Channel{
		Id:     1,
		Key:    "sk-test",
		Name:   "tag-enable",
		Status: 1,
		Group:  "default",
	}

	if err := channel.SaveWithoutKey(); err != nil {
		t.Fatalf("SaveWithoutKey should not fall back to create when update affects 0 rows: %v", err)
	}

	if len(conn.queries) != 1 {
		t.Fatalf("expected exactly one exec, got %d: %#v", len(conn.queries), conn.queries)
	}

	if !strings.HasPrefix(conn.queries[0], "UPDATE `channels`") {
		t.Fatalf("expected update query, got: %s", conn.queries[0])
	}
}
