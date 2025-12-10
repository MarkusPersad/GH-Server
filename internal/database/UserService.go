package database

import (
	"GH-Server/pkg/request"
)

type UserService interface {
	Register(register request.UserRegisterRequest) error
}

func (s *service) Register(register request.UserRegisterRequest) error {

	return nil

}
