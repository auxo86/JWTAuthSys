package UsrAuthForWebapi

import (
	"errors"
	"github.com/valyala/fasthttp"
	"net"
	"regexp"
	"strings"
)

// GetClientOnlyIP 回傳 client 的 ip
// 如果 X-Forwarded-For 有 ip 可以抓就優先抓這個，否則抓 ctxR.RemoteIP()
func GetClientOnlyIP(ctxR *fasthttp.RequestCtx) (string, error) {
	sIP := ""
	sIPList := string(ctxR.Request.Header.Peek("X-Forwarded-For"))

	if sIPList != "" {
		sIP = strings.Split(sIPList, ", ")[0]
	} else {
		sIP = ctxR.RemoteIP().String()
		if sIP == net.IPv4zero.String() {
			return "", errors.New("network error: can't get remote IP")
		}
	}
	return sIP, nil
}

func GetTokenStrFromReq(ctxR *fasthttp.RequestCtx) (string, error) {
	// 做一個 regex 的置換 pattern ，只要是符合 "bearer "，不論大小寫，一律置換成空字串
	re := regexp.MustCompile(`(?i)bearer[ ]{0,1}`)
	strBearerToken := string(ctxR.Request.Header.Peek("Authorization"))
	// 如果取不到 Bearer Token 就回應錯誤。
	if strBearerToken == "" {
		return "", errors.New("權限問題：Error extracting the token")
	}

	// 把 "(?i)bearer " 置換為空白
	strToken := re.ReplaceAllString(strBearerToken, "")
	if strToken == "" {
		return "", errors.New("權限問題：Error extracting the token")
	}
	return strToken, nil
}
