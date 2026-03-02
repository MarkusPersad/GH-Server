package exceptions

import "net/http"

// Exception 错误结构体
type Exception struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// Error 实现error接口
func (err *Exception) Error() string {
	return err.Msg
}

// NewException 创建新的错误实例
func NewException(code int, msg string) *Exception {
	return &Exception{
		Code: code,
		Msg:  msg,
	}
}

// 定义常用的HTTP状态码错误
var (
	// 4xx 客户端错误
	ErrBadRequest          = NewException(http.StatusBadRequest, "请求参数错误")
	ErrUnauthorized        = NewException(http.StatusUnauthorized, "未授权访问")
	ErrForbidden           = NewException(http.StatusForbidden, "禁止访问")
	ErrNotFound            = NewException(http.StatusNotFound, "资源不存在")
	ErrMethodNotAllowed    = NewException(http.StatusMethodNotAllowed, "方法不被允许")
	ErrRequestTimeout      = NewException(http.StatusRequestTimeout, "请求超时")
	ErrConflict            = NewException(http.StatusConflict, "资源冲突")
	ErrGone                = NewException(http.StatusGone, "资源已失效")
	ErrUnprocessableEntity = NewException(http.StatusUnprocessableEntity, "请求格式正确但语义错误")

	// 5xx 服务器错误
	ErrInternalServerError = NewException(http.StatusInternalServerError, "服务器内部错误")
	ErrNotImplemented      = NewException(http.StatusNotImplemented, "功能未实现")
	ErrBadGateway          = NewException(http.StatusBadGateway, "网关错误")
	ErrServiceUnavailable  = NewException(http.StatusServiceUnavailable, "服务暂时不可用")
	ErrGatewayTimeout      = NewException(http.StatusGatewayTimeout, "网关超时")

	// 自定义业务错误 (1000+)
	ErrOperationIllegal       = NewException(1000, "操作非法")
	ErrUserNotFound            = NewException(1001, "用户不存在")
	ErrUserAlreadyExists       = NewException(1002, "用户已存在")
	ErrInvalidCredentials      = NewException(1003, "用户名或密码错误")
	ErrInvalidToken            = NewException(1004, "无效的认证令牌")
	ErrTokenExpired            = NewException(1005, "认证令牌已过期")
	ErrInsufficientPermissions = NewException(1006, "权限不足")
	ErrInvalidParameters       = NewException(1007, "参数无效")
	ErrDatabaseError           = NewException(1008, "数据库操作失败")
	ErrEmailAlreadyExists      = NewException(1009, "邮箱已被注册")
	ErrUsernameAlreadyExists   = NewException(1010, "用户名已被占用")
	ErrInvalidVerificationCode = NewException(1011, "验证码错误")
	ErrVerificationCodeExpired = NewException(1012, "验证码已过期")
	ErrVerificationCodeSent    = NewException(1013, "验证码已发送")
	ErrAccountLogined          = NewException(1014, "账户已登录")
	ErrAccountLocked           = NewException(1015, "账户已锁定")
	ErrPasswordIncorrect       = NewException(1016, "密码错误")
	ErrFriendAlreadyExists     = NewException(1017, "好友已存在")
	ErrGroupAlreadyExists      = NewException(1018, "群组已存在")
)
