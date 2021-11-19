package facilities

import (
	"JWTAuth/model"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

func init() {
	go UpdRedisKeyTTLNoReturn(ChannelQueueUpdTTL)
}

// RegJwtOnRedis 用於在 login 時向 Redis 註冊 JWT，回傳 boolStatus 和 err
// 成功：如果 JWT 不存在則 SETNX 會成功
// 失敗：如果 JWT 已存在會失敗
// 命令在設置成功時返回 true ， 設置失敗時返回 false。
// 如果沒有設定自訂的 Redis TTL ，會帶預設值一小時
// key 的 value 為"存取次數"，當然第一次 login 存取次數必然是 1
func RegJwtOnRedis(sKey string, redisTTL time.Duration) (string, error) {
	ctxbg := context.Background()
	// redis 的回應訊息
	sStatus := ""
	// 本函數執行要回應的錯誤訊息
	errSetSession := errors.New("")

	sStatus, errSetSession = RedisClientWriteOpr.Do(ctxbg, "SET", sKey, 1, "EX", redisTTL.Seconds(), "NX").Text()

	// 先判斷是不是有網路斷線的錯誤
	if errSetSession != nil && strings.Contains(errSetSession.Error(), "connection refused") {
		return sStatus, errors.New("connection refused, 無法連線到 session server")
	}
	// 注意，redis.Nil 也不是 nil ，表示註冊了相同的 key
	if errSetSession != nil {
		return sStatus, errors.New("SessionExist")
	}
	// 真的成功要回傳 "OK", nil
	return sStatus, nil
}

func GetSessionKey(sSessionKey string) error {
	// go-redis 需要的 context 設定
	ctxbg := context.Background()
	_, errGetSessionKey := RedisClientReadOpr.Get(ctxbg, sSessionKey).Result()
	if errGetSessionKey != nil {
		return errGetSessionKey
	}
	return nil
}

// 執行延長 session TTL 做驗證的動作，但是這個版本沒有 return
// 可以利用剩餘的 TTL 判斷最近存取時間
func UpdRedisKeyTTLNoReturn(chanQueueUpdTTL chan model.RedisKeyWithTTL) {
	for {
		instSessionKeyWithTTL := <-chanQueueUpdTTL
		// go-redis 需要的 context 設定
		ctxbg := context.Background()
		// 所有的人都必須更新 redis key 值的 TTL，即使是 TTL 為 -1 的 webapi，因為這樣我才能停用這個 JWT
		bExdTTL := RedisClientWriteOpr.Expire(ctxbg, instSessionKeyWithTTL.RedisKey, instSessionKeyWithTTL.RedisTTL).Val()
		if bExdTTL == false {
			fmt.Println(fmt.Sprint(instSessionKeyWithTTL.RedisKey, " 無法展延 session 的 ttl，可能是伺服器主節點故障或是無 session 。"))
		}
	}
}

// 真正的錯誤，透過這個回應到 console 上
func Fatal(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func SeprateOnlyIPAddr(strIPWithPort string) (string, error) {
	var strIP string
	strSliceIPAddr := strings.Split(strIPWithPort, ":")
	if len(strSliceIPAddr) < 2 {
		return "", errors.New("無法取得 ip")
	}
	strIP = strSliceIPAddr[0]
	return strIP, nil
}

// GetUsrPwdHashFromDB 傳入使用者名稱得到資料庫中的 hash (salted)
func GetUsrPwdHashFromDB(sUser string) (string, error) {
	strPassHash := UserDb[sUser].PwHash
	if strPassHash == "" {
		return "", errors.New("權限問題：使用者認證失敗")
	}
	return strPassHash, nil
}

// CalHashOfSaltedPw 傳入密碼然後算出 SHA256 salted passwd hash
func CalHashOfSaltedPw(strPass string) string {
	h := sha256.New()
	pwWithSalt := strPass + StrSalt
	// Write (via the embedded io.Writer interface) adds more data to the running hash.
	h.Write([]byte(pwWithSalt))
	byteSliceHash := h.Sum(nil)
	return hex.EncodeToString(byteSliceHash)
}

// CorsHandler 處理 Method == "OPTIONS" 的 preflight request
// 如果不是 preflight request 就轉發到對應處理的 httpHandler
func CorsHandler(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 下面這兩行檔頭一定要設定，不然會失敗。
		// 檔頭要設定什麼是根據 主request 內容決定的
		w.Header().Set("Access-Control-Allow-Origin", "*")
		// 因為我要使用 JWT ，所以 header 必須允許 Authorization
		// 如果 Access-Control-Allow-Headers 有多個值，用 , 隔開就好。
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		// 如果是 preflight request ，直接 return 就好。
		if r.Method == "OPTIONS" {
			return
		}
		// 其實下面這行就算沒有設一樣 post 會成功。
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, DELETE, PUT")
		// 如果不是就轉發到 相對應的 httpHandler
		h.ServeHTTP(w, r)
	}
}

func JsonResponse(response interface{}, w http.ResponseWriter) {
	byteSliceJSON, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(byteSliceJSON)
}

func GetTokenFromReq(r *http.Request) (*jwt.Token, error) {
	// 做一個 regex 的置換 pattern ，只要是符合 "bearer "，不論大小寫，一律置換成空字串
	regexGetTokenStr := regexp.MustCompile(`(?i)bearer `)
	strBearerToken := r.Header["Authorization"][0]
	// 如果取不到 Bearer Token 就回應錯誤。
	if strBearerToken == "" {
		return nil, errors.New("權限問題：Error in extracting the token")
	}

	// 把 "(?i)bearer " 置換為空白
	strToken := regexGetTokenStr.ReplaceAllString(strBearerToken, "")
	// 下面這句是說
	// 得到的 strToken 用 jwt.ParseWithClaims() 中的 KeyFunc() 解密 parse 成帶有 CustomClaims 這種資料結構的 jwt.Token
	tokenWithClaims, errParseWithClaims := jwt.ParseWithClaims(strToken, &model.CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(JWTSecretKey), nil
	})

	if errParseWithClaims != nil || tokenWithClaims == nil {
		if errParseWithClaims == nil {
			errParseWithClaims = errors.New("tokenWithClaims is nil")
		}
		return nil, errors.New("token 問題：" + "Error when ParseWithClaims, " + errParseWithClaims.Error())
	}

	return tokenWithClaims, nil
}

