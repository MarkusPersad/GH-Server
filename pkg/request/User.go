package request

type UserRegisterRequest struct {
	UserName string `json:"userName" validate:"required,min=5,max=20"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Avatar   string `json:"avatar"`
}
