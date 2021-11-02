package UsrAuthForWebapi

import (
	"errors"
	"net/http"
	"regexp"
	"strings"
)

func RipIPFromStr(strIPWithPort string) (string, error) {
	var strIP string
	strSliceIPAddr := strings.Split(strIPWithPort, ":")
	if len(strSliceIPAddr) < 2 {
		return "", errors.New("無法取得 ip")
	}
	strIP = strSliceIPAddr[0]
	return strIP, nil
}

// 回傳 client 的 ip
// 如果 X-Forwarded-For 有 ip 可以抓就優先抓這個，否則抓 r.RemoteAddr
func GetClientOnlyIP(r *http.Request) (string, error) {
	sIP := ""
	sIPSeq := r.Header.Get("X-Forwarded-For")
	errGetIp := errors.New("")

	if sIPSeq != "" {
		sIP = strings.Split(sIPSeq, ", ")[0]
	} else {
		sIP, errGetIp = RipIPFromStr(r.RemoteAddr)
		if errGetIp != nil {
			return "", errGetIp
		}
	}
	return sIP, nil
}

func GetTokenStrFromReq(r *http.Request) (string, error) {
	// 做一個 regex 的置換 pattern ，只要是符合 "bearer "，不論大小寫，一律置換成空字串
	re := regexp.MustCompile(`(?i)bearer `)
	strBearerToken := r.Header["Authorization"][0]
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
