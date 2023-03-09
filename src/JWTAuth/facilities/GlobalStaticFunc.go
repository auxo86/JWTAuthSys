package facilities

import (
	"JWTAuth/model"
	"context"
	"errors"
	"fmt"
	"github.com/cristalhq/jwt/v5"
	"github.com/valyala/fasthttp"
	"golang.org/x/crypto/bcrypt"
	"net"
	"strings"
	"time"
)

func init() {
	// 每隔一段時間就批次更新 redis key TTL 並且同時也檢查 SliceTTLExdCache 是不是塞了太多 items
	// 如果塞太多當然要送更新，但是如果太久沒人塞，一段時間也應該送更新
	go PolicyUpdRedisKeyTTL()
	// 每隔一段時間就送出批次更新 Redis key TTL 的定時器訊號
	go SendUpdTTLTimeoutSignal()
	// 每隔一段時間就檢查 SliceTTLExdCache 的 length ，避免塞的太多
	go CheckSlTTLExdCacheLen()
}

// RegSessionOnRedis 用於在 login 時向 Redis 註冊 JWT，回傳 boolStatus 和 err
// 成功：如果 JWT 不存在則 SETNX 會成功
// 失敗：如果 JWT 已存在會失敗
// 命令在設置成功時返回 true ， 設置失敗時返回 false。
// 如果沒有設定自訂的 Redis TTL ，會帶預設值一小時
// key 的 value 為"存取次數"，當然第一次 login 存取次數必然是 1
func RegSessionOnRedis(sKey string, redisTTL time.Duration) (string, error) {
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

//func GetSessionKeyFromRedis(sSessionKey string) error {
//	// go-redis 需要的 context 設定
//	ctxbg := context.Background()
//	_, errGetSessionKey := RedisClientReadOpr.Get(ctxbg, sSessionKey).Result()
//	if errGetSessionKey != nil {
//		return errGetSessionKey
//	}
//	return nil
//}

func GetSessionKeyFromRedisCache(sSessionKey string) error {
	_, boolIfKeyExist := DictAllSessionKeysOnRedis[sSessionKey]
	if boolIfKeyExist != true {
		return errors.New("session: session not exist")
	}
	return nil
}

/*
PolicyUpdRedisKeyTTL ：本函數的邏輯如下：
1. 循環執行以下操作，每次等待固定時間(DefaultUpdAllKindCacheSecs)後再執行下一次循環。
另外每隔一小段時間就檢查一下 SliceTTLExdCache 是否已經填充了太多資料，如果已經太多就先更新。

2. 檢查是否有需要更新 TTL 的 Redis key，
如果有，就加鎖，並將 SliceTTLExdCache 中的內容複製到新的指標 sliceTmpTTLExdCache 中，
然後重置 SliceTTLExdCache 以便接收新資料。如果沒有需要更新 TTL 的 Redis key，就跳過當前循環。

3. 對 sliceTmpTTLExdCache 進行去重複，然後將其傳入 UpdRedisSessionTTL 函數中並作為 goroutine 執行，
透過 RedisTTLExd() 調用 RedisClientWriteOpr.Expire() 函數來更新 Redis key 的 TTL。
無法成功更新 TTL 時，印出錯誤信息。真正的更新 redis session key 的 TTL 。
*/
func PolicyUpdRedisKeyTTL() {
	for {
		select {
		case <-ChBatchUpdTTLTimeout:
			go BatchUpdRedisTTL()
		case <-ChSlUpdTTLFull:
			go BatchUpdRedisTTL()
		}
	}
}

func SendUpdTTLTimeoutSignal() {
	ticker := time.NewTicker(DefaultUpdAllKindCacheSecs)
	for range ticker.C {
		ChBatchUpdTTLTimeout <- true
	}
}

func CheckSlTTLExdCacheLen() {
	ticker := time.NewTicker(DefaultUpdAllKindCacheSecs / 3)
	for range ticker.C {
		MuSliceTTLExdCache.Lock()
		if len(SliceTTLExdCache) >= 10000 {
			ChSlUpdTTLFull <- true
		}
		MuSliceTTLExdCache.Unlock()
	}
}

func BatchUpdRedisTTL() {
	if SliceTTLExdCache != nil && len(SliceTTLExdCache) != 0 {
		MuSliceTTLExdCache.Lock()
		// 把 SliceTTLExdCache 的內容指派給新的指針 sliceTmpTTLExdCache
		sliceTmpTTLExdCache := SlPool.Get().([]model.RedisKeyWithTTL)
		sliceTmpTTLExdCache = append(sliceTmpTTLExdCache[:0], SliceTTLExdCache...)
		// 重置 SliceTTLExdCache 的內容，以便接收新資料
		SliceTTLExdCache = SliceTTLExdCache[:0]
		MuSliceTTLExdCache.Unlock()
		// 把 sliceTmpTTLExdCache 拿來去重複
		sliceTmpPassToUpdFunc := DistinctSlice(&sliceTmpTTLExdCache)
		SlPool.Put(sliceTmpTTLExdCache)

		go UpdRedisSessionTTL(&sliceTmpPassToUpdFunc)
	} else {
		return
	}
}

// DistinctSlice 去重複 SliceTTLExdCache 的副本
func DistinctSlice(sliceIn *[]model.RedisKeyWithTTL) []model.RedisKeyWithTTL {
	mapExistItems := make(map[model.RedisKeyWithTTL]bool)
	var sliceOut []model.RedisKeyWithTTL

	if *sliceIn != nil && len(*sliceIn) != 0 {
		for _, itemKeyWithTTL := range *sliceIn {
			if _, ifExist := mapExistItems[itemKeyWithTTL]; !ifExist {
				sliceOut = append(sliceOut, itemKeyWithTTL)
				mapExistItems[itemKeyWithTTL] = true
			}
		}
	}

	return sliceOut
}

func UpdRedisSessionTTL(sliceIn *[]model.RedisKeyWithTTL) {
	if sliceIn == nil || len(*sliceIn) == 0 {
		return
	}
	// 把 去重複以後的 SliceTTLExdCache 那來餵進 RedisTTLExd()
	for _, itemKeyWithTTL := range *sliceIn {
		// 在這裡延長 Redis Session TTL
		RedisTTLExd(itemKeyWithTTL)
	}
}

// RedisTTLExd 真正用於延長 redis 上 session ttl 的函數
func RedisTTLExd(instSessionKeyWithTTL model.RedisKeyWithTTL) {
	// go-redis 需要的 context 設定
	ctxbg := context.Background()
	// 所有的人都必須更新 redis key 值的 TTL，即使是 TTL 為 -1 的 webapi，因為這樣我才能停用這個 JWT
	bExdTTL := RedisClientWriteOpr.Expire(ctxbg, instSessionKeyWithTTL.RedisKey, instSessionKeyWithTTL.RedisTTL).Val()
	if !bExdTTL {
		fmt.Println(fmt.Sprint(instSessionKeyWithTTL.RedisKey, " 無法展延 session 的 ttl，可能是伺服器主節點故障或是無 session 。"))
	}
}

// GetUsrPwdHashFromDB 傳入使用者名稱得到資料庫中的 hash (salted)
func GetUsrPwdHashFromDB(sUser string) (string, error) {
	strPassHash := UserDb[sUser].PwHash
	if strPassHash == "" {
		return "", errors.New("權限問題：使用者認證失敗")
	}
	return strPassHash, nil
}

// CalHashOfSaltedPw 傳入密碼然後算出 bcrypt salted passwd hash
func CalHashOfSaltedPw(strPass string) ([]byte, error) {
	byteArrayPass := []byte(strPass)

	// 隨機生成 salt 並且使用 default cost 進行 byteArrayHash
	byteArrayHash, errGetHash := bcrypt.GenerateFromPassword(byteArrayPass, bcrypt.DefaultCost)
	if errGetHash != nil {
		return nil, errGetHash
	}

	return byteArrayHash, nil
}

//// JsonResponseToByteSlice 用於將物件轉換為 byte slice
//func JsonResponseToByteSlice(response interface{}) ([]byte, error) {
//	byteSliceJSON, err := json.Marshal(response)
//	if err != nil {
//		return nil, err
//	}
//
//	return byteSliceJSON, nil
//}

func GetCustomClaimsFromReq(ctxR *fasthttp.RequestCtx, customClaims *model.CustomClaims) error {
	strBearerToken := string(ctxR.Request.Header.Peek("Authorization"))
	// 如果取不到 Bearer Token 就回應錯誤。
	if strBearerToken == "" {
		return errors.New("權限問題：Error in extracting the token")
	}

	// 把 "(?i)bearer " 置換為空白，並且轉成 []byte
	bsToken := []byte(RegexGetTokenStr.ReplaceAllString(strBearerToken, ""))

	// 下面這句是說
	// 得到的 bsToken 用 jwt.Parse() 搭配 JWTverifier 解密 parse 成帶有 claims (type 是 json.RawMessage)這種資料結構的 jwt.Token
	// 在這個階段就做過驗證 jwt 了
	bsTokenWithClaims, errParseWithClaims := jwt.Parse(bsToken, JWTverifier)

	if errParseWithClaims != nil {
		return errors.New("token 問題： " + "error occurred while parse JWT, " + errParseWithClaims.Error())
	}

	if bsTokenWithClaims == nil {
		return errors.New("token 問題： " + "bsTokenWithClaims is nil")
	}

	// 從 bsTokenWithClaims 中 取出 customClaims
	// 這一行是說，把 token 中的 CustomClaims 依照 model.CustomClaims 的結構 parse 成 customClaims
	errDecodeCustomClaims := bsTokenWithClaims.DecodeClaims(&customClaims)

	if !customClaims.IsValidAt(time.Now()) {
		return errors.New("token 問題： " + "token was expired")
	}

	if customClaims == nil {
		return errors.New("token 問題： " + "custom claims is nil")
	}

	if errDecodeCustomClaims != nil {
		return errors.New("token 問題： " + errDecodeCustomClaims.Error())
	}

	return nil
}

// login 時用來產生 return 的 token
func GenerateCustomTokenClaimData(token *jwt.Token, timeNow time.Time, sTimeNowWithCollisionCnt string, userInfo model.UsrAuthData) (*model.Token, error) {
	dbUserCategory := UserDb[userInfo.UserID].UserCategoryID
	var timeExpireDuration time.Duration
	if dbUserCategory == 0 {
		timeExpireDuration = DefaultAPIJwtExpireDuration
	} else {
		timeExpireDuration = DefaultUserJwtExpireDuration
	}

	// 產生 JWT 中定義的資料結構的 instance
	claims := &model.CustomClaims{
		// 其實在 claim 中有註冊可以使用的 key 有底下這些：
		// iss(Issuer)：頒發者，是區分大小寫的字串，可以是一個字串或是網址
		// sub(Subject)：主體內容，是區分大小寫的字串，可以是一個字串或是網址
		// aud(Audience)：受眾，是區分大小寫的字串，可以是一個字串或是網址
		// exp(Expiration Time)：Expiration Time，過期時間，是數字，使用 unix 時間戳的格式，不可以是奈秒
		// nbf(Not Before)：定義在什麼時間之前，不可用，是數字日期
		// iat(Issued At)：頒發時間，是數字，使用 unix 時間戳的格式
		// jti(JWT ID)：唯一識別碼，是區分大小寫的字串，不可以是奈秒
		// 如果結構中這裡是個指標，那麼就要給 & (ref)
		RegisteredClaims: &jwt.RegisteredClaims{
			ExpiresAt: &jwt.NumericDate{Time: timeNow.Add(timeExpireDuration)}, // 指定過期時間，使用 unix 時間戳的格式
			IssuedAt:  &jwt.NumericDate{Time: timeNow},                         // 定義從什麼時候開始算
		},
		TokenType: "level1",
		AuthInfo: &model.AuthInfo{
			UserID:                              userInfo.UserID,
			JWTRegTimeInNanoSecWithCollisionCnt: sTimeNowWithCollisionCnt,
			// 如果伺服程式位在 NAT 或是代理後面，就會沒法抓到正確的 IP
			// NAT 或是代理伺服器會改寫封包，所以會遺失原本的 IP
			AllowedClientIP: userInfo.AllowedUserIP,
			JWTRedisTTL:     userInfo.JWTRedisTTL,
		},
	}

	token, errBuildToken := JWTbuilder.Build(claims)
	if errBuildToken != nil {
		return nil, errors.New("伺服器錯誤：Error while building signed token")
	}

	// 這裡建立 token 的 string
	response := model.Token{Token: token.String()}
	return &response, nil
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

// GetClientOnlyIP 回傳 client 的 ip
// 如果 X-Forwarded-For 有 ip 可以抓就優先抓這個，否則抓 ctxR.RemoteIP()
func GetClientOnlyIP(ctxR *fasthttp.RequestCtx) (string, error) {
	sIP := ""
	sIPList := string(ctxR.Request.Header.Peek("X-Forwarded-For"))

	if sIPList != "" {
		sIP = strings.Split(sIPList, ", ")[0]
	} else {
		sIP = ctxR.RemoteIP().String()
		if sIP == net.IPv4zero.String() {
			return "", errors.New("network error: can't get remote IP")
		}
	}
	return sIP, nil
}

func UpdDictSessionKeys(boolIfLoop bool) {
	for {
		// 做一個暫存的字典
		tmpDictSessions, errGetAllKeys := getAllSessionKeysOnRedis()
		if errGetAllKeys != nil {
			fmt.Println("Redis error: " + errGetAllKeys.Error())
			time.Sleep(DefaultUpdAllKindCacheSecs)
			continue
		}
		// 把 DictAllSessionKeysOnRedis 用暫存字典瞬間換過來
		DictAllSessionKeysOnRedis = tmpDictSessions
		if boolIfLoop == false {
			break
		}
		time.Sleep(DefaultUpdAllKindCacheSecs)
	}
}

func getAllSessionKeysOnRedis() (map[string]int, error) {
	tmpDictSessions := make(map[string]int)
	// 透過 SCAN 指令遍歷 redis 中所有的 session keys
	iterRedisDb := RedisClientReadOpr.Scan(RedisClientReadOpr.Context(), 0, "*", 0).Iterator()
	for iterRedisDb.Next(RedisClientReadOpr.Context()) {
		// 取得 key value pair
		key := iterRedisDb.Val()
		// 向 tmpDictSessions 添加新的 key value pair
		tmpDictSessions[key] = 0
	}

	if errIterRedisDb := iterRedisDb.Err(); errIterRedisDb != nil {
		return nil, errIterRedisDb
	}

	return tmpDictSessions, nil
}
