package controller

import (
	"JWTAuth/dao/UserDB"
	"JWTAuth/facilities"
	"JWTAuth/model"
	"encoding/json"
	"fmt"
	"net/http"
	"tsgh.edu.tw/UsrAuthForWebapi"
)

func HandlerGetOneUserFromPg(w http.ResponseWriter, r *http.Request) {
	var objUserForQry model.UserIDForQry
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
	// 把 request 的 Body (JSON) parse 出來以後塞到 objUserWillBeAdded （呼叫的客戶端的來源地址） 中
	errDecodeReqBodyJson := json.NewDecoder(r.Body).Decode(&objUserForQry)
	// 資料 parse 有問題就 return
	if errDecodeReqBodyJson != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, fmt.Sprint("http request 相關： parse 欲查詢的使用者資料失敗，", errDecodeReqBodyJson.Error()))
		return
	}
	// 向資料庫查詢使用者
	ptrUserData, errQryUser := UserDB.GetUserFromPg(objUserForQry.UserID)
	if errQryUser != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, fmt.Sprint("資料庫相關：查詢使用者 ", objUserForQry.UserID, " 失敗，", errQryUser.Error()))
		return
	}

	// 回傳新增使用者成功的 http response
	w.WriteHeader(http.StatusOK)
	// 把結構經由序列化寫到 body 中。
	facilities.JsonResponse(*ptrUserData, w)
	return
}
