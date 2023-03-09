package controller

import (
	"JWTAuth/dao/UserDB"
	"JWTAuth/facilities"
	"JWTAuth/model"
	"context"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"tsgh.edu.tw/UsrAuthForWebapi"
)

func HandlerUpdOneUserInPg(c *fiber.Ctx) error {
	var objUserBeUpded model.UserDataForUpd
	var objReqUsrData model.UserCredentialsReqForUpd
	// 先做身份驗證，驗證過了再繼續往下
	objAuthRespData, errAuth := UsrAuthForWebapi.UserAuth(c.Context())
	if errAuth != nil {
		return fiber.NewError(fiber.StatusForbidden, fmt.Sprint("認證相關：token 驗證過程出錯，", errAuth))
	}

	if objAuthRespData.UserValid != true {
		return fiber.NewError(fiber.StatusForbidden, "認證相關：token 無效，請重新登入。")
	}

	// parse http request 把 body 中要更新的使用者資料抓出來
	// 把 request 的 Body (JSON) parse 出來以後塞到 objReqUsrData 中
	if errDecodeReqBodyJson := c.BodyParser(&objReqUsrData); errDecodeReqBodyJson != nil {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprint("http request 相關： parse 欲更新的使用者資料失敗，", errDecodeReqBodyJson.Error()))
	}

	// 這裡要檢查兩個條件
	// 確認使用者的 iUserCatID ，如果是 -1 表示是使用者管理員
	// 如果 objAuthRespData.UserID == objReqUsrData.UserID 表示是本人 (通常出現在使用者已經登入而且想修改密碼的情況)
	iUserCatID := facilities.UserDb[objAuthRespData.UserID].UserCategoryID
	if iUserCatID != -1 && objAuthRespData.UserID != objReqUsrData.UserID {
		return fiber.NewError(fiber.StatusUnauthorized, "授權相關：非管理者權限或本人")
	}

	// 做資料庫連線準備
	ctxbg := context.Background()
	conn := UserDB.GetConn(facilities.GlobalOpPool)
	txUpdUser, errGenTx := conn.Begin(ctxbg)
	if errGenTx != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprint("資料庫連線相關：建立 tansaction 物件失敗，", errGenTx.Error()))
	}

	// 只要 tx commit 有成功，底下的 Rollback 就不會執行。
	defer txUpdUser.Rollback(ctxbg)
	defer conn.Release()

	// 計算 user pass salted hash
	byteArrayPwSaltedHash, errCalHash := facilities.CalHashOfSaltedPw(objReqUsrData.Password)
	if errCalHash != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "系統錯誤：無法正確計算 pass hash, "+errCalHash.Error())
	}

	// 填充要更新的使用者物件
	objUserBeUpded = model.UserDataForUpd{
		IntUserCatID:  objReqUsrData.UserCategoryID,
		StrUserID:     objReqUsrData.UserID,
		StrUserName:   objReqUsrData.UserName,
		StrPwHash:     string(byteArrayPwSaltedHash),
		IntIfCancel:   objReqUsrData.IntIfCancel,
		StrModifyOpID: objAuthRespData.UserID,
	}
	errUpdUser := UserDB.UpdUser(ctxbg, txUpdUser, objUserBeUpded)
	if errUpdUser != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprint("資料庫相關：更新使用者資料失敗，", errUpdUser.Error()))
	}
	// 更新完成就要 commit
	errCommit := txUpdUser.Commit(ctxbg)
	if errCommit != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprint("資料庫相關：更新使用者 commit 失敗，", errCommit.Error()))
	}
	// 更新 facilities.UserDb
	errReloadUserDB := UserDB.LoadUserDB()
	if errReloadUserDB != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprint("JWTAuth 相關：更新使用者資料 map 失敗，", errReloadUserDB.Error()))
	}
	// 回傳新增使用者成功的 http response
	return c.Status(fiber.StatusOK).SendString("資料庫相關：更新使用者 " + objUserBeUpded.StrUserID + " 成功。")
}
