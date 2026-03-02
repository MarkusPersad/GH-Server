package response

import (
	"GH-Server/internal/model"
)


type UserInfoResponse struct {
	UUID string`json:"uuid"`
	UserName string `json:"userName"`
	Email string `json:"email"`
	Avatar string `json:"avatar"`
	Role string `json:"role"`
}

type SearchResponse struct {
	 User      model.User `json:"user"`
	Group  model.Group `json:"group"`
}