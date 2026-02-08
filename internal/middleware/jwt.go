package middleware

import (
	"GH-Server/internal/database"
	"GH-Server/pkg/exceptions"
	"GH-Server/pkg/utils"
	"GH-Server/pkg/zaplog"
	"errors"
	"fmt"
	"os"
	"strconv"

	jwtware "github.com/gofiber/contrib/v3/jwt"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/extractors"
	"github.com/golang-jwt/jwt/v5"
)

var (
	jwtSecret = os.Getenv("JWT_SECRET")
	jwtexpir,_ = strconv.Atoi(os.Getenv("JWT_EXPIRE"))
	jwtrefresh,_ = strconv.Atoi(os.Getenv("JWT_REFRESH"))
	appName = os.Getenv("APP_NAME")
	access = "access:"
	refresh = "refresh:"
	blackList = "blacklist:"
	refreshHeader = "X-Refresh"
	accessHeader = "Authorization"
)

func JwtErrorHandler(ctx fiber.Ctx,err error) error{
	if errors.Is(err,extractors.ErrNotFound)||errors.Is(err,jwt.ErrTokenInvalidClaims){
		return exceptions.ErrInvalidToken
	}
	if errors.Is(err,jwt.ErrTokenExpired){
		return exceptions.ErrTokenExpired
	}
	zaplog.Zap.Error(fmt.Sprintf("Token Error:%v",err))
	return exceptions.ErrInternalServerError
}

// JwtSuccessHandler 是一个JWT认证成功的处理函数，用于验证用户是否被加入黑名单
// 该函数会从JWT令牌中获取用户主题(Subject)，然后检查该用户是否在Redis黑名单中
// 如果用户在黑名单中，则返回账户锁定错误；否则继续执行下一个中间件
//
// 参数:
//   ctx: Fiber框架的上下文对象，包含请求信息和应用状态
//
// 返回值:
//   error: 如果出现错误则返回相应错误，正常情况下调用ctx.Next()继续执行
func JwtSuccessHandler(ctx fiber.Ctx) error {
	// 从JWT上下文中获取用户主题(Subject)
	sub,err := jwtware.FromContext(ctx).Claims.GetSubject()
	if err != nil {
		zaplog.Zap.Error(fmt.Sprintf("Get Subject Error:%v",err))
		return exceptions.ErrInvalidToken
	}
	// 检查数据库实例是否存在并获取Redis客户端
	if databaseInstance,ok := ctx.App().State().MustGet(database.STATENAME).(database.Service);ok{
		// 检查用户是否在Redis黑名单中
		if cmdBool,err := databaseInstance.GetRedisClient().SIsMember(ctx,fmt.Sprintf("%s%s",blackList,appName),sub).Result(); err != nil {
			zaplog.Zap.Error(fmt.Sprintf("Redis SIsMember Error:%v",err))
			return err
		} else {
			// 如果用户在黑名单中，返回账户锁定错误
			if cmdBool {
				return exceptions.ErrAccountLocked
			}
		}
	}
	// 用户未在黑名单中，继续执行后续中间件
	return ctx.Next()
}

// ... existing code ...

// RefreshTokenHandler 处理刷新令牌的请求，验证现有刷新令牌并生成新的访问和刷新令牌
// 该函数执行以下步骤：
// 1. 从请求头中提取刷新令牌
// 2. 解析并验证刷新令牌
// 3. 验证用户是否在黑名单中
// 4. 检查刷新令牌是否与存储的令牌匹配
// 5. 删除旧的刷新令牌并创建新的访问和刷新令牌
//
// 参数:
//   ctx: Fiber框架的上下文对象，包含请求信息和应用状态
//
// 返回值:
//   error: 如果出现错误则返回相应错误，成功时返回fiber.StatusOK
func RefreshTokenHandler(ctx fiber.Ctx) error {
	// 从请求头中提取刷新令牌
	tokenStr,err := extractors.FromHeader(refreshHeader).Extract(ctx)
	if err != nil {
		zaplog.Zap.Error(fmt.Sprintf("Extract Refresh Token Error:%v",err))
		return exceptions.ErrInvalidToken
	}
	// 解析刷新令牌
	token,err := utils.ParseRefreshToken(tokenStr)
	if err != nil {
		return err
	}
	// 获取刷新令牌声明
	refreshClaim,ok := token.Claims.(*utils.RefreshClaims)
	if !ok {
		zaplog.Zap.Error("Get RefreshClaim Error")
		return exceptions.ErrInvalidToken
	}
	// 获取数据库实例
	databaseInstance,ok := ctx.App().State().MustGet(database.STATENAME).(database.Service)
	if !ok {
		zaplog.Zap.Error("Database not initialized")
		return exceptions.ErrInternalServerError
	}
	// 检查用户是否被锁
	if cmdBool,err := databaseInstance.GetRedisClient().SIsMember(ctx,fmt.Sprintf("%s%s",blackList,appName),refreshClaim.Subject).Result(); err != nil {
		zaplog.Zap.Error(fmt.Sprintf("Redis SIsMember Error:%v",err))
		return exceptions.ErrInternalServerError
	} else {
		if cmdBool {
			return exceptions.ErrAccountLocked
		}
	}
	// 检查刷新令牌是否是旧的
	if oldRefresh,err := databaseInstance.GetRedisClient().Get(ctx,fmt.Sprintf("%s%s",refresh,refreshClaim.Subject)).Result(); err != nil {
		zaplog.Zap.Error(fmt.Sprintf("Redis Get Error:%v",err))
		return exceptions.ErrInternalServerError
	} else {
		if oldRefresh != tokenStr {
			return exceptions.ErrInvalidToken
		}
	}
	// 删除旧的刷新令牌
	if err := databaseInstance.GetRedisClient().Del(ctx,fmt.Sprintf("%s%s",refresh,refreshClaim.Subject)).Err(); err != nil {
		zaplog.Zap.Error(fmt.Sprintf("Redis Del Error:%v",err))
		return  exceptions.ErrInternalServerError
	}
	// 创建新的刷新令牌
	newTokenStr,err :=utils.CreateRefreshToken(ctx,databaseInstance.GetRedisClient(),refreshClaim.Subject,refreshClaim.Role)
	if err != nil {
		return err
	}
	// 创建新的访问令牌
	newAccessTokenStr,err := utils.CreateAccessToken(refreshClaim.Subject)
	if err != nil {
		return err
	}
	ctx.Set(refreshHeader,newTokenStr)
	ctx.Set(accessHeader,newAccessTokenStr)
	return ctx.SendStatus(fiber.StatusOK)
}

