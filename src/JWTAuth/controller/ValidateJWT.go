package controller

import (
	"JWTAuth/facilities"
	"JWTAuth/model"
	"fmt"
	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
)

// handlerValidateJWT 驗證 token 的有效性，但是這個版本對 redis 做了讀寫分離。
// 讀成功以後把 session key 丟到 channelQueueUpdTTL 中去 update TTL
func HandlerValidateJWT(c *fiber.Ctx) error {
	// 放 http request body 的內容，也就是 remote ip
	var reqValidationIpData model.ValidationIPData
	// 用來放 decode JWT 後產出的 custom claims
	var customClaims model.CustomClaims

	// 從 request 中 parse 出 bsTokenWithClaims
	errGetCustomClaims := facilities.GetCustomClaimsFromReq(c.Context(), &customClaims)

	// 設置 header 的 ContentType 為 "application/json; charset=utf-8"
	c.Response().Header.SetContentType("application/json; charset=utf-8")

	if errGetCustomClaims != nil {
		bsJson, _ := json.Marshal(model.ResponseValidationInfo{
			false,
			"token error: " + errGetCustomClaims.Error(),
			""})
		return c.Status(fiber.StatusUnauthorized).Send(bsJson)
	}

	// 把 request 的 Body (JSON) parse 出來以後塞到 reqValidationIpData （呼叫的客戶端的來源地址） 中
	if errDecode := c.BodyParser(&reqValidationIpData); errDecode != nil {
		bsJson, _ := json.Marshal(model.ResponseValidationInfo{
			false,
			"HTTP: 封包內容有誤，無法解譯出封包來源地址",
			""})

		return c.Status(fiber.StatusUnauthorized).Send(bsJson)
	}

	// 在 token 有效而且 來源 IP 正確的情況下去更新 Redis 存取時間戳
	if customClaims.AllowedClientIP == reqValidationIpData.FromIP {
		sSessionKey := fmt.Sprint(customClaims.UserID, ":", customClaims.AllowedClientIP)
		// 從 redis 中讀取 key
		// 如果有 return http status code 200
		// 如果沒有就 return http.StatusUnauthorized
		errGetKey := facilities.GetSessionKeyFromRedisCache(sSessionKey)
		if errGetKey != nil {
			bsJson, _ := json.Marshal(model.ResponseValidationInfo{
				false,
				fmt.Sprint("session 取得失敗：", errGetKey.Error(), ", 可能已過期，請嘗試重新登入。"),
				""})

			return c.Status(fiber.StatusUnauthorized).Send(bsJson)
		}

		// 把要更新 TTL 的 session key 值丟進去 ChannelQueueUpdTTL 慢慢更新
		facilities.MuSliceTTLExdCache.Lock()
		facilities.SliceTTLExdCache = append(facilities.SliceTTLExdCache, model.RedisKeyWithTTL{RedisKey: sSessionKey, RedisTTL: customClaims.AuthInfo.JWTRedisTTL})
		facilities.MuSliceTTLExdCache.Unlock()

		// 讀到 session key 以後 return http response
		baJson, _ := json.Marshal(model.ResponseValidationInfo{
			true,
			"Gained access to protected resource",
			customClaims.UserID})
		return c.Status(fiber.StatusOK).Send(baJson)
	} else {
		baJson, _ := json.Marshal(model.ResponseValidationInfo{
			false,
			"權限問題：token 失效或是封包來源位置有誤",
			""})
		return c.Status(fiber.StatusUnauthorized).Send(baJson)
	}
}
