package UsrAuthForWebapi

import (
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/valyala/fasthttp"
	"log"
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

// UserAuth 把 request 丟往認證 server 做身份確認。他有幾件事要做：
// 第一要先取得 request 中的 JWT token ，
// 第二要取得 request 中的 ip 位置，
// 第三要利用這兩者組成新的 http request ，
// 第四要傳送到 JWTAuth 的 JwtValidation 做認證並回傳認證結果。
func UserAuth(ctxR *fasthttp.RequestCtx) (*ResponseValidationInfo, error) {
	mapAuthRespData := new(ResponseValidationInfo)
	authReq := fasthttp.AcquireRequest()
	authResponse := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(authReq)
	defer fasthttp.ReleaseResponse(authResponse)

	// 先取得 request 中的 JWT token
	tokenStr, errGetTokenStr := GetTokenStrFromReq(ctxR)
	if errGetTokenStr != nil {
		return nil, errGetTokenStr
	}

	sBearerToken := "Bearer " + tokenStr

	// 取得 request 的 ip address
	sRemoteAddr, errGetRemoteIP := GetClientOnlyIP(ctxR)
	if errGetRemoteIP != nil {
		return nil, errGetRemoteIP
	}
	// 組字串形成 JSON
	sJson := []byte(fmt.Sprint(`{"FromIP":"`, sRemoteAddr, `"}`))

	// 利用這兩者組成新的 http request
	authReq.SetRequestURI(sAuthServer)
	authReq.Header.SetMethod("POST")
	authReq.Header.Add("Content-Type", "application/json")
	authReq.Header.Add("Authorization", sBearerToken)
	authReq.SetBody(sJson)

	// 要傳送到 JWTAuth 的 JwtValidation 做認證並回傳認證結果
	if errSendAuthReq := fasthttp.Do(authReq, authResponse); errSendAuthReq != nil {
		return nil, errSendAuthReq
	}

	// 取回認證結果的 http response ，並 parse body 為自定義的 struct
	if errParseAuthRespBody := json.Unmarshal(authResponse.Body(), &mapAuthRespData); errParseAuthRespBody != nil {
		return nil, errParseAuthRespBody
	}

	return mapAuthRespData, nil
}
