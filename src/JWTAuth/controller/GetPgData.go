package controller

import (
	"JWTAuth/dao/UserDB"
	"JWTAuth/facilities"
	"JWTAuth/model"
	"fmt"
	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"tsgh.edu.tw/UsrAuthForWebapi"
)

func HandlerGetOneUserFromPg(c *fiber.Ctx) error {
	var objUserForQry model.UserIDForQry
	// 先做身份驗證，驗證過了再繼續往下
	objAuthRespData, errAuth := UsrAuthForWebapi.UserAuth(c.Context())
	if errAuth != nil {
		return fiber.NewError(fiber.StatusForbidden, fmt.Sprint("認證相關：token 驗證過程出錯，", errAuth))
	}

	if objAuthRespData.UserValid != true {
		return fiber.NewError(fiber.StatusForbidden, "認證相關：token 無效，請重新登入。")
	}

	// 確認使用者的 usercategory.id ，如果是 -1 表示是使用者管理員
	iUserCatID := facilities.UserDb[objAuthRespData.UserID].UserCategoryID
	if iUserCatID != -1 {
		return fiber.NewError(fiber.StatusUnauthorized, "授權相關：非管理者權限")
	}

	// parse http request 把 body 中要新增的使用者資料抓出來
	// 把 request 的 Body (JSON) parse 出來以後塞到 objUserWillBeAdded （呼叫的客戶端的來源地址） 中
	if errDecodeReqBodyJson := c.BodyParser(&objUserForQry); errDecodeReqBodyJson != nil {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprint("http request 相關： parse 欲查詢的使用者資料失敗，", errDecodeReqBodyJson.Error()))
	}

	// 向資料庫查詢使用者
	ptrUserData, errQryUser := UserDB.GetUserFromPg(objUserForQry.UserID)
	if errQryUser != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprint("資料庫相關：查詢使用者 ", objUserForQry.UserID, " 失敗，", errQryUser.Error()))
	}

	// 回傳查詢使用者成功的 http response
	bsUserData, errToJSON := json.Marshal(ptrUserData)
	if errToJSON != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "JSON 相關：無法成功轉換成 JSON")
	}

	// 設置 header 的 ContentType 為 "application/json; charset=utf-8"
	c.Response().Header.SetContentType("application/json; charset=utf-8")

	return c.Status(fiber.StatusOK).Send(bsUserData)
}
