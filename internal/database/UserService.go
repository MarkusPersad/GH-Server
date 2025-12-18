package database

import (
	"GH-Server/internal/model"
	"GH-Server/pkg/exceptions"
	"GH-Server/pkg/request"
	"GH-Server/pkg/utils"
	"GH-Server/pkg/zaplog"
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	verifyCodePrefix = "VerifyCode:"
)

type UserService interface {
	Register(register *request.UserRegisterRequest, ctx context.Context) error
	SendVerifyCode(emailVerify *request.UserMailVerifyRequest, ctx context.Context) error
}

// Register 创建新用户并将其信息存储到数据库中
// 该函数在一个数据库事务中执行，确保数据一致性
// 参数:
//   - register: 包含用户注册信息的结构体指针，包括用户名、邮箱、密码和头像
//   - ctx: 上下文对象，用于控制请求的生命周期
//
// 返回值:
//   - error: 如果在创建用户过程中出现错误则返回错误信息，否则返回nil
func (s *service) Register(register *request.UserRegisterRequest, ctx context.Context) error {
	// 在数据库事务中执行用户注册逻辑
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 检查验证码
		if verifyCode, err := s.rdb.Get(ctx, fmt.Sprintf("%s%s", verifyCodePrefix, register.Email)).Result(); err == nil {
			if register.VerifyCode == verifyCode {
				if err = s.rdb.Del(ctx, fmt.Sprintf("%s%s", verifyCodePrefix, register.Email)).Err(); err != nil {
					zaplog.Zap.Error(fmt.Sprintf("del redis failed: %v", err))
					return err
				}
			} else {
				return exceptions.ErrInvalidVerificationCode
			}
		} else {
			return err
		}
		// 基于用户名生成唯一的 UUID 作为用户标识
		userUuid := uuid.NewSHA1(uuid.NameSpaceDNS, []byte(register.UserName))

		// 对用户密码进行哈希处理以保证安全性
		hashedPassword, err := utils.GenerateFromPassword(register.Password)
		if err != nil {
			return err
		}

		// 将用户信息保存到数据库中
		err = gorm.G[model.User](tx).Create(ctx, &model.User{
			UUID:     userUuid,
			UserName: register.UserName,
			Email:    register.Email,
			Password: hashedPassword,
			Avatar:   register.Avatar,
		})
		if err != nil {
			zaplog.Zap.Error(fmt.Sprintf("create user failed: %v", err))
			return err
		}
		return nil
	})
}

func (s *service) SendVerifyCode(emailVerify *request.UserMailVerifyRequest, ctx context.Context) error {
	code := rand.Intn(900000) + 100000
	if err := utils.VerifyMailSend(emailVerify.Email, emailVerify.UserName, strconv.Itoa(code)); err != nil {
		return err
	}
	if err := s.rdb.SetNX(ctx, fmt.Sprintf("%s%s", verifyCodePrefix, emailVerify.Email), code, 5*time.Minute).Err(); err != nil {
		zaplog.Zap.Error(fmt.Sprintf("set redis failed: %v", err))
		return err
	}
	return nil
}
