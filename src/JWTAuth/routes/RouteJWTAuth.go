package routes

import (
	"JWTAuth/controller"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/joho/godotenv"
	"log"

	"net/http"
	_ "net/http/pprof"
)

// StartServer 啟動伺服器
func StartServer() {
	var myEnv map[string]string
	myEnv, enverr := godotenv.Read()
	if enverr != nil {
		log.Fatal("無法載入 .env 檔。")
	}

	app := fiber.New()

	// 處理 OPTIONS 請求
	app.Use(func(c *fiber.Ctx) error {
		if c.Method() == "OPTIONS" {
			c.Set("Access-Control-Allow-Origin", "*")
			c.Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			c.Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE")
			return c.SendStatus(fiber.StatusNoContent)
		}
		return c.Next()
	})

	// 設定 CORS
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Content-Type, Authorization",
		AllowMethods: "POST, GET, DELETE, PUT",
	}))

	// 向 redis server 註冊一個 session token
	app.Post("/login", controller.LoginHandler)
	// 這行的意思是，先驗證 token 的有效性
	// 解碼 JWT 後，驗證 是否可以 extend session 的 TTL 藉此判斷 session 的有效性。
	app.Post("/JwtValidation", controller.HandlerValidateJWT)
	// logout ，刪除 session ，一般使用者需要 token + IP ，API 使用者要加 帳密
	app.Post("/logout", controller.LogoutHandler)
	// 向 userauth database 新增一個新使用者。
	app.Post("/AddOneUser", controller.HandlerAddOnePgUser)
	// 向 userauth database 新增一批新使用者。
	app.Post("/BatchAddUsers", controller.HandlerBatchAddPgUsers)
	// 向 userauth database 查詢一個使用者。
	app.Post("/GetUserData", controller.HandlerGetOneUserFromPg)
	// 向 userauth database 更新一個新使用者。
	app.Post("/UpdOneUser", controller.HandlerUpdOneUserInPg)
	// 管理員功能，刪除 redis server 上的 session
	app.Post("/DeleteOneSession", controller.HandlerDelRedisSessionForOp)

	// 啟動 pprof 監聽器
	go func() {
		fmt.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	log.Println("Now listening...")
	log.Fatal(app.ListenTLS(":8080", myEnv["sshCert"], myEnv["sshKey"]))
}
