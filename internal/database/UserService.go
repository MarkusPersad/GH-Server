package database

import (
	"GH-Server/internal/middleware"
	"GH-Server/internal/model"
	"GH-Server/pkg/exceptions"
	"GH-Server/pkg/request"
	"GH-Server/pkg/response"
	"GH-Server/pkg/utils"
	"GH-Server/pkg/zaplog"
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	verifyCodePrefix = "VerifyCode:"
	loginPrefix = "Login:"
	jwtExpire,_ = strconv.Atoi(os.Getenv("JWT_EXPIRE"))
)

type UserService interface {
	Register(register *request.UserRegisterRequest, ctx context.Context) error
	SendVerifyCode(emailVerify *request.UserMailVerifyRequest, ctx context.Context) error
	Login(login *request.UserLoginRequest,ctx context.Context) (*response.UserInfoResponse,error)
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

		//检测用户是否存在
		if _, err := gorm.G[model.User](s.db).Where("email = ?", register.Email).Or("user_name = ?",register.UserName).First(ctx); err == nil {
			return exceptions.ErrUserAlreadyExists
		}

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

func(s *service) Login(login *request.UserLoginRequest,ctx context.Context) (*response.UserInfoResponse,error) {
	userInfo := new(response.UserInfoResponse)
	err := s.db.Transaction(func(tx *gorm.DB) error {
		user := new(model.User)
		if userModel,err := gorm.G[model.User](tx).Where("email = ?",login.Email).First(ctx);err != nil {
			zaplog.Zap.Error(fmt.Sprintf("Select model.User Failed:%v",err))
			if errors.Is(err,gorm.ErrRecordNotFound){
				return exceptions.ErrUserNotFound
			}
			return err
		} else {
			user = &userModel
		}
		if err := utils.ComparedWithPassword(user.Password,login.Password);err != nil {
			zaplog.Zap.Error(fmt.Sprintf("Password Compare Failed:%v",err))
			return err
		}
		if token,err := middleware.CreateJwtToken(user);err != nil {
			return err
		}else {
			userInfo.Token = token
			if err = s.rdb.SetNX(ctx,fmt.Sprintf("%s%s",loginPrefix,user.UUID),userInfo.Token,time.Duration(jwtExpire)*24*time.Hour).Err();err != nil {
				zaplog.Zap.Error(fmt.Sprintf("set redis failed: %v", err))
				return err
			}
		}
		if _,err := gorm.G[model.User](tx).Where("email = ?",user.Email).Updates(ctx,model.User{
			LastLoginAt: time.Now(),
			Status: 1,
		});err != nil {
			zaplog.Zap.Error(fmt.Sprintf("Update user failed: %v", err))
			return err
		}
		userInfo.Avatar = user.Avatar
		userInfo.Email = user.Email
		userInfo.Role = user.Role
		userInfo.UUID = user.UUID
		userInfo.UserName = user.UserName		
		return nil
	})
	return userInfo,err
}