package facilities

import (
	"JWTAuth/model"
	"github.com/cristalhq/jwt/v5"
	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/joho/godotenv"
	"log"
	"regexp"
	"strconv"
	"sync"
	"time"
)

var JWTSecretKey []byte
var JWTverifier *jwt.HSAlg
var JWTsigner *jwt.HSAlg
var JWTbuilder *jwt.Builder

var DefaultUserJwtExpireDuration time.Duration
var DefaultAPIJwtExpireDuration time.Duration
var DefaultAPIRedisTTLHours time.Duration
var DefaultUsrRedisTTLHours time.Duration

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

// DictAllSessionKeysOnRedis 是 redis 中 user session keys 的暫存字典
var DictAllSessionKeysOnRedis = make(map[string]int)
var DefaultUpdAllKindCacheSecs time.Duration

// SliceTTLExdCache 是用來收集要丟進 RedisTTLExd() 的物件 slice 的指標
// 但是應用上會先去重複以後再丟進去
var SliceTTLExdCache = make([]model.RedisKeyWithTTL, 0, 500000)

var SlPool = sync.Pool{
	New: func() interface{} {
		return make([]model.RedisKeyWithTTL, 0, 500000)
	},
}
var MuSliceTTLExdCache sync.Mutex
var ChBatchUpdTTLTimeout = make(chan bool)
var ChSlUpdTTLFull = make(chan bool)

// 全域環境變數設定
var MyEnv map[string]string
var errEnv error

// CollisionCnt 因為在 high concurrency 情境下就算使用奈秒仍然會發生 key 值重複的問題。
// 所以原本的 key 值的登入時間奈秒後面加上一個碰撞序號
// 只要碰撞一次，這個值就加 1
// 所以現在的 key 值格式是這樣的：
// [使用者帳號]:[註冊時間 unix 奈秒時間戳].[碰撞序號][jwt]
// (請省略左右中括號)
var CollisionCnt int64 = 0

// program facilities
// 做一個 regex 的置換 pattern ，只要是符合 "bearer "，不論大小寫，一律置換成空字串
var RegexGetTokenStr *regexp.Regexp

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

	// 從 .env 讀取預設更新 DictAllSessionKeysOnRedis 的秒數
	iUpdAllKindCacheSecs, _ := strconv.Atoi(MyEnv["DefaultUpdAllKindCacheSecs"])

	// 頒發和驗證 JWT 用的
	JWTSecretKey = []byte(MyEnv["JWTSecretKey"])
	JWTverifier, _ = jwt.NewVerifierHS(jwt.HS256, JWTSecretKey) // 在這裡指定演算法
	JWTsigner, _ = jwt.NewSignerHS(jwt.HS256, JWTSecretKey)
	JWTbuilder = jwt.NewBuilder(JWTsigner)

	DefaultAPIJwtExpireDuration = time.Hour * time.Duration(apiJwtExpireDurationHours)
	DefaultUserJwtExpireDuration = time.Hour * time.Duration(usrJwtExpireDurationHours)
	DefaultAPIRedisTTLHours = time.Hour * time.Duration(defaultAPIRedisTTLHours)
	DefaultUsrRedisTTLHours = time.Hour * time.Duration(defaultUsrRedisTTLHours)

	RegexGetTokenStr = regexp.MustCompile(`(?i)bearer `)

	// 預設更新 DictAllSessionKeysOnRedis 的秒數
	DefaultUpdAllKindCacheSecs = time.Second * time.Duration(iUpdAllKindCacheSecs)
}
