package database

import (
	fibersatoken "GH-Server/internal/middleware/fiber-sa-token"
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

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	_ "github.com/joho/godotenv/autoload"
)

var (
	verifyCodePrefix = "VerifyCode:"
	loginPrefix = "Login:"
	jwtExpire,_ = strconv.Atoi(os.Getenv("JWT_EXPIRE"))
)
const(
	UserRole = "user"
	AdminRole = "admin"
)

type UserService interface {
	Register(register *request.UserRegisterRequest, ctx context.Context) error
	SendVerifyCode(emailVerify *request.UserMailVerifyRequest, ctx context.Context) error
	Login(login *request.UserLoginRequest,ctx context.Context) (*response.UserInfoResponse,error)
	Logout(ctx fiber.Ctx) error
	UploadAvatar(ctx context.Context,email string,avatar string) error
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
	session := s.xdb.NewSession()
	defer session.Close()
	if err := session.Begin(); err != nil {
		zaplog.Zap.Error(fmt.Sprintf("begin transaction failed: %v", err))
		return err
	}
	account := new(model.Account)
	if total,err := session.Where("email=?",register.Email).Count(account); err != nil {
		zaplog.Zap.Error(fmt.Sprintf("count user failed: %v", err))
		return err
	}else {
		if total > 0 {
			return exceptions.ErrUserAlreadyExists
		}
	}
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
	userUUID := uuid.NewSHA1(uuid.NameSpaceX500,[]byte(register.UserName)).String()
	hashedPassword,err := utils.GenerateFromPassword(register.Password)
	if err != nil {
		return err
	}
	account.UUID = userUUID
	account.UserName = register.UserName
	account.Email = register.Email
	account.Password = hashedPassword
	account.Role = UserRole
	if _,err := session.InsertOne(account); err != nil {
		zaplog.Zap.Error(fmt.Sprintf("insert user failed: %v", err))
		return err
	}
	return session.Commit()
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
	session := s.xdb.NewSession()
	defer session.Close()
	if err := session.Begin();err != nil {
		zaplog.Zap.Error(fmt.Sprintf("begin transaction failed: %v", err))
		return nil,err
	}
	userInfo := new(response.UserInfoResponse)
	account := new(model.Account)
	account.Email = login.Email
	if has,err := session.Get(account);err != nil {
		zaplog.Zap.Error(fmt.Sprintf("select user failed: %v", err))
		return nil,err
	}else {
		if !has { 
			return nil,exceptions.ErrUserNotFound
		}
	}
	if err := utils.ComparedWithPassword(account.Password,login.Password);err != nil {
		return nil,err
	}
	if _,err := session.Cols("status","last_login_at").Update(&model.Account{
		LastLoginAt: time.Now(),
		Status: 1,
	}); err != nil {
		zaplog.Zap.Error(fmt.Sprintf("Update user failed: %v", err))
		return nil,err
	}
	if token,err := fibersatoken.Login(account.UUID); err != nil {
		zaplog.Zap.Error(fmt.Sprintf("Token String Generated Failed:%v",err))
		return nil,err
	} else {
		userInfo.Token = token
	}
	if err := fibersatoken.GetManager().SetRoles(userInfo.Token,[]string{
		account.Role,
	}); err != nil {
		zaplog.Zap.Error(fmt.Sprintf("set roles failed: %v",err))
		return nil,err
	}
	userInfo.Avatar = account.Avatar
	userInfo.Email = account.Email
	userInfo.Role = account.Role
	userInfo.UUID = account.UUID
	userInfo.UserName = account.UserName
	if err := session.Commit(); err != nil {
		zaplog.Zap.Error(fmt.Sprintf("commit transaction failed: %v", err))
		return nil,err
	}
	return userInfo,nil
}


// Logout 使用户退出登录，更新用户状态为离线并从Redis中删除用户的登录信息
// 参数:
//   ctx: Fiber上下文，包含JWT用户信息和请求数据
// 返回值:
//   error: 操作失败时返回错误，成功时返回nil
func(s *service) Logout(ctx fiber.Ctx) error{
	session := s.xdb.NewSession()
	defer session.Close()
	if err := session.Begin();err != nil {
		zaplog.Zap.Error(fmt.Sprintf("begin transaction failed: %v", err))
		return err
	}
	saCtx,ok := fibersatoken.GetSaToken(ctx)
	if !ok {
		zaplog.Zap.Error("get sa token failed")
		return errors.New("get sa token failed")
	}
	var accountID string
	if loginID,err := saCtx.GetLoginID();err != nil {
		zaplog.Zap.Error(fmt.Sprintf("get login id failed: %v", err))
		return err
	} else {
		accountID = loginID
	}
	if _,err := session.Where("uuid=?",accountID).Cols("status","last_logout_at").Update(&model.Account{
		Status: 0,
		LastLogoutAt: time.Now(),
	}); err != nil {
		zaplog.Zap.Error(fmt.Sprintf("Update user failed: %v", err))
		return err
	}
	//TODO satoken 登出
	if err := fibersatoken.Logout(accountID);err != nil {
		zaplog.Zap.Error(fmt.Sprintf("Logout user failed: %v", err))
		return err
	}
	return session.Commit()
}
func(s *service)UploadAvatar(ctx context.Context,email string,avatar string) error {
	session := s.xdb.NewSession()
	defer session.Close()
	if err := session.Begin(); err != nil {
		zaplog.Zap.Error(fmt.Sprintf("Failed to Begin:%v",err))
		return err
	}
	account := new(model.Account)

	if has,err := session.Where("email=?",email).Get(account); err != nil {
		zaplog.Zap.Error(fmt.Sprintf("Failed to Get:%v",err))
		return err
	} else {
		if !has {
			return exceptions.ErrUserNotFound
		}
	}
	if _,err := session.Where("email=?",email).Cols("avatar").Update(&model.Account{
		Version: account.Version,
		Avatar: avatar,
	}); err != nil {
		zaplog.Zap.Error(fmt.Sprintf("Failed to Update:%v",err))
		return err
	}
	return session.Commit()
}