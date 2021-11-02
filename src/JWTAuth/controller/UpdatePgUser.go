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
)

func HandlerUpdOneUserInPg(w http.ResponseWriter, r *http.Request) {
	var objUserBeUpded model.UserDataForUpd
	var objReqUsrData model.UserCredentialsReqForUpd
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

	// parse http request 把 body 中要更新的使用者資料抓出來
	// 把 request 的 Body (JSON) parse 出來以後塞到 objReqUsrData 中
	errDecodeReqBodyJson := json.NewDecoder(r.Body).Decode(&objReqUsrData)
	// 資料 parse 有問題就 return
	if errDecodeReqBodyJson != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, fmt.Sprint("http request 相關： parse 欲更新的使用者資料失敗，", errDecodeReqBodyJson.Error()))
		return
	}

	// 這裡要檢查兩個條件
	// 確認使用者的 iUserCatID ，如果是 -1 表示是使用者管理員
	// 如果 objAuthRespData.UserID == objReqUsrData.UserID 表示是本人 (通常出現在使用者已經登入而且想修改密碼的情況)
	iUserCatID := facilities.UserDb[objAuthRespData.UserID].UserCategoryID
	if iUserCatID != -1 && objAuthRespData.UserID != objReqUsrData.UserID {
		sAuthErrMsg := "授權相關：非管理者權限或本人"
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintln(w, sAuthErrMsg)
		return
	}

	// 做資料庫連線準備
	ctxbg := context.Background()
	conn := UserDB.GetConn(facilities.GlobalOpPool)
	txUpdUser, errGenTx := conn.Begin(ctxbg)
	if errGenTx != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, fmt.Sprint("資料庫連線相關：建立 tansaction 物件失敗，", errGenTx.Error()))
		return
	}

	// 只要 tx commit 有成功，底下的 Rollback 就不會執行。
	defer txUpdUser.Rollback(ctxbg)
	defer conn.Release()

	// 填充要更新的使用者物件
	objUserBeUpded = model.UserDataForUpd{
		IntUserCatID:  objReqUsrData.UserCategoryID,
		StrUserID:     objReqUsrData.UserID,
		StrUserName:   objReqUsrData.UserName,
		StrPwHash:     facilities.CalHashOfSaltedPw(objReqUsrData.Password),
		IntIfCancel:   objReqUsrData.IntIfCancel,
		StrModifyOpID: objAuthRespData.UserID,
	}
	errUpdUser := UserDB.UpdUser(ctxbg, txUpdUser, objUserBeUpded)
	if errUpdUser != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, fmt.Sprint("資料庫相關：更新使用者資料失敗，", errUpdUser.Error()))
		// return 之前會自動 rollback
		return
	}
	// 更新完成就要 commit
	errCommit := txUpdUser.Commit(ctxbg)
	if errCommit != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, fmt.Sprint("資料庫相關：更新使用者 commit 失敗，", errCommit.Error()))
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
	fmt.Fprintln(w, fmt.Sprint("資料庫相關：更新使用者 ", objUserBeUpded.StrUserID, " 成功。"))
	return
}
