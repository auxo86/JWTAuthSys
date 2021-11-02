package UsrAuthForWebapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"net/http"
)

// 全域環境變數設定
var JWTAuthEnv map[string]string
var errEnv error

// 全域的認證 server
var sAuthServer string

func init() {
	// 讀取環境設定檔
	// 作為一個專案的通用工具模組，必須使用引入本模組的主專案的 .env 檔。
	// 意思是說如果要使用這個 module 必須
	// 先修改主專案的 .env.template ，
	// 在 WebapiJwtValidation 處填入 JWTAuth 的 webapi 位置，
	// 然後把 .template 副檔名去掉
	JWTAuthEnv, errEnv = godotenv.Read()
	if errEnv != nil {
		log.Fatal("無法載入 .env 檔。")
	}

	sAuthServer = JWTAuthEnv["WebapiJwtValidation"]
}

// UserAuth 把 request 丟往認證 server 做身份確認。
func UserAuth(r *http.Request) (*ResponseValidationInfo, error) {
	mapAuthRespData := new(ResponseValidationInfo)

	tokenStr, errGetTokenStr := GetTokenStrFromReq(r)
	if errGetTokenStr != nil {
		return nil, errGetTokenStr
	}

	sBearerToken := "Bearer " + tokenStr

	// 取得 request 的 ip address
	sRemoteAddr, errGetRemoteIP := GetClientOnlyIP(r)
	if errGetRemoteIP != nil {
		return nil, errGetRemoteIP
	}
	// 組字串形成 JSON
	sJson := []byte(fmt.Sprint(`{"FromIP":"`, sRemoteAddr, `"}`))

	authReq, makeReqErr := http.NewRequest("POST", sAuthServer, bytes.NewBuffer(sJson))

	if makeReqErr != nil {
		return nil, makeReqErr
	}

	authReq.Header.Add("Content-Type", "application/json")
	authReq.Header.Add("Authorization", sBearerToken)

	client := &http.Client{}
	authResponse, authErr := client.Do(authReq)

	if authErr != nil {
		return nil, authErr
	}

	// 記得要關閉
	defer authResponse.Body.Close()

	readBodyErr := json.NewDecoder(authResponse.Body).Decode(&mapAuthRespData)
	if readBodyErr != nil {
		return nil, readBodyErr
	}

	return mapAuthRespData, nil
}
