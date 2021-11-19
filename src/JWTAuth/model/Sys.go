package model

import (
	"github.com/jackc/pgx/v4/pgxpool"
	"time"
)

type PgDbPool struct {
	Pool           **pgxpool.Pool
	StrPgSQLHost   string
	IntPgSQLPort   int
	StrPgSQLUser   string
	StrPgSQLPw     string
	StrPgSQLDbName string
}

// UserDataNoPwHash 表示 Pg database 中的資料列，但是去掉 password 的 hash
type UserDataNoPwHash struct {
	// -1: 人員管理者,
	// 0: API,
	// 1: 一般使用者
	IntUserCatID        int32     `json:"iUserCatID"`
	StrUserID           string    `json:"sUserID"`
	StrUserName         string    `json:"sUserName,omitempty"`
	IntIfCancel         int       `json:"iIfCancel"`
	StrCreateOpID       string    `json:"sCreateOpID"`
	StrModifyOpID       string    `json:"sModifyOpID"`
	TimeCreateTimestamp time.Time `json:"timeCreateTimestamp"`
	TimeModifyTimestamp time.Time `json:"timeModifyTimestamp"`
}

// UserDataForUpd 表示 Pg database 中的資料列，但是保留 password 的 hash
type UserDataForUpd struct {
	IntUserCatID  int32
	StrUserID     string
	StrUserName   string
	StrPwHash     string
	IntIfCancel   int
	StrModifyOpID string
}

type SessionDataForOp struct {
	StrUserID string `json:"sUserID"`
	StrIP     string `json:"sIP"`
}

type RedisKeyWithTTL struct {
	RedisKey string
	RedisTTL time.Duration
}
