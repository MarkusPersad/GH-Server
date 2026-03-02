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
	UploadAvatar(ctx fiber.Ctx,email string,avatar string) error
	GetUserDetails(ctx fiber.Ctx,searchinfo string) (*response.UserInfoResponse,error)
	Search(ctx fiber.Ctx,searchinfo string) (*response.SearchResponse,error)
	GetUserList(ctx fiber.Ctx) (*[]model.User, error)
	
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
		if err := CheckLogin(ctx,user.UUID); err != nil {
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
	return s.gdb.Transaction(func(tx *gorm.DB) error {
		accountID,err := jwtware.FromContext(ctx).Claims.GetSubject()
		if err != nil {
			zaplog.Zap.Error(fmt.Sprintf("Get accountID failed: %v", err))
			return err
		}
		if err := CheckLogin(ctx,accountID); err != nil {
			return nil
		}
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


func (s *service) GetUserDetails(ctx fiber.Ctx,searchinfo string) (*response.UserInfoResponse,error) {
	accountID,err := jwtware.FromContext(ctx).Claims.GetSubject()
	if err != nil {
		zaplog.Zap.Error(fmt.Sprintf("Get accountID failed: %v", err))
		return nil,err
	}
	if err := CheckLogin(ctx,accountID); err != nil {
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

func (s *service) Search(ctx fiber.Ctx, searchinfo string) (*response.SearchResponse, error) {
	// 提取账户ID并验证登录状态
	accountID, err := jwtware.FromContext(ctx).Claims.GetSubject()
	if err != nil {
		zaplog.Zap.Error(fmt.Sprintf("Get accountID failed: %v", err))
		return nil, err
	}
	if err := CheckLogin(ctx, accountID); err != nil {
		return nil, err
	}

	// 初始化结果容器
	user := new([]model.User)
	group := new([]model.Group)

	// 构造模糊搜索模式
	pattern := fmt.Sprintf("%%%s%%", searchinfo)

	// 执行数据库事务
	err = s.gdb.Transaction(func(tx *gorm.DB) error {
		// 模糊搜索用户
		users, err := gorm.G[model.User](tx).
			Select("uuid", "avatar", "user_name", "email", "role").
			Where("user_name LIKE ? OR email LIKE ?", pattern, pattern).
			Find(ctx)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			zaplog.Zap.Error(fmt.Sprintf("select user failed: %v", err))
			return err
		}
		if err == nil {
			user = &users
		}

		// 模糊搜索群组
		groups, err := gorm.G[model.Group](tx).
			Select("name", "uuid").
			Where("name LIKE ?", pattern).
			Find(ctx)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			zaplog.Zap.Error(fmt.Sprintf("select group failed: %v", err))
			return err
		}
		if err == nil {
			group = &groups
		}

		return nil
	})

	// 构造响应结果
	searchResponse := new(response.SearchResponse)
	searchResponse.User = *user
	searchResponse.Group = *group
	return searchResponse, err
}

// GetUserList 获取当前账户关联的用户列表
// 首先从上下文中的JWT令牌提取账户ID
// 然后检查账户是否已登录
// 如果账户未登录且错误不是ErrAccountLogined，则返回错误
// 否则，查询数据库获取账户的好友列表
// 函数使用事务确保查询期间的数据一致性
//
// 参数:
//   - ctx: 包含请求信息和JWT令牌的Fiber上下文
//
// 返回值:
//   - *[]model.User: 指向用户模型切片的指针，表示用户列表
//   - error: 如果任何步骤失败则返回错误，否则返回nil
func (s *service) GetUserList(ctx fiber.Ctx) (*[]model.User, error) {
	// 从上下文中的JWT令牌提取账户ID
	accountID, err := jwtware.FromContext(ctx).Claims.GetSubject()
	if err != nil {
		zaplog.Zap.Error(fmt.Sprintf("获取账户ID失败: %v", err))
		return nil, err
	}

	// 检查账户是否已登录
	if err := CheckLogin(ctx, accountID); err != nil {
		return nil, err
	}

	// 初始化用户列表
	userList := new([]model.User)

	// 执行数据库事务以获取用户好友
	err = s.gdb.Transaction(func(tx *gorm.DB) error {
		
		return tx.Table("user_friend").
			Select("user.uuid,user.user_name,user.avatar").
			Joins("JOIN user ON user_friend.friend_id = user.uuid").
			Where("(user_friend.user_id = ? OR user_friend.friend_id = ?) AND status = 0",accountID,accountID).
			Scan(userList).Error
	})

	// 返回用户列表和遇到的任何错误
	return userList, err
}

func CheckLogin(ctx fiber.Ctx,uuid string) error {
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
	if cmdInt != 0 {
		err = exceptions.ErrAccountLogined
	}
	return err
}