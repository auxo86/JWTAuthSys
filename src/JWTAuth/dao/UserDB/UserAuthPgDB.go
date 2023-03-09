package UserDB

import (
	"JWTAuth/facilities"
	"JWTAuth/model"
	"context"
	"fmt"
	"github.com/jackc/pgx/v4/pgxpool"
	"os"
	"strconv"
)

// 得到 Database 的連線池，並指派給 GlobalQryPool
// 如果連線失敗會中斷並且傳回錯誤訊息
func init() {
	sPgSQLHost := facilities.MyEnv["PgSQLHost"]
	iPgSQLPort, _ := strconv.Atoi(facilities.MyEnv["PgSQLPort"])
	sPgSQLDbName := facilities.MyEnv["PgSQLDbName"]
	sPgSQLQryUser := facilities.MyEnv["PgSQLQryUser"]
	sPgSQLQryUserPw := facilities.MyEnv["PgSQLQryUserPw"]

	sPgSQLOpUser := facilities.MyEnv["PgSQLOpUser"]
	sPgSQLOpUserPw := facilities.MyEnv["PgSQLOpUserPw"]

	// 因為要初始化兩個 pool ，所以做了一個 slice ，要利用 for 一次初始化所有的 pools
	sliceAllPools := []model.PgDbPool{
		{&facilities.GlobalQryPool, sPgSQLHost, iPgSQLPort, sPgSQLQryUser, sPgSQLQryUserPw, sPgSQLDbName},
		{&facilities.GlobalOpPool, sPgSQLHost, iPgSQLPort, sPgSQLOpUser, sPgSQLOpUserPw, sPgSQLDbName},
	}

	for _, poolData := range sliceAllPools {
		errInitPool := InitPools(poolData)
		if errInitPool != nil {
			fmt.Println("無法連線到 UserPgDB: 請檢查資料庫及其連線情況。")
			os.Exit(1)
		}
	}

	fmt.Println("GlobalQryPool and GlobalOpPool are created and Successfully connected!")
}

func InitPools(dataPool model.PgDbPool) error {
	ctxbg := context.Background()
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		dataPool.StrPgSQLHost, dataPool.IntPgSQLPort, dataPool.StrPgSQLUser, dataPool.StrPgSQLPw, dataPool.StrPgSQLDbName)

	var errInitPool error
	*dataPool.Pool, errInitPool = pgxpool.Connect(ctxbg, psqlInfo)
	if errInitPool != nil {
		return errInitPool
	}

	return nil
}
