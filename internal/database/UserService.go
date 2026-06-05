package database

import (
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

	jwtware "github.com/gofiber/contrib/v3/jwt"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	_ "github.com/joho/godotenv/autoload"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var (
	verifyCodePrefix = "VerifyCode:"
	loginPrefix = "Login:"
	appName = os.Getenv("APP_NAME")
	blackList = "blacklist:"
	refresh = "refresh:"
	refreshHeader = "X-Refresh"
	accessHeader = "Authorization"
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
	UploadAvatar(ctx fiber.Ctx,request *request.UserUploadAvatarRequest) error
	GetUserDetails(ctx fiber.Ctx,searchinfo string) (*response.UserInfoResponse,error)
	
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
		if errors.Is(err,redis.Nil) {
			return exceptions.ErrVerificationCodeExpired
		}
		return err
	}
	userUUID := uuid.NewSHA1(uuid.NameSpaceX500,[]byte(register.UserName))
	hashedPassword,err := utils.GenerateFromPassword(register.Password)
	if err != nil {
		return err
	}
	if err := gorm.G[model.User](tx).Create(ctx,&model.User{
		UUID: userUUID.String(),
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
	verifiedKey := fmt.Sprintf("%s%s", verifyCodePrefix, emailVerify.Email)
	if existsCount,err := s.rdb.Exists(ctx,verifiedKey).Result(); err == nil {
		if existsCount >0 {
			return  exceptions.ErrVerificationCodeSent
		}
	} else {
		zaplog.Zap.Error(fmt.Sprintf("redis exists failed: %v", err))
		return err
	}
	code := rand.Intn(900000) + 100000
	if err := utils.VerifyMailSend(emailVerify.Email, emailVerify.UserName, strconv.Itoa(code)); err != nil {
		return err
	}
	if err := s.rdb.Set(ctx,verifiedKey , code, 5*time.Minute).Err(); err != nil {
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
		if user.Status == 1  {
			return exceptions.ErrAccountLogined
		}
		if err := CheckLogin(ctx,user.UUID,false); err != nil {
			return err
		}
		if err := utils.ComparedWithPassword(user.Password,login.Password);err != nil {
			return exceptions.ErrPasswordIncorrect
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
		if token,err := utils.CreateAccessToken(user.UUID); err != nil {
			zaplog.Zap.Error(fmt.Sprintf("Token String Generated Failed:%v",err))
			return err
		} else {
			ctx.Set(accessHeader,token)
		}
		if token,err := utils.CreateRefreshToken(ctx,s.rdb,user.UUID,user.Role); err != nil {
			return err
		} else {
			ctx.Set(refreshHeader,token)
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
	accountID,err := jwtware.FromContext(ctx).Claims.GetSubject()
		if err != nil {
			zaplog.Zap.Error(fmt.Sprintf("Get accountID failed: %v", err))
			return err
		}
		if err := CheckLogin(ctx,accountID,true); err != nil {
			zaplog.Zap.Error(fmt.Sprintf("CheckLogin failed: %v", err))
			return err
		}
	return s.gdb.Transaction(func(tx *gorm.DB) error {
		if _,err := gorm.G[model.User](tx).Where("uuid = ? ",accountID).Select("status","last_logout_at","version").Updates(ctx,model.User{
			LastLogoutAt: time.Now(),
			Status: 0,
		});err != nil {
			zaplog.Zap.Error(fmt.Sprintf("Update user failed: %v", err))
			return err
		}
		if err:= s.rdb.Del(ctx,fmt.Sprintf("%s%s",refresh,accountID)).Err(); err != nil {
			zaplog.Zap.Error(fmt.Sprintf("Redis Del Error:%v",err))
			return err
		}
		return nil
	})
}
func(s *service)UploadAvatar(ctx fiber.Ctx,request *request.UserUploadAvatarRequest) error {
	accountID,err := jwtware.FromContext(ctx).Claims.GetSubject()
	if err != nil {
		zaplog.Zap.Error(fmt.Sprintf("Get accountID failed: %v", err))
		return err
	}
	if err := CheckLogin(ctx,accountID,true); err != nil {
			zaplog.Zap.Error(fmt.Sprintf("CheckLogin failed: %v", err))
			return err
		}
	return s.gdb.Transaction(func(tx *gorm.DB) error {
		if _,err := gorm.G[model.User](tx).Where("uuid = ? ",accountID).Select("avatar","version").Updates(ctx,model.User{
			Avatar: request.Avatar,
		});err != nil {
			zaplog.Zap.Error(fmt.Sprintf("Update user failed: %v", err))
			return err
		}
		return  nil
	})
}


func (s *service) GetUserDetails(ctx fiber.Ctx,searchinfo string) (*response.UserInfoResponse,error) {
	accountID,err := jwtware.FromContext(ctx).Claims.GetSubject()
	if err != nil {
		zaplog.Zap.Error(fmt.Sprintf("Get accountID failed: %v", err))
		return nil,err
	}
	if err := CheckLogin(ctx,accountID,true); err != nil {
			return nil,err
		}
	userInfo := new(response.UserInfoResponse)
	err = s.gdb.Transaction(func(tx *gorm.DB) error {
		user := new(model.User)

		if usr,err := gorm.G[model.User](tx).
			Where("email = ? OR user_name = ?", searchinfo, searchinfo).
			First(ctx);err != nil {
			if errors.Is(err,gorm.ErrRecordNotFound) {
				zaplog.Zap.Error(fmt.Sprintf("record not found: %v", err))
				return exceptions.ErrUserNotFound
			}
			zaplog.Zap.Error(fmt.Sprintf("select user failed: %v", err))
			return err
		} else {
			user = &usr 
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


func CheckLogin(ctx fiber.Ctx,uuid string,ignore bool) error {
	var err error = nil
	databaseInstance,ok := ctx.App().State().MustGet(STATENAME).(Service)
	if !ok {
		zaplog.Zap.Error("Database not initialized")
		err = exceptions.ErrInternalServerError
	}
	cmdBool,err := databaseInstance.GetRedisClient().SIsMember(ctx,fmt.Sprintf("%s%s",blackList,appName),uuid).Result()
	if err != nil {
		zaplog.Zap.Error(fmt.Sprintf("Redis SIsMember Error:%v",err))
		err = exceptions.ErrInternalServerError
	}
	if cmdBool {
		err = exceptions.ErrAccountLocked
	}
	cmdInt,err := databaseInstance.GetRedisClient().Exists(ctx,fmt.Sprintf("%s%s",refresh,uuid)).Result()
	if err != nil {
		zaplog.Zap.Error(fmt.Sprintf("Redis Exists Error:%v",err))
		err = exceptions.ErrInternalServerError
	}
	if !ignore {
		if cmdInt != 0 {
			err = exceptions.ErrAccountLogined
		}
	}
	
	return err
}