func GenerateCustomTokenClaimData(w http.ResponseWriter, token *jwt.Token, timeNow time.Time, sTimeNowWithCollisionCnt string, userInfo model.UsrAuthData) model.Token {
	dbUserCategory := UserDb[userInfo.UserID].UserCategoryID
	var timeExpireDuration time.Duration
	if dbUserCategory == 0 {
		timeExpireDuration = DefaultAPIJwtExpireDuration
	} else {
		timeExpireDuration = DefaultUserJwtExpireDuration
	}
	// 產生 JWT 中定義的資料結構的 instance
	token.Claims = &model.CustomClaims{
		// 其實在 claim 中有註冊可以使用的 key 有底下這些：
		// iss(Issuer)：頒發者，是區分大小寫的字串，可以是一個字串或是網址
		// sub(Subject)：主體內容，是區分大小寫的字串，可以是一個字串或是網址
		// aud(Audience)：受眾，是區分大小寫的字串，可以是一個字串或是網址
		// exp(Expiration Time)：Expiration Time，過期時間，是數字，使用 unix 時間戳的格式，不可以是奈秒
		// nbf(Not Before)：定義在什麼時間之前，不可用，是數字日期
		// iat(Issued At)：頒發時間，是數字，使用 unix 時間戳的格式
		// jti(JWT ID)：唯一識別碼，是區分大小寫的字串，不可以是奈秒
		// 如果結構中這裡是個指標，那麼就要給 & (ref)
		&jwt.StandardClaims{
			ExpiresAt: timeNow.Add(timeExpireDuration).Unix(), // 指定過期時間，使用 unix 時間戳的格式
			IssuedAt:  timeNow.Unix(),                         // 定義從什麼時候開始算
		},
		"level1",
		&model.AuthInfo{
			UserID:                              userInfo.UserID,
			JWTRegTimeInNanoSecWithCollisionCnt: sTimeNowWithCollisionCnt,
			// 如果伺服程式位在 NAT 或是代理後面，就會沒法抓到正確的 IP
			// NAT 或是代理伺服器會改寫封包，所以會遺失原本的 IP
			AllowedClientIP: userInfo.AllowedUserIP,
			JWTRedisTTL:     userInfo.JWTRedisTTL,
		},
	}

	// 密鑰看起來不能使用字串，要轉換成 []byte
	// 這裡建立 token 的 string
	tokenString, err := token.SignedString([]byte(JWTSecretKey))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "伺服器錯誤：Error while signing the token")
		Fatal(err)
	}

	// 把 rawstring 變成 "結構"
	response := model.Token{tokenString}
	return response
}

// 刪除 redis session ( key )
func UnLinkKey(sKey string) error {
	ctxbg := context.Background()
	i64IfUnlinkOK := RedisClientWriteOpr.Unlink(ctxbg, sKey).Val()
	if i64IfUnlinkOK == 1 {
		return nil
	} else {
		return errors.New("session Key 不存在。")
	}
}

// 回傳 client 的 ip
// 如果 X-Forwarded-For 有 ip 可以抓就優先抓這個，否則抓 r.RemoteAddr
func GetClientOnlyIP(r *http.Request) (string, error) {
	sIP := ""
	sIPSeq := r.Header.Get("X-Forwarded-For")
	errGetIp := errors.New("")

	if sIPSeq != "" {
		sIP = strings.Split(sIPSeq, ", ")[0]
	} else {
		sIP, errGetIp = SeprateOnlyIPAddr(r.RemoteAddr)
		if errGetIp != nil {
			return "", errGetIp
		}
	}
	return sIP, nil
}
