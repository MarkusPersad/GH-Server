package response

import "github.com/google/uuid"

type UserInfoResponse struct {
	UUID uuid.UUID `json:"uuid"`
	UserName string `json:"userName"`
	Email string `json:"email"`
	Avatar string `json:"avatar"`
	Role uint8 `json:"role"`
	Token string `json:"token"`
}