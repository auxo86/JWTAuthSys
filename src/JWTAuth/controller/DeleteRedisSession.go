package controller

import (
	"JWTAuth/facilities"
	"JWTAuth/model"
	"encoding/json"
	"fmt"
	"net/http"
	"tsgh.edu.tw/UsrAuthForWebapi"
)

func HandlerDelRedisSessionForOp(w http.ResponseWriter, r *http.Request) {
	var objSessionDataInReq model.SessionDataForOp
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

	// 這裡要檢查條件，確認使用者的 iUserCatID ，如果是 -1 表示是使用者管理員
	iUserCatID := facilities.UserDb[objAuthRespData.UserID].UserCategoryID
	if iUserCatID != -1 {
		sAuthErrMsg := "授權相關：非管理者權限"
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintln(w, sAuthErrMsg)
		return
	}

	// parse http request 把 body 中要更新的使用者資料抓出來
	// 把 request 的 Body (JSON) parse 出來以後塞到 objSessionDataInReq 中
	errDecodeReqBodyJson := json.NewDecoder(r.Body).Decode(&objSessionDataInReq)
	// 資料 parse 有問題就 return
	if errDecodeReqBodyJson != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, fmt.Sprint("http request 相關： parse 欲刪除的 session 資料失敗，", errDecodeReqBodyJson.Error()))
		return
	}
	// 產生 session key
	sSessionKeyWillBeUnlinked := objSessionDataInReq.StrUserID + ":" + objSessionDataInReq.StrIP

	// 以上都通過就刪除 redis 上的 session key
	errUnlinkKey := facilities.UnLinkKey(sSessionKeyWillBeUnlinked)
	if errUnlinkKey != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "Session server 相關：無法成功刪除 session ， "+errUnlinkKey.Error())
		return
	}

	// 如果都沒問題就回應成功訊息
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "刪除 session key "+sSessionKeyWillBeUnlinked+" 成功。")
	return
}
