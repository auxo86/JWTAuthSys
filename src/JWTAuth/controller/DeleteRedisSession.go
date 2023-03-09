package controller

import (
	"JWTAuth/facilities"
	"JWTAuth/model"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"tsgh.edu.tw/UsrAuthForWebapi"
)

func HandlerDelRedisSessionForOp(c *fiber.Ctx) error {
	var objSessionDataInReq model.SessionDataForOp

	// 先做身份驗證，驗證過了再繼續往下
	objAuthRespData, errAuth := UsrAuthForWebapi.UserAuth(c.Context())
	if errAuth != nil {
		return fiber.NewError(fiber.StatusForbidden, fmt.Sprint("認證相關：token 驗證過程出錯，", errAuth))
	}

	if objAuthRespData.UserValid != true {
		return fiber.NewError(fiber.StatusForbidden, "認證相關：token 無效，請重新登入。")
	}

	// 這裡要檢查條件，確認使用者的 iUserCatID ，如果是 -1 表示是使用者管理員
	iUserCatID := facilities.UserDb[objAuthRespData.UserID].UserCategoryID
	if iUserCatID != -1 {
		return fiber.NewError(fiber.StatusUnauthorized, "授權相關：非管理者權限")
	}

	// parse http request 把 body 中要更新的使用者資料抓出來
	// 把 request 的 Body (JSON) parse 出來以後塞到 objSessionDataInReq 中
	if errDecodeReqBodyJson := c.BodyParser(&objSessionDataInReq); errDecodeReqBodyJson != nil {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprint("http request 相關： parse 欲刪除的 session 資料失敗，", errDecodeReqBodyJson.Error()))
	}

	// 產生 session key
	sSessionKeyWillBeUnlinked := objSessionDataInReq.StrUserID + ":" + objSessionDataInReq.StrIP

	// 以上都通過就刪除 redis 上的 session key
	errUnlinkKey := facilities.UnLinkKey(sSessionKeyWillBeUnlinked)
	if errUnlinkKey != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Session server 相關：無法成功刪除 session ， "+errUnlinkKey.Error())
	}

	// 如果都沒問題就回應成功訊息
	// 更新 DictAllSessionKeysOnRedis ( redis session keys cache )
	go facilities.UpdDictSessionKeys(false)
	return c.Status(fiber.StatusOK).SendString("刪除 session key " + sSessionKeyWillBeUnlinked + " 成功。")
}
