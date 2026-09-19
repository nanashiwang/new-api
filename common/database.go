package common

import (
	"net/url"
	"strings"
)

const (
	DatabaseTypeMySQL      = "mysql"
	DatabaseTypeSQLite     = "sqlite"
	DatabaseTypePostgreSQL = "postgres"
)

var UsingSQLite = false
var UsingPostgreSQL = false
var LogSqlType = DatabaseTypeSQLite // Default to SQLite for logging SQL queries
var UsingMySQL = false
var UsingClickHouse = false

var SQLitePath = "one-api.db"

// SQLiteDSN supplies modernc's actual pragmas instead of the silently ignored
// mattn-style _busy_timeout. Explicit user pragmas/transaction modes win.
func SQLiteDSN(path string) (string, error) {
	base, raw, _ := strings.Cut(path, "?")
	query, err := url.ParseQuery(raw)
	if err != nil {
		return "", err
	}
	hasPragma := func(name string) bool {
		for _, pragma := range query["_pragma"] {
			pragma = strings.ToLower(strings.TrimSpace(pragma))
			if strings.HasPrefix(pragma, name+"(") || strings.HasPrefix(pragma, name+"=") {
				return true
			}
		}
		return false
	}
	if !hasPragma("busy_timeout") {
		timeout := query.Get("_busy_timeout")
		if timeout == "" {
			timeout = "30000"
		}
		query.Add("_pragma", "busy_timeout("+timeout+")")
	}
	query.Del("_busy_timeout")
	if !hasPragma("journal_mode") {
		query.Add("_pragma", "journal_mode(WAL)")
	}
	if !query.Has("_txlock") {
		query.Set("_txlock", "immediate")
	}
	return base + "?" + query.Encode(), nil
}
