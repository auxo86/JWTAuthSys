package routes

import (
	"JWTAuth/controller"
	"JWTAuth/facilities"
	"github.com/joho/godotenv"
	"log"
	"net/http"
)

// StartServer 啟動伺服器
func StartServer() {
	var myEnv map[string]string
	myEnv, enverr := godotenv.Read()
	if enverr != nil {
		log.Fatal("無法載入 .env 檔。")
	}

	// 不使用 Default 的 ServeMux
	myMux := http.NewServeMux()

	// 設定登入使用的 Handler ，這句的意思是說，
	// 把 LoginHandler (HandlerFunction) 轉換為 Handler
	// 經過這樣的轉換才可以使用 ServeMux.Handle 做 url rewrite
	LoginHandler := http.HandlerFunc(controller.LoginHandler)
	// ServeMux.Handle() 第二個參數是 Handler
	// 所以要有上面那一行
	myMux.Handle("/login", facilities.CorsHandler(LoginHandler))
	// 這行的意思是，先驗證 token 的有效性
	// 解碼 JWT 後，驗證 是否可以 extend session 的 TTL 藉此判斷 session 的有效性。
	myMux.Handle("/JwtValidation", facilities.CorsHandler(http.HandlerFunc(controller.HandlerValidateJWT)))
	// logout ，刪除 session ，一般使用者需要 token + IP ，API 使用者要加 帳密
	myMux.Handle("/logout", facilities.CorsHandler(http.HandlerFunc(controller.LogoutHandler)))
	// 向 userauth database 新增一個新使用者。
	myMux.Handle("/AddOneUser", facilities.CorsHandler(http.HandlerFunc(controller.HandlerAddOnePgUser)))
	// 向 userauth database 新增一批新使用者。
	myMux.Handle("/BatchAddUsers", facilities.CorsHandler(http.HandlerFunc(controller.HandlerBatchAddPgUsers)))
	// 向 userauth database 查詢一個使用者。
	myMux.Handle("/GetUserData", facilities.CorsHandler(http.HandlerFunc(controller.HandlerGetOneUserFromPg)))
	// 向 userauth database 更新一個新使用者。
	myMux.Handle("/UpdOneUser", facilities.CorsHandler(http.HandlerFunc(controller.HandlerUpdOneUserInPg)))
	// 管理員功能，刪除 redis server 上的 session
	myMux.Handle("/DeleteOneSession", facilities.CorsHandler(http.HandlerFunc(controller.HandlerDelRedisSessionForOp)))

	log.Println("Now listening...")
	log.Fatal(http.ListenAndServeTLS(":8080", myEnv["sshCert"], myEnv["sshKey"], myMux))
}