// CheckRole 创建一个检查用户角色权限的中间件处理器
// 该中间件会验证当前用户的刷新令牌中的角色是否与指定的角色匹配
// 如果角色不匹配，则返回权限不足错误
// 
// 参数:
//   - ctx: Fiber上下文对象（此参数实际上在函数签名中可能不需要，因为返回的Handler会接收上下文）
//   - role: 需要检查的目标角色字符串
//
// 返回值:
//   - fiber.Handler: 一个Fiber中间件处理函数，用于验证用户角色权限
func CheckRole(role string) fiber.Handler {
	return  func (ctx fiber.Ctx) error {
		// 获取访问令牌中的主题信息（通常是用户ID）
		sub,err := jwtware.FromContext(ctx).Claims.GetSubject()
		if err != nil {
			zaplog.Zap.Error(fmt.Sprintf("Get Subject Error:%v",err))
			return err
		}
		// 获取数据库实例
		databaseInstance,ok := ctx.App().State().MustGet(database.STATENAME).(database.Service)
		if !ok {
			zaplog.Zap.Error("Database not initialized")
			return exceptions.ErrInternalServerError
		}
		// 获取刷新令牌声明
		refreshClaim := new(utils.RefreshClaims)
		if tokenStr,err := databaseInstance.GetRedisClient().Get(ctx,fmt.Sprintf("%s%s",refresh,sub)).Result(); err != nil {
			zaplog.Zap.Error(fmt.Sprintf("Redis Get Error:%v",err))
			return exceptions.ErrInternalServerError
		} else {
			token,err := utils.ParseRefreshToken(tokenStr)
			if err != nil {
				return err
			}
			refreshClaim,ok = token.Claims.(*utils.RefreshClaims)
			if !ok {
				zaplog.Zap.Error("Get RefreshClaim Error")
				return exceptions.ErrInvalidToken
			}
			// 检查角色
			if refreshClaim.Role != role {
				return exceptions.ErrInsufficientPermissions
			}
		}

		return ctx.Next()
	}
}

// Tick 将指定用户ID加入JWT黑名单，使其令牌失效
// 该函数从请求参数中提取用户ID，并将其添加到Redis集合中作为黑名单处理
// 
// 参数:
//   ctx - Fiber上下文对象，用于处理HTTP请求和响应
//
// 返回值:
//   error - 执行过程中可能出现的错误，如fiber.StatusOK表示成功，
//           其他错误包括ErrBadRequest或ErrInternalServerError
func Tick(ctx fiber.Ctx) error {
	// 从路径参数中提取用户ID
	userId,err := extractors.FromParam("UserID").Extract(ctx)
	if err != nil {
		zaplog.Zap.Error(fmt.Sprintf("Extract UserID Error:%v",err))
		return exceptions.ErrBadRequest
	}
	// 获取数据库实例
	databaseInstance,ok := ctx.App().State().MustGet(database.STATENAME).(database.Service)
	if !ok {
		zaplog.Zap.Error("Database not initialized")
		return exceptions.ErrInternalServerError
	}
	// 将用户ID添加到Redis黑名单集合中
	if err := databaseInstance.GetRedisClient().SAdd(ctx,fmt.Sprintf("%s%s",blackList,appName),userId).Err(); err != nil {
		zaplog.Zap.Error(fmt.Sprintf("Redis SAdd Error:%v",err))
		return exceptions.ErrInternalServerError
	}
	return ctx.SendStatus(fiber.StatusOK)
}