package controller

import (
	"JWTAuth/facilities"
	"JWTAuth/model"
	"encoding/json"
	"fmt"
	"net/http"
)

// handlerValidateJWT 驗證 token 的有效性，但是這個版本對 redis 做了讀寫分離。
// 讀成功以後把 session key 丟到 channelQueueUpdTTL 中去 update TTL
func HandlerValidateJWT(w http.ResponseWriter, r *http.Request) {
	var reqValidationData model.ValidationIPData
	// 先取出 token
	tokenWithClaims, errGetToken := facilities.GetTokenFromReq(r)
	if errGetToken != nil {
		w.WriteHeader(http.StatusUnauthorized)
		facilities.JsonResponse(model.ResponseValidationInfo{
			false,
			"Get Token error: " + errGetToken.Error(),
			""}, w)
		return
	}

	// 如果 tokenWithClaims 內容是空的也 return
	if tokenWithClaims == nil {
		w.WriteHeader(http.StatusUnauthorized)
		facilities.JsonResponse(model.ResponseValidationInfo{
			false,
			"JWT error: " + "tokenWithClaims 內容是空的。",
			""}, w)
		return
	}

	// 把 request 的 Body (JSON) parse 出來以後塞到 reqValidationData （呼叫的客戶端的來源地址） 中
	decodeErr := json.NewDecoder(r.Body).Decode(&reqValidationData)
	// 資料 parse 有問題就 return
	if decodeErr != nil {
		w.WriteHeader(http.StatusUnauthorized)
		facilities.JsonResponse(model.ResponseValidationInfo{
			false,
			"HTTP: 封包內容有誤，無法解譯出地址",
			""}, w)
		return
	}

	// 這一行是說，把 token 中的 CustomClaims 依照 model.CustomClaims 的結構 parse 成 customClaims
	customClaims := tokenWithClaims.Claims.(*model.CustomClaims)

	// 在 token 有效而且 來源 IP 正確的情況下去更新 Redis 存取時間戳
	if tokenWithClaims.Valid == true && customClaims.AllowedClientIP == reqValidationData.FromIP {
		sSessionKey := fmt.Sprint(customClaims.UserID, ":", customClaims.AllowedClientIP)
		// 從 redis 中讀取 key
		// 如果有 return http status code 200
		// 如果沒有就 return http.StatusUnauthorized
		errGetKey := facilities.GetSessionKey(sSessionKey)
		if errGetKey != nil {
			w.WriteHeader(http.StatusUnauthorized)
			facilities.JsonResponse(model.ResponseValidationInfo{
				false,
				fmt.Sprint("session 取得失敗：", errGetKey.Error(), ", 可能已過期，請嘗試重新登入。"),
				""}, w)
			return
		}

		// 把要更新 TTL 的 session key 值丟進去 ChannelQueueUpdTTL 慢慢更新
		facilities.ChannelQueueUpdTTL <- model.RedisKeyWithTTL{RedisKey: sSessionKey, RedisTTL: customClaims.AuthInfo.JWTRedisTTL}

		// 讀到 session key 以後 return http response
		w.WriteHeader(http.StatusOK)
		facilities.JsonResponse(model.ResponseValidationInfo{
			true,
			"Gained access to protected resource",
			customClaims.UserID}, w)
		return
	} else {
		w.WriteHeader(http.StatusUnauthorized)
		facilities.JsonResponse(model.ResponseValidationInfo{
			false,
			"權限問題：token 失效或是封包來源位置有誤",
			""}, w)
		return
	}
}
