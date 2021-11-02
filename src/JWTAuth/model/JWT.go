package model

import (
	"github.com/dgrijalva/jwt-go"
	"time"
)

// AuthInfo 紀錄額外的認證資訊
type AuthInfo struct {
	UserID                              string
	JWTRegTimeInNanoSecWithCollisionCnt string
	AllowedClientIP                     string
	JWTRedisTTL                         time.Duration
}

// CustomClaims 客製化的 Claims
type CustomClaims struct {
	*jwt.StandardClaims
	TokenType string
	*AuthInfo
}
