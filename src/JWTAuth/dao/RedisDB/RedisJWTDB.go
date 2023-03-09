package RedisDB

import (
	"JWTAuth/facilities"
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
	"os"
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

	// 處理連線失敗也不會顯示失敗的問題
	facilities.RedisClientWriteOpr = redis.NewClient(&redis.Options{
		Addr:       sRedisHost + ":" + sRedisWritePort,
		Username:   sRedisUpdater,
		Password:   sRedisUpdaterPwd,
		DB:         iRedisDbName, // use default DB
		PoolSize:   20,
		MaxRetries: -1,
	})

	facilities.RedisClientReadOpr = redis.NewClient(&redis.Options{
		Addr:       sRedisHost + ":" + sRedisReadPort,
		Username:   sRedisReader,
		Password:   sRedisReaderPwd,
		DB:         iRedisDbName, // use default DB
		PoolSize:   50,
		MaxRetries: -1,
	})

	if err := ifRedisConnected(facilities.RedisClientWriteOpr); err != nil {
		fmt.Println("無法連線到 redis cluster: 請檢查 redis 伺服器及 opr 連線情況。")
		os.Exit(1)
	}

	if err := ifRedisConnected(facilities.RedisClientReadOpr); err != nil {
		fmt.Println("無法連線到 redis cluster: 請檢查 redis 伺服器及 reader 連線情況。")
		os.Exit(1)
	}

	fmt.Println("RedisClientWriteOpr & RedisClientReadOpr was created and Successfully connected!")

	go facilities.UpdDictSessionKeys(true)
}

func ifRedisConnected(RedisClient *redis.Client) error {
	ctxbg := context.Background()
	// 使用 ping() 測試是否有成功連線 (好像只有這個辦法)
	if _, errRedisConnSuccess := RedisClient.Ping(ctxbg).Result(); errRedisConnSuccess != nil {
		return errRedisConnSuccess
	}
	return nil
}
