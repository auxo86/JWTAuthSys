package RedisDB

import (
	"JWTAuth/facilities"
	"fmt"
	"github.com/go-redis/redis/v8"
	"strconv"
)

func init() {
	sRedisHost := facilities.MyEnv["RedisHost"]
	sRedisReadPort := facilities.MyEnv["RedisReadPort"]
	sRedisReader := facilities.MyEnv["RedisReader"]
	sRedisReaderPwd := facilities.MyEnv["PwdRedisReader"]

	sRedisWritePort := facilities.MyEnv["RedisWritePort"]
	sRedisUpdater := facilities.MyEnv["RedisOpr"]
	sRedisUpdaterPwd := facilities.MyEnv["PwdRedisOpr"]

	iRedisDbName, _ := strconv.Atoi(facilities.MyEnv["RedisDbName"])

	facilities.RedisClientWriteOpr = redis.NewClient(&redis.Options{
		Addr:     sRedisHost + ":" + sRedisWritePort,
		Username: sRedisUpdater,
		Password: sRedisUpdaterPwd,
		DB:       iRedisDbName, // use default DB
	})

	facilities.RedisClientReadOpr = redis.NewClient(&redis.Options{
		Addr:     sRedisHost + ":" + sRedisReadPort,
		Username: sRedisReader,
		Password: sRedisReaderPwd,
		DB:       iRedisDbName, // use default DB
	})

	fmt.Println("RedisClientWriteOpr & RedisClientReadOpr was created and Successfully connected!")
}
