package UserDB

import (
	"JWTAuth/facilities"
	"JWTAuth/model"
	"context"
	"errors"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"time"
)

func init() {
	errLoadUserDB := LoadUserDB()
	if errLoadUserDB != nil {
		panic(errLoadUserDB)
	}
}

// GetConn 從連線池取得連線
func GetConn(usePool *pgxpool.Pool) *pgxpool.Conn {
	ctxbg := context.Background()
	conn, connerr := usePool.Acquire(ctxbg)
	if connerr != nil {
		panic(connerr)
	}

	return conn
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func LoadUserDB() error {
	ctxbg := context.Background()
	conn := GetConn(facilities.GlobalQryPool)
	defer conn.Release()

	var tmpUserDb = map[string]model.DBUserCredentials{}
	// 讀入 userauth database 使用者資料
	sSQLGetAllUsr := `select categoryid, userid, username, pwhash
						from users.usersecret
						where cancelflag = 0`

	rows, errQry := conn.Query(ctxbg, sSQLGetAllUsr)
	if errQry != nil {
		return errQry
	}

	defer rows.Close()
	for rows.Next() {
		var userFromRow model.DBUserCredentials
		errRowScan := rows.Scan(&userFromRow.UserCategoryID, &userFromRow.UserID, &userFromRow.UserName, &userFromRow.PwHash)
		checkErr(errRowScan)
		tmpUserDb[userFromRow.UserID] = userFromRow
	}
	facilities.UserDb = tmpUserDb
	return nil
}

// InsertPgUserAuthDB 傳入 ctxbg 和 conn 是為了要可以實現 transaction
func InsertPgUserAuthDB(ctxbg context.Context, tx pgx.Tx, sCreateOpID string, objUserWillBeAdded model.NewUserCredentials) error {
	// 定義要塞入資料表的 當地現在時間
	var tNowLocal time.Time
	// 定義寫入 userauth database 使用者資料的 SQL 字串
	sSQLInsertUsr := `
	insert into users.usersecret
		(categoryid, userid, username, pwhash, cancelflag, createopid, modifyopid, createdatetime, modifydatetime)
	values
		($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	// 先取得當地現在時間
	tNowLocal = time.Now().Local()
	// 塞資料進 DB，這裡要注意，塞密碼的時候要轉換成 bcrypt hash 。
	// 計算 user pass salted hash
	byteArrayPwSaltedHash, errCalHash := facilities.CalHashOfSaltedPw(objUserWillBeAdded.Password)
	if errCalHash != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "系統錯誤：無法正確計算 pass hash, "+errCalHash.Error())
	}

	_, errInsertUser := tx.Exec(ctxbg, sSQLInsertUsr,
		objUserWillBeAdded.UserCategoryID, objUserWillBeAdded.UserID, objUserWillBeAdded.UserName, string(byteArrayPwSaltedHash),
		0, sCreateOpID, sCreateOpID, tNowLocal, tNowLocal)
	if errInsertUser != nil {
		return errInsertUser
	}

	return nil
}

// BatchInsertPgUsersAuthDB 批次插入使用者資料到 userauth database 中。
func BatchInsertPgUsersAuthDB(ctxbg context.Context, tx pgx.Tx, sCreateOpID string, sliceUsersWillBeAdded []model.NewUserCredentials) (int64, error) {
	// 定義要塞入資料庫的記憶體資料列
	var sliceNewUserRows [][]interface{}
	// 定義寫入 userauth database 使用者資料的 SQL 字串
	sliceInsertUsrColNames := []string{
		"categoryid",
		"userid",
		"username",
		"pwhash",
		"cancelflag",
		"createopid",
		"modifyopid",
		"createdatetime",
		"modifydatetime"}

	// 先取得當地現在時間
	tNowLocal := time.Now().Local()
	// 轉換 sliceUsersWillBeAdded 成 sliceNewUserRows
	for _, rowUser := range sliceUsersWillBeAdded {
		// 計算 user pass salted hash
		byteArrayPwSaltedHash, errCalHash := facilities.CalHashOfSaltedPw(rowUser.Password)
		if errCalHash != nil {
			return 0, fiber.NewError(fiber.StatusInternalServerError, "系統錯誤：無法正確計算 pass hash, "+errCalHash.Error())
		}

		sliceNewUserRows = append(sliceNewUserRows, []interface{}{
			rowUser.UserCategoryID,
			rowUser.UserID,
			rowUser.UserName,
			string(byteArrayPwSaltedHash),
			0,
			sCreateOpID,
			sCreateOpID,
			tNowLocal,
			tNowLocal,
		})
	}

	// 塞資料進 DB，這裡要注意，塞密碼的時候要轉換成 bcrypt hash 。
	iCopyCnt, errBatchInsert := tx.CopyFrom(
		ctxbg,
		pgx.Identifier{"users", "usersecret"},
		sliceInsertUsrColNames,
		pgx.CopyFromRows(sliceNewUserRows))

	if errBatchInsert != nil {
		return iCopyCnt, errBatchInsert
	}

	return iCopyCnt, nil
}

func GetUserFromPg(sUserIDBeQryed string) (*model.UserDataNoPwHash, error) {
	// 資料庫回傳的一定是一個 slice of rows
	// 所以一定要預產生一個 slice 準備裝回傳的資料。(即使可能根本沒有資料。)
	var sliceUsers []model.UserDataNoPwHash
	// 做資料庫連線準備
	ctxbg := context.Background()
	conn := GetConn(facilities.GlobalQryPool)
	defer conn.Release()

	sSQLGetUser := `
		select
			categoryid,
			userid,
			username,
			cancelflag,
			createopid,
			modifyopid,
			createdatetime,
			modifydatetime
		from
			users.usersecret where userid = $1`
	rowResult, errQry := conn.Query(ctxbg, sSQLGetUser, sUserIDBeQryed)
	defer rowResult.Close()
	if errQry != nil {
		return nil, errQry
	}
	//開始一筆筆把資料轉成物件塞進預生成的 slice 中
	for rowResult.Next() {
		objTmpUser := model.UserDataNoPwHash{}
		errRowScan := rowResult.Scan(
			&objTmpUser.IntUserCatID,
			&objTmpUser.StrUserID,
			&objTmpUser.StrUserName,
			&objTmpUser.IntIfCancel,
			&objTmpUser.StrCreateOpID,
			&objTmpUser.StrModifyOpID,
			&objTmpUser.TimeCreateTimestamp,
			&objTmpUser.TimeModifyTimestamp)
		if errRowScan != nil {
			return nil, errRowScan
		}
		sliceUsers = append(sliceUsers, objTmpUser)
	}
	if len(sliceUsers) != 1 {
		return nil, errors.New(fmt.Sprint("資料庫相關：可能無資料或是資料不只一筆，請仔細檢查欲查詢的使用者代碼。"))
	}

	return &sliceUsers[0], nil
}

func UpdUser(ctxbg context.Context, tx pgx.Tx, objUserBeUpded model.UserDataForUpd) error {
	// 定義寫入 userauth database 使用者資料的 SQL 字串
	sSQLUpdUsr := `
	UPDATE users.usersecret
		set categoryid=$1, username=$2, pwhash=$3, cancelflag=$4, modifyopid=$5, modifydatetime=$6
	WHERE userid=$7`

	// 先取得當地現在時間
	tNowLocal := time.Now().Local()
	// 塞資料進 DB，這裡要注意，塞密碼的時候要轉換成 SHA256 hash 。
	_, errUpdUser := tx.Exec(ctxbg, sSQLUpdUsr,
		objUserBeUpded.IntUserCatID, objUserBeUpded.StrUserName, objUserBeUpded.StrPwHash,
		objUserBeUpded.IntIfCancel, objUserBeUpded.StrModifyOpID, tNowLocal,
		objUserBeUpded.StrUserID)
	if errUpdUser != nil {
		return errUpdUser
	}

	return nil
}
