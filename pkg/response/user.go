package response


type UserInfoResponse struct {
	UUID string`json:"uuid"`
	UserName string `json:"userName"`
	Email string `json:"email"`
	Avatar string `json:"avatar"`
	Role string `json:"role"`
}