package main

import (
	_ "JWTAuth/dao/RedisDB"
	_ "JWTAuth/dao/UserDB"
	"JWTAuth/facilities"
	"JWTAuth/routes"
)

func main() {
	defer facilities.GlobalQryPool.Close()
	defer facilities.RedisClientReadOpr.Close()
	defer facilities.RedisClientWriteOpr.Close()
	routes.StartServer()
}
