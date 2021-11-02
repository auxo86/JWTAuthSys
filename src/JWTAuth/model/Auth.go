package model

import (
	"time"
)

// UsrAuthData 是前端傳來認證的 JSON 必要的內容
type UsrAuthData struct {
	UserID string `json:"userid"`
	Pw     string `json:"pw"`
	// 是 int64 nanosecond count
	// 1 second = 1 000 000 000 nanoseconds
	JWTRedisTTL   time.Duration `json:"JWTRedisTTL"`
	AllowedUserIP string        `json:"AllowedUserIP"`
}

// 是資料庫中儲存的資料
type DBUserCredentials struct {
	// -1: 人員管理者,
	// 0: API,
	// 1: 一般使用者
	UserCategoryID int32
	UserID         string
	UserName       string
	PwHash         string
}
