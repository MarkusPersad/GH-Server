package database

import (
	fibersatoken "GH-Server/internal/middleware/fiber-sa-token"
	"GH-Server/internal/model"
	"GH-Server/pkg/exceptions"
	"GH-Server/pkg/request"
	"GH-Server/pkg/response"
	"GH-Server/pkg/utils"
	"GH-Server/pkg/zaplog"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	_ "github.com/joho/godotenv/autoload"
	"gorm.io/gorm"
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
	Register(register *request.UserRegisterRequest, ctx fiber.Ctx) error
	SendVerifyCode(emailVerify *request.UserMailVerifyRequest, ctx fiber.Ctx) error
	Login(login *request.UserLoginRequest,ctx fiber.Ctx) (*response.UserInfoResponse,error)
	Logout(ctx fiber.Ctx) error
	UploadAvatar(ctx fiber.Ctx,email string,avatar string) error
}

// Register 创建新用户并将其信息存储到数据库中
// 该函数在一个数据库事务中执行，确保数据一致性
// 参数:
//   - register: 包含用户注册信息的结构体指针，包括用户名、邮箱、密码和头像
//   - ctx: 上下文对象，用于控制请求的生命周期
//
// 返回值:
//   - error: 如果在创建用户过程中出现错误则返回错误信息，否则返回nil
func (s *service) Register(register *request.UserRegisterRequest, ctx fiber.Ctx) error {
	return s.gdb.Transaction(func(tx *gorm.DB) error {
		if _,err := gorm.G[model.User](tx).Where("email = ?",register.Email).First(ctx); err == nil {
			return exceptions.ErrUserAlreadyExists
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
	userUUID := uuid.NewSHA1(uuid.NameSpaceX500,[]byte(register.UserName))
	hashedPassword,err := utils.GenerateFromPassword(register.Password)
	if err != nil {
		return err
	}
	if err := gorm.G[model.User](tx).Create(ctx,&model.User{
		UUID: userUUID,
		UserName: register.UserName,
		Email: register.Email,
		Password: hashedPassword,
	}); err != nil {
		zaplog.Zap.Error(fmt.Sprintf("insert user failed: %v", err))
		return err
	}
		return nil
	})
}

func (s *service) SendVerifyCode(emailVerify *request.UserMailVerifyRequest, ctx fiber.Ctx) error {
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

func(s *service) Login(login *request.UserLoginRequest,ctx fiber.Ctx) (*response.UserInfoResponse,error) {
	userInfo := new(response.UserInfoResponse)
	err := s.gdb.Transaction(func(tx *gorm.DB) error {
		user := new(model.User)
		if usr,err := gorm.G[model.User](tx).Where("email = ?",login.Email).First(ctx);err != nil {
			if errors.Is(err,gorm.ErrRecordNotFound) {
				zaplog.Zap.Error(fmt.Sprintf("record not found: %v", err))
				return exceptions.ErrUserNotFound
			}
			zaplog.Zap.Error(fmt.Sprintf("select user failed: %v", err))
			return err
		} else {
			user = &usr 
		}
		if err := fibersatoken.CheckLogin(user.UUID.String()); err == nil {
			zaplog.Zap.Error(fmt.Sprintf("user %s is already logged in", user.UUID.String()))
			return exceptions.ErrAccountLogined
		} else{
			zaplog.Zap.Info(fmt.Sprintf("checkLogin:%v",err))
		}
		if user.Status == 1  {
			return exceptions.ErrAccountLogined
		}
		if err := utils.ComparedWithPassword(user.Password,login.Password);err != nil {
			return err
		}
		if _,err := gorm.G[model.User](tx).Where("email = ? ",login.Email).Select("status","last_login_at","version").Updates(ctx,
			model.User{
				LastLoginAt: time.Now(),
				Status: 1,
			},
		);err != nil {
			zaplog.Zap.Error(fmt.Sprintf("Update user failed: %v", err))
			return err
		}
		if token,err := fibersatoken.Login(user.UUID.String()); err != nil {
			zaplog.Zap.Error(fmt.Sprintf("Token String Generated Failed:%v",err))
			return err
		} else {
			userInfo.Token = token
		}
		if err := fibersatoken.GetManager().SetRoles(userInfo.Token,[]string{
			user.Role,
		}); err != nil {
			zaplog.Zap.Error(fmt.Sprintf("set roles failed: %v",err))
			return err
		}
		userInfo.UUID = user.UUID
		userInfo.UserName = user.UserName
		userInfo.Avatar = user.Avatar
		userInfo.Email = user.Email
		userInfo.Role = user.Role
		return nil
	})
	return userInfo,err
}


// Logout 使用户退出登录，更新用户状态为离线并从Redis中删除用户的登录信息
// 参数:
//   ctx: Fiber上下文，包含JWT用户信息和请求数据
// 返回值:
//   error: 操作失败时返回错误，成功时返回nil
func(s *service) Logout(ctx fiber.Ctx) error{
	return s.gdb.Transaction(func(tx *gorm.DB) error {
		saCtx,ok := fibersatoken.GetSaToken(ctx)
		if !ok {
			zaplog.Zap.Error("get sa token failed")
			return errors.New("get sa-token failed")
		}
		var accountID string
		if loginID,err := saCtx.GetLoginID();err != nil {
			zaplog.Zap.Error(fmt.Sprintf("get login id failed: %v", err))
			return err
		} else {
			accountID = loginID
		}
		if _,err := gorm.G[model.User](tx).Where("uuid = ? ",accountID).Select("status","last_logout_at","version").Updates(ctx,model.User{
			LastLogoutAt: time.Now(),
			Status: 0,
		});err != nil {
			zaplog.Zap.Error(fmt.Sprintf("Update user failed: %v", err))
			return err
		}
		if err := fibersatoken.Logout(accountID);err != nil {
			zaplog.Zap.Error(fmt.Sprintf("Logout user failed: %v", err))
			return err
		}
		return nil
	})
}
func(s *service)UploadAvatar(ctx fiber.Ctx,email string,avatar string) error {
	return s.gdb.Transaction(func(tx *gorm.DB) error {
		if _,err := gorm.G[model.User](tx).Where("email = ? ",email).First(ctx); err != nil {
			if errors.Is(err,gorm.ErrRecordNotFound) {
				return exceptions.ErrUserNotFound
			}
			zaplog.Zap.Error(fmt.Sprintf("select user failed: %v", err))
			return err
		}
		if _,err := gorm.G[model.User](tx).Where("email = ? ",email).Select("avatar","version").Updates(ctx,model.User{
			Avatar: avatar,
		}); err != nil {
			zaplog.Zap.Error(fmt.Sprintf("update user failed: %v", err))
			return err
		}
		return nil
	})
}