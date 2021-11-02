package model

type ValidationIPData struct {
	FromIP string `json:"FromIP"`
}

type ResponseValidationInfo struct {
	UserValid   bool   `json:"UserValid"`
	ResponseMsg string `json:"ResponseMsg"`
	UserID      string `json:"UserID"`
}

type Token struct {
	Token string `json:"token"`
}

type NewUserCredentials struct {
	// -1: 人員管理者,
	// 0: API,
	// 1: 一般使用者
	UserCategoryID int32  `json:"iUserCatID,int32"`
	UserID         string `json:"sUserID"`
	UserName       string `json:"sUserName,omitempty"`
	Password       string `json:"sPassword"`
}

type UserCredentialsReqForUpd struct {
	NewUserCredentials
	IntIfCancel int `json:"iIfCancel"`
}

type UserIDForQry struct {
	UserID string `json:"sUserID"`
}
