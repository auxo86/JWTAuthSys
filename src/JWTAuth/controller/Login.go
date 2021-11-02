package controller

import (
	"JWTAuth/facilities"
	"JWTAuth/model"
	"encoding/json"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"net/http"
	"time"
)

// LoginHandler 處理登入資料
// 驗證帳密以後
// 產生一個 token 的 instance
// 然後開始塞客製化的 claims
// 裡面有標準的內容有 iat 和 exp
// 客製化的內容有
//		RowID: 使用者代碼（數字）
// 		JWTRegTimeInNanoSecWithCollisionCnt: JWT 登錄到 Redis 的時間 ( 使用 unix 奈秒時間戳 ),
//		AllowedClientIP: 送來的封包的 IP。 如果伺服程式位在 NAT 或是代理後面，就會沒法抓到正確的 IP，NAT 或是代理伺服器會改寫封包，所以會遺失原本的 IP
//		JWTRedisTTL: redis 中紀錄的有效時間（Time To Live, 可以自定義, 是 int64 nanosecond count 1 second = 1 000 000 000 nanoseconds）
// 然後加了 secret 就簽好了
// 感覺上沒有塞什麼東西
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var userInfo model.UsrAuthData
	var errGetIP error

	// 把 request 的 Body (JSON) parse 出來以後塞到 userInfo 中
	decodeErr := json.NewDecoder(r.Body).Decode(&userInfo)

	// 資料 parse 有問題就 return
	if decodeErr != nil {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "HTTP: Error while parsing request body.")
		return
	}

	// 取得 client 的 IP address ，如果取不到 userInfo.AllowedUserIP 就是 ""
	userInfo.AllowedUserIP, errGetIP = facilities.GetClientOnlyIP(r)
	if errGetIP != nil {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "權限問題：無法取得 IP 位置。")
		return
	}

	DBUserPwdHash, getDBPwdHashErr := facilities.GetUsrPwdHashFromDB(userInfo.UserID)

	// 無法正確取得密碼，正常密碼的 hash 應該不會是空白
	if getDBPwdHashErr != nil || DBUserPwdHash == "" {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "權限問題：使用者資料庫認證資料錯誤，不是正確的使用者。")
		return
	}

	strUserPwSaltedHash := facilities.CalHashOfSaltedPw(userInfo.Pw)

	// 密碼有問題也 return
	if strUserPwSaltedHash != DBUserPwdHash {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "權限問題：認證錯誤。")
		return
	}

	// 判斷是否為 API ( UserCategoryID == 0 )
	// 如果是，redisTTL 設定為 facilities.DefaultAPIRedisTTLHours, 也就是幾乎不過期
	if facilities.UserDb[userInfo.UserID].UserCategoryID == 0 {
		userInfo.JWTRedisTTL = facilities.DefaultAPIRedisTTLHours
	} else {
		if userInfo.JWTRedisTTL == 0 {
			// 如果本來沒有設定，就設定為 facilities.DefaultUsrRedisTTLHours
			userInfo.JWTRedisTTL = facilities.DefaultUsrRedisTTLHours
		}
	}
	// --------------- 底下是製造要回應的 token -----------------
	// 在這裡指定演算法並建立 token
	token := jwt.New(jwt.SigningMethodHS256)
	// 從這裡開始固定"現在時間"
	timeNow := time.Now()
	// Unix 奈秒時間戳 + 碰撞序號 (為了增加效率，算一次可以用很多地方，所以先算起來)
	sTimeNowWithCollisionCnt := fmt.Sprint(timeNow.UnixNano(), ".", facilities.CollisionCnt)
	// 在註冊的時候產生 jwt 回傳，
	// 裡面的 TTL 就已經根據是那一種身份(例如是不是 webapi)做了相對應的配置。
	response := facilities.GenerateCustomTokenClaimData(w, token, timeNow, sTimeNowWithCollisionCnt, userInfo)

	// 在 redis 的 session 紀錄前面加上 UserID
	_, redisRegErr := facilities.RegJwtOnRedis(
		fmt.Sprint(userInfo.UserID, ":", userInfo.AllowedUserIP),
		userInfo.JWTRedisTTL)

	// 如果 session 已經存在在這裡判斷
	// 表示應該 key 重複了 ( redisRegErr.Error() == "SessionExist" )
	// 依照宇翔的建議，如果已經有 session 就不重新註冊 session ，但是依然回傳新的 token
	// http status code 回傳為 201 (正常的 login 是回傳 200)
	if redisRegErr != nil && redisRegErr.Error() == "SessionExist" {
		// 注意，這裡回傳的是 201
		w.WriteHeader(http.StatusCreated)
		// 把結構經由序列化寫到 body 中。
		facilities.JsonResponse(response, w)
		return
	}
	// 表示真的出錯了
	if redisRegErr != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "session 問題："+redisRegErr.Error())
		return
	}

	// 把結構經由序列化寫到 body 中。
	facilities.JsonResponse(response, w)
}
