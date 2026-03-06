package model

import (
	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// GetDBTimestamp 返回数据库时间对应的 UNIX 时间戳。
// 出错时回退到应用本地时间。
func GetDBTimestamp() int64 {
	return getDBTimestampWithDB(DB)
}

// GetDBTimestampTx 使用同一个事务连接读取 UNIX 时间戳。
// 出错时回退到应用本地时间。
func GetDBTimestampTx(tx *gorm.DB) int64 {
	return getDBTimestampWithDB(tx)
}

func getDBTimestampWithDB(db *gorm.DB) int64 {
	if db == nil {
		db = DB
	}
	var ts int64
	var err error
	switch {
	case common.UsingPostgreSQL:
		err = db.Raw("SELECT EXTRACT(EPOCH FROM NOW())::bigint").Scan(&ts).Error
	case common.UsingSQLite:
		err = db.Raw("SELECT strftime('%s','now')").Scan(&ts).Error
	default:
		err = db.Raw("SELECT UNIX_TIMESTAMP()").Scan(&ts).Error
	}
	if err != nil || ts <= 0 {
		return common.GetTimestamp()
	}
	return ts
}
