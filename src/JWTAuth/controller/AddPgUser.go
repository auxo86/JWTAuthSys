package controller

import (
	"JWTAuth/dao/UserDB"
	"JWTAuth/facilities"
	"JWTAuth/model"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"tsgh.edu.tw/UsrAuthForWebapi"
	_ "tsgh.edu.tw/UsrAuthForWebapi"
)

// 僅 insert 一個使用者。
func HandlerAddOnePgUser(w http.ResponseWriter, r *http.Request) {
	var objUserWillBeAdded model.NewUserCredentials
	// 先做身份驗證，驗證過了再繼續往下
	objAuthRespData, errAuth := UsrAuthForWebapi.UserAuth(r)
	if errAuth != nil {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprintln(w, fmt.Sprint("認證相關：token 驗證過程出錯，", errAuth))
		return
	}

	if objAuthRespData.UserValid != true {
		sErrMsg := "認證相關：token 無效，請重新登入。"
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprintln(w, sErrMsg)
		return
	}

	// 確認使用者的 usercategory.id ，如果是 -1 表示是使用者管理員
	iUserCatID := facilities.UserDb[objAuthRespData.UserID].UserCategoryID
	if iUserCatID != -1 {
		sAuthErrMsg := "授權相關：非管理者權限"
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintln(w, sAuthErrMsg)
		return
	}

	// parse http request 把 body 中要新增的使用者資料抓出來
	// 把 request 的 Body (JSON) parse 出來以後塞到 objUserWillBeAdded 中
	errDecodeReqBodyJson := json.NewDecoder(r.Body).Decode(&objUserWillBeAdded)
	// 資料 parse 有問題就 return
	if errDecodeReqBodyJson != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, fmt.Sprint("http request 相關： parse 欲新增的使用者資料失敗，", errDecodeReqBodyJson.Error()))
		return
	}
	// 做資料庫連線準備
	ctxbg := context.Background()
	conn := UserDB.GetConn(facilities.GlobalOpPool)
	txAddUser, errGenTx := conn.Begin(ctxbg)
	if errGenTx != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, fmt.Sprint("資料庫連線相關：建立 tansaction 物件失敗，", errGenTx.Error()))
		return
	}
	// 只要 tx commit 有成功，底下的 Rollback 就不會執行。
	defer txAddUser.Rollback(ctxbg)
	defer conn.Release()
	// 利用 transaction 物件向資料庫插入新使用者
	errInsertUser := UserDB.InsertPgUserAuthDB(ctxbg, txAddUser, objAuthRespData.UserID, objUserWillBeAdded)
	if errInsertUser != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, fmt.Sprint("資料庫相關：新增使用者失敗，", errInsertUser.Error()))
		// return 之前會自動 rollback
		return
	}
	// 插入完成就要 commit
	errCommit := txAddUser.Commit(ctxbg)
	if errCommit != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, fmt.Sprint("資料庫相關：新增使用者 commit 失敗，", errCommit.Error()))
		// return 之前會自動 rollback
		return
	}
	// 更新 facilities.UserDb
	errReloadUserDB := UserDB.LoadUserDB()
	if errReloadUserDB != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, fmt.Sprint("JWTAuth 相關：更新使用者資料 map 失敗，", errReloadUserDB.Error()))
		return
	}
	// 回傳新增使用者成功的 http response
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, fmt.Sprint("資料庫相關：新增使用者 ", objUserWillBeAdded.UserID, " 成功。"))
	return
}

func HandlerBatchAddPgUsers(w http.ResponseWriter, r *http.Request) {
	var sliceUsersWillBeAdded []model.NewUserCredentials
	// 先做身份驗證，驗證過了再繼續往下
	objAuthRespData, errAuth := UsrAuthForWebapi.UserAuth(r)
	if errAuth != nil {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprintln(w, fmt.Sprint("認證相關：token 驗證過程出錯，", errAuth))
		return
	}

	if objAuthRespData.UserValid != true {
		sErrMsg := "認證相關：token 無效，請重新登入。"
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprintln(w, sErrMsg)
		return
	}

	// 確認使用者的 usercategory.id ，如果是 -1 表示是使用者管理員
	iUserCatID := facilities.UserDb[objAuthRespData.UserID].UserCategoryID
	if iUserCatID != -1 {
		sAuthErrMsg := "授權相關：非管理者權限"
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintln(w, sAuthErrMsg)
		return
	}

	// parse http request 把 body 中要新增的使用者資料抓出來
	// 把 request 的 Body (JSON) parse 出來以後塞到 sliceUsersWillBeAdded 中
	errDecodeReqBodyJson := json.NewDecoder(r.Body).Decode(&sliceUsersWillBeAdded)
	// 資料 parse 有問題就 return
	if errDecodeReqBodyJson != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, fmt.Sprint("http request 相關： parse 欲新增的使用者資料失敗，", errDecodeReqBodyJson.Error()))
		return
	}
	// 做資料庫連線準備
	ctxbg := context.Background()
	conn := UserDB.GetConn(facilities.GlobalOpPool)
	txAddUser, errGenTx := conn.Begin(ctxbg)
	if errGenTx != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, fmt.Sprint("資料庫連線相關：建立 tansaction 物件失敗，", errGenTx.Error()))
		return
	}
	// 只要 tx commit 有成功，底下的 Rollback 就不會執行。
	defer txAddUser.Rollback(ctxbg)
	defer conn.Release()
	// 利用 transaction 物件向資料庫插入新使用者
	iCopyCnt, errInsertUser := UserDB.BatchInsertPgUsersAuthDB(ctxbg, txAddUser, objAuthRespData.UserID, sliceUsersWillBeAdded)
	if errInsertUser != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, fmt.Sprint("資料庫相關：新增使用者失敗，", errInsertUser.Error()))
		// return 之前會自動 rollback
		return
	}
	// 插入完成就要 commit
	errCommit := txAddUser.Commit(ctxbg)
	if errCommit != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, fmt.Sprint("資料庫相關：新增使用者 commit 失敗，", errCommit.Error()))
		// return 之前會自動 rollback
		return
	}
	// 更新 facilities.UserDb
	errReloadUserDB := UserDB.LoadUserDB()
	if errReloadUserDB != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, fmt.Sprint("JWTAuth 相關：更新使用者資料 map 失敗，", errReloadUserDB.Error()))
		return
	}
	// 回傳新增使用者成功的 http response
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, fmt.Sprint("資料庫相關：批次新增使用者 ", iCopyCnt, " 位成功。"))
	return
}
