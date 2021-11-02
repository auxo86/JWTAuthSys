package controller

import (
	"JWTAuth/facilities"
	"JWTAuth/model"
	"encoding/json"
	"fmt"
	"net/http"
)

// 使用者用來遠端刪除 redis 上的 session
// 一般使用者和 API 使用者不同
// 一般使用者只要 token 有效而且封包 IP 符合 token 的 IP 就可以刪除 redis 上的 session
// API 使用者為求慎重，要再加上帳號密碼確認才可以刪除
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	// 取得封包的 IP address
	packetIP, errGetIP := facilities.SeprateOnlyIPAddr(r.RemoteAddr)
	if errGetIP != nil {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "權限問題：無法取得 IP 位置。")
		return
	}

	// 從 request 中 parse 出 tokenWithClaims
	tokenWithClaims, errGetToken := facilities.GetTokenFromReq(r)
	if errGetToken != nil {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "Get Token error: "+errGetToken.Error())
		return
	}
	if tokenWithClaims == nil {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "JWT error: "+"tokenWithClaims 內容是空的。")
		return
	}
	// 從 tokenWithClaims 中 取出 customClaims
	// 這一行是說，把 token 中的 CustomClaims 依照 model.CustomClaims 的結構 parse 成 customClaims
	customClaims := tokenWithClaims.Claims.(*model.CustomClaims)
	// 驗證 tokenWithClaims 有效性而且驗證驗證封包 IP 跟 tokenWithClaims 中的是否一致，如果有效
	if tokenWithClaims.Valid == true && packetIP == customClaims.AuthInfo.AllowedClientIP {
		// 判斷使用者是否為 API
		if facilities.UserDb[customClaims.AuthInfo.UserID].UserCategoryID == 0 {
			// 如果是加驗證帳號密碼
			// 如果是 API 使用者要比較謹慎要使用帳號密碼才可以登出
			var usrAuthData model.UsrAuthData
			// 把 request 的 Body (JSON) parse 出來以後塞到 userInfo 中
			errGetUsrAuthData := json.NewDecoder(r.Body).Decode(&usrAuthData)
			if errGetUsrAuthData != nil {
				w.WriteHeader(http.StatusUnauthorized)
				fmt.Fprint(w, "安全性錯誤: 要登出 API session 必須要有帳號密碼。")
				return
			}

			// 根據 token 中的 使用者代碼取出 salted password hash
			sDbUserPwdHash, errGetDBPwdHash := facilities.GetUsrPwdHashFromDB(customClaims.AuthInfo.UserID)
			// 無法正確取得密碼，正常密碼的 hash 應該不會是空白
			if errGetDBPwdHash != nil || sDbUserPwdHash == "" {
				w.WriteHeader(http.StatusUnauthorized)
				fmt.Fprint(w, "權限問題：使用者資料庫認證資料錯誤，不是正確的使用者。")
				return
			}

			sUserSaltedPwHash := facilities.CalHashOfSaltedPw(usrAuthData.Pw) // 先從密碼算 hash 出來
			// 如果使用者傳來的密碼跟資料庫不符合也回傳錯誤訊息
			if sUserSaltedPwHash != sDbUserPwdHash {
				w.WriteHeader(http.StatusUnauthorized)
				fmt.Fprint(w, "權限問題：認證資料錯誤，無法登出。")
				return
			}
		}

		// 以上都通過就刪除 redis 上的 session key
		errUnlinkKey := facilities.UnLinkKey(fmt.Sprint(customClaims.AuthInfo.UserID, ":", packetIP))
		if errUnlinkKey != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, "session 問題："+errUnlinkKey.Error())
			return
		}

		// 如果都沒問題就回應成功訊息
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "登出成功。")
		return
	} else {
		// 如果無效就回傳錯誤訊息
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "權限問題：token 失效或是封包來源位置有誤")
		return
	}
}
