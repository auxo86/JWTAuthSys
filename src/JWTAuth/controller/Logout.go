package controller

import (
	"JWTAuth/facilities"
	"JWTAuth/model"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

// LogoutHandler 使用者用來遠端刪除 redis 上的 session
// 一般使用者和 API 使用者不同
// 一般使用者只要 token 有效而且封包 IP 符合 token 的 IP 就可以刪除 redis 上的 session
// API 使用者為求慎重，要再加上帳號密碼確認才可以刪除
func LogoutHandler(c *fiber.Ctx) error {
	var customClaims model.CustomClaims
	// 取得封包的 IP address
	packetIP, errGetIP := facilities.GetClientOnlyIP(c.Context())
	if errGetIP != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "權限問題：無法取得 IP 位置。")
	}

	// 從 request 中 parse 出 bsTokenWithClaims
	errGetCustomClaims := facilities.GetCustomClaimsFromReq(c.Context(), &customClaims)

	if errGetCustomClaims != nil {
		return fiber.NewError(fiber.StatusUnauthorized, errGetCustomClaims.Error())
	}

	// 驗證驗證封包 IP 跟 bsTokenWithClaims 中的是否一致，如果有效
	if packetIP == customClaims.AuthInfo.AllowedClientIP {
		// 判斷使用者是否為 API (UserCategoryID == 0)
		if facilities.UserDb[customClaims.AuthInfo.UserID].UserCategoryID == 0 {
			// 如果是加驗證帳號密碼
			// 如果是 API 使用者要比較謹慎要使用帳號密碼才可以登出
			usrAuthData := new(model.UsrAuthData)
			// 把 request 的 Body (JSON) parse 出來以後塞到 usrAuthData 中
			if errDecode := c.BodyParser(&usrAuthData); errDecode != nil {
				return fiber.NewError(fiber.StatusUnauthorized, "安全性錯誤: error occurred while decoding http request body, "+errDecode.Error())
			}
			if usrAuthData == nil {
				return fiber.NewError(fiber.StatusUnauthorized, "安全性錯誤: 無法取得登出 API session 必須要用的帳號密碼。")
			}

			// 根據 token 中的 使用者代碼取出 salted password hash
			sDbUserPwdHash, errGetDBPwdHash := facilities.GetUsrPwdHashFromDB(customClaims.AuthInfo.UserID)
			// 無法正確取得密碼，正常密碼的 hash 應該不會是空白
			if errGetDBPwdHash != nil || sDbUserPwdHash == "" {
				return fiber.NewError(fiber.StatusUnauthorized, "權限問題：使用者資料庫認證資料錯誤，不是正確的使用者。")
			}

			errChkPass := bcrypt.CompareHashAndPassword([]byte(sDbUserPwdHash), []byte(usrAuthData.Pw))

			if errChkPass != nil {
				return fiber.NewError(fiber.StatusUnauthorized, "權限問題：認證資料錯誤，無法登出。")
			}
		}

		// 以上都通過就刪除 redis 上的 session key
		errUnlinkKey := facilities.UnLinkKey(fmt.Sprint(customClaims.AuthInfo.UserID, ":", packetIP))
		if errUnlinkKey != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "session 問題： "+errUnlinkKey.Error())
		}

		// 更新 DictAllSessionKeysOnRedis ( redis session keys cache )
		go facilities.UpdDictSessionKeys(false)
		// 如果都沒問題就回應成功訊息
		return c.Status(fiber.StatusOK).SendString("登出成功。")
	} else {
		// 如果無效就回傳錯誤訊息
		return fiber.NewError(fiber.StatusUnauthorized, "權限問題：token 失效或是封包來源位置有誤")
	}
}
