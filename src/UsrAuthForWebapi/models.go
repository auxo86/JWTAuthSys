package UsrAuthForWebapi

type ResponseValidationInfo struct {
	UserValid   bool   `json:"UserValid"`
	ResponseMsg string `json:"ResponseMsg"`
	UserID      string `json:"UserID"`
}
