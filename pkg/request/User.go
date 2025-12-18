package request

type UserMailVerifyRequest struct {
	UserName string `json:"userName" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
}
type UserRegisterRequest struct {
	UserName   string `json:"userName" validate:"required,min=5,max=20"`
	Email      string `json:"email" validate:"required,email"`
	Password   string `json:"password" validate:"required,min=8"`
	VerifyCode string `json:"verifyCode" validate:"required"`
	Avatar     string `json:"avatar"`
}

type UserLoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}
