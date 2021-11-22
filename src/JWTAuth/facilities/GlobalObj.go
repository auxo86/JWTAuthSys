package facilities

import (
	"JWTAuth/model"
	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/joho/godotenv"
	"log"
	"strconv"
	"time"
)

var JWTSecretKey string
var StrSalt string
var DefaultUserJwtExpireDuration time.Duration
var DefaultAPIJwtExpireDuration time.Duration
var DefaultAPIRedisTTLHours time.Duration
var DefaultUsrRedisTTLHours time.Duration
var ChannelQueueUpdTTL chan model.RedisKeyWithTTL

// 全域連線池，有兩個，GlobalQryPool 和 GlobalOpPool
var GlobalQryPool *pgxpool.Pool
var GlobalOpPool *pgxpool.Pool

// 要記得 JWTAuth 結束之前要關閉
var RedisClientWriteOpr *redis.Client
var RedisClientReadOpr *redis.Client

// UserDb 是使用者資料庫 mapping 的 map，裡面有帳號和 salted passwd hash
// 型態 uint64 的 key 是使用者表 (userauth.users.usersecret) 的 id (其實就是 RowID。)
// UserID 可能是 API 或是人類
var UserDb = map[string]model.DBUserCredentials{}

// 全域環境變數設定
var MyEnv map[string]string
var errEnv error

func init() {
	MyEnv, errEnv = godotenv.Read()
	if errEnv != nil {
		log.Fatal("無法載入 .env 檔。")
	}

	// Jwt TTL 設定
	apiJwtExpireDurationHours, _ := strconv.Atoi(MyEnv["APIJwtExpireDurationHours"])
	usrJwtExpireDurationHours, _ := strconv.Atoi(MyEnv["UsrJwtExpireDurationHours"])
	// redis session TTL 設定
	defaultAPIRedisTTLHours, _ := strconv.Atoi(MyEnv["DefaultRedisAPITTLHours"])
	defaultUsrRedisTTLHours, _ := strconv.Atoi(MyEnv["DefaultRedisUsrTTLHours"])

	// 頒發和驗證 JWT 用的
	JWTSecretKey = MyEnv["JWTSecretKey"]
	StrSalt = MyEnv["StrSalt"]
	DefaultAPIJwtExpireDuration = time.Hour * time.Duration(apiJwtExpireDurationHours)
	DefaultUserJwtExpireDuration = time.Hour * time.Duration(usrJwtExpireDurationHours)
	DefaultAPIRedisTTLHours = time.Hour * time.Duration(defaultAPIRedisTTLHours)
	DefaultUsrRedisTTLHours = time.Hour * time.Duration(defaultUsrRedisTTLHours)

	// channelQueueUpdTTL 相關設定
	ChannelQueueUpdTTL = make(chan model.RedisKeyWithTTL)
}

// 因為在 high concurrency 情境下就算使用奈秒仍然會發生 key 值重複的問題。
// 所以原本的 key 值的登入時間奈秒後面加上一個碰撞序號
// 只要碰撞一次，這個值就加 1
// 所以現在的 key 值格式是這樣的：
// [使用者帳號]:[註冊時間 unix 奈秒時間戳].[碰撞序號][jwt]
// (請省略左右中括號)
var CollisionCnt int64 = 0
