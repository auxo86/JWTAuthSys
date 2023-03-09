package controller

import (
	"JWTAuth/facilities"
	"JWTAuth/model"
	"fmt"
	"github.com/cristalhq/jwt/v5"
	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
	"time"
)

// LoginHandler 處理登入資料
// 驗證帳密以後
// 產生一個 token 的 instance
// 然後開始塞客製化的 claims
// 裡面有標準的內容有 iat 和 exp
// 客製化的內容有
//
//	RowID: 使用者代碼（數字）
//	JWTRegTimeInNanoSecWithCollisionCnt: JWT 登錄到 Redis 的時間 ( 使用 unix 奈秒時間戳 ),
//	AllowedClientIP: 送來的封包的 IP。 如果伺服程式位在 NAT 或是代理後面，就會沒法抓到正確的 IP，NAT 或是代理伺服器會改寫封包，所以會遺失原本的 IP
//	JWTRedisTTL: redis 中紀錄的有效時間（Time To Live, 可以自定義, 是 int64 nanosecond count 1 second = 1 000 000 000 nanoseconds）
//
// 然後加了 secret 就簽好了
// 感覺上沒有塞什麼東西
func LoginHandler(c *fiber.Ctx) error {
	userInfo := new(model.UsrAuthData)
	var errGetIP error

	// 把 request 的 Body (JSON) parse 出來以後塞到 userInfo 中
	// 同時資料 parse 有問題就 return
	if decodeErr := c.BodyParser(userInfo); decodeErr != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "HTTP: Error while parsing request body. "+decodeErr.Error())
	}

	// 取得 client 的 IP address ，如果取不到 userInfo.AllowedUserIP 就是 ""
	userInfo.AllowedUserIP, errGetIP = facilities.GetClientOnlyIP(c.Context())
	if errGetIP != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "權限問題：無法取得 IP 位置。 "+errGetIP.Error())
	}

	DBUserPwdHash, errGetDBPwdHash := facilities.GetUsrPwdHashFromDB(userInfo.UserID)

	// 無法正確取得密碼，正常密碼的 hash 應該不會是空白
	if errGetDBPwdHash != nil || DBUserPwdHash == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "權限問題：使用者資料庫認證資料錯誤，不是正確的使用者。")
	}

	// 改進加密算法的演算機制為 bcrypt
	// 密碼有問題就 return
	errChkPass := bcrypt.CompareHashAndPassword([]byte(DBUserPwdHash), []byte(userInfo.Pw))

	if errChkPass != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "權限問題：認證錯誤。 ")
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
	// 建立 token
	var token jwt.Token
	// 從這裡開始固定"現在時間"
	timeNow := time.Now()
	// Unix 奈秒時間戳 + 碰撞序號 (為了增加效率，算一次可以用很多地方，所以先算起來)
	sTimeNowWithCollisionCnt := fmt.Sprint(timeNow.UnixNano(), ".", facilities.CollisionCnt)
	// 在註冊的時候產生 jwt 回傳，
	// 裡面的 TTL 就已經根據是那一種身份(例如是不是 webapi)做了相對應的配置。
	responseToken, errGenCustomTokenClaimData := facilities.GenerateCustomTokenClaimData(&token, timeNow, sTimeNowWithCollisionCnt, *userInfo)

	if errGenCustomTokenClaimData != nil {
		return fiber.NewError(fiber.StatusInternalServerError, errGenCustomTokenClaimData.Error())
	}

	// 在 redis 的 session 紀錄前面加上 UserID
	_, redisRegErr := facilities.RegSessionOnRedis(
		fmt.Sprint(userInfo.UserID, ":", userInfo.AllowedUserIP),
		userInfo.JWTRedisTTL)

	// 把要回傳的內容寫到 http responseToken body 中。
	bsToken, errToJson := json.Marshal(responseToken)
	if errToJson != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "JSON 相關：無法成功轉換 JSON, "+errToJson.Error())
	}

	// 設置 header 的 ContentType 為 "application/json; charset=utf-8"
	c.Response().Header.SetContentType("application/json; charset=utf-8")

	// 如果 session 已經存在在這裡判斷
	// 表示應該 key 重複了 ( redisRegErr.Error() == "SessionExist" )
	// 依照宇翔的建議，如果已經有 session 就不重新註冊 session ，但是依然回傳新的 token
	// http status code 回傳為 201 (正常的 login 是回傳 200)
	if redisRegErr != nil && redisRegErr.Error() == "SessionExist" {
		// 注意，這裡回傳的是 201
		return c.Status(fiber.StatusCreated).Send(bsToken)
	}
	// 表示真的出錯了
	if redisRegErr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "session 問題："+redisRegErr.Error())
	}

	// 更新 DictAllSessionKeysOnRedis ( redis session keys cache )
	go facilities.UpdDictSessionKeys(false)

	return c.Status(fiber.StatusOK).Send(bsToken)
}
