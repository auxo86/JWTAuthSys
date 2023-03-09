package controller

import (
	"JWTAuth/dao/UserDB"
	"JWTAuth/facilities"
	"JWTAuth/model"
	"context"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"strconv"
	"tsgh.edu.tw/UsrAuthForWebapi"
	_ "tsgh.edu.tw/UsrAuthForWebapi"
)

// 僅 insert 一個使用者。
func HandlerAddOnePgUser(c *fiber.Ctx) error {
	objUserWillBeAdded := new(model.NewUserCredentials)

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
	// 把 request 的 Body (JSON) parse 出來以後塞到 objUserWillBeAdded 中
	// 資料 parse 有問題就 return
	if errDecodeReqBodyJson := c.BodyParser(objUserWillBeAdded); errDecodeReqBodyJson != nil {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprint("http request 相關： parse 欲新增的使用者資料失敗，", errDecodeReqBodyJson.Error()))
	}

	// 做資料庫連線準備
	ctxbg := context.Background()
	conn := UserDB.GetConn(facilities.GlobalOpPool)
	txAddUser, errGenTx := conn.Begin(ctxbg)
	if errGenTx != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprint("資料庫連線相關：建立 tansaction 物件失敗，", errGenTx.Error()))
	}
	// 只要 tx commit 有成功，底下的 Rollback 就不會執行。
	defer txAddUser.Rollback(ctxbg)
	defer conn.Release()
	// 利用 transaction 物件向資料庫插入新使用者
	errInsertUser := UserDB.InsertPgUserAuthDB(ctxbg, txAddUser, objAuthRespData.UserID, *objUserWillBeAdded)
	if errInsertUser != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprint("資料庫相關：新增使用者失敗，", errInsertUser.Error()))
	}
	// 插入完成就要 commit
	errCommit := txAddUser.Commit(ctxbg)
	if errCommit != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprint("資料庫相關：新增使用者 commit 失敗，", errCommit.Error()))
	}
	// 更新 facilities.UserDb
	errReloadUserDB := UserDB.LoadUserDB()
	if errReloadUserDB != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprint("JWTAuth 相關：更新使用者資料 map 失敗，", errReloadUserDB.Error()))
	}
	// 回傳新增使用者成功的 http response
	return c.Status(fiber.StatusOK).SendString("資料庫相關：新增使用者 " + objUserWillBeAdded.UserID + " 成功。")
}

func HandlerBatchAddPgUsers(c *fiber.Ctx) error {
	var sliceUsersWillBeAdded []model.NewUserCredentials
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
	// 把 request 的 Body (JSON) parse 出來以後塞到 sliceUsersWillBeAdded 中
	if errDecodeReqBodyJson := c.BodyParser(&sliceUsersWillBeAdded); errDecodeReqBodyJson != nil {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprint("http request 相關： parse 欲新增的使用者資料失敗，", errDecodeReqBodyJson.Error()))
	}

	// 做資料庫連線準備
	ctxbg := context.Background()
	conn := UserDB.GetConn(facilities.GlobalOpPool)
	txAddUser, errGenTx := conn.Begin(ctxbg)
	if errGenTx != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprint("資料庫連線相關：建立 tansaction 物件失敗，", errGenTx.Error()))
	}
	// 只要 tx commit 有成功，底下的 Rollback 就不會執行。
	defer txAddUser.Rollback(ctxbg)
	defer conn.Release()
	// 利用 transaction 物件向資料庫插入新使用者
	iCopyCnt, errInsertUser := UserDB.BatchInsertPgUsersAuthDB(ctxbg, txAddUser, objAuthRespData.UserID, sliceUsersWillBeAdded)
	if errInsertUser != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprint("資料庫相關：新增使用者失敗，", errInsertUser.Error()))
	}
	// 插入完成就要 commit
	errCommit := txAddUser.Commit(ctxbg)
	if errCommit != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprint("資料庫相關：新增使用者 commit 失敗，", errCommit.Error()))
	}
	// 更新 facilities.UserDb
	errReloadUserDB := UserDB.LoadUserDB()
	if errReloadUserDB != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprint("JWTAuth 相關：更新使用者資料 map 失敗，", errReloadUserDB.Error()))
	}
	// 回傳新增使用者成功的 http response
	return c.Status(fiber.StatusOK).SendString("資料庫相關：批次新增使用者 " + strconv.FormatInt(iCopyCnt, 10) + " 位成功。")
}
