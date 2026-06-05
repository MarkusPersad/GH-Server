package handler

import (
	"GH-Server/internal/database"
	"GH-Server/pkg/exceptions"
	"GH-Server/pkg/request"
	"GH-Server/pkg/response"
	"GH-Server/pkg/utils"
	"GH-Server/pkg/zaplog"
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v3"
)



func  SendVerifyMail(ctx fiber.Ctx) error {
	mailVerify := new(request.UserMailVerifyRequest)
	if err := ctx.Bind().Body(mailVerify); err != nil {
		zaplog.Zap.Error(fmt.Sprintf("bind body failed: %v", err))
		return exceptions.ErrBadRequest
	}
	if err := ctx.App().State().MustGet(utils.ValidatorSTATENAME).(*utils.StructValidator).Validate(mailVerify); err != nil {
		return exceptions.ErrInvalidParameters
	}
	if err := ctx.App().State().MustGet(database.STATENAME).(database.Service).SendVerifyCode(mailVerify, ctx); err != nil {
		return err
	}
	return ctx.Status(http.StatusOK).JSON(response.Success("发送成功", nil))
}
func  UserRegister(ctx fiber.Ctx) error {
	register := new(request.UserRegisterRequest)
	if err := ctx.Bind().Body(register); err != nil {
		zaplog.Zap.Error(fmt.Sprintf("bind body failed: %v", err))
		return exceptions.ErrBadRequest
	}
	if err := ctx.App().State().MustGet(utils.ValidatorSTATENAME).(*utils.StructValidator).Validate(register); err != nil {
		return exceptions.ErrInvalidParameters
	}
	if err := ctx.App().State().MustGet(database.STATENAME).(database.Service).Register(register, ctx); err != nil {
		return err
	}
	return ctx.Status(http.StatusOK).JSON(response.Success("注册成功", nil))
}

func  UserLogin(ctx fiber.Ctx) error {
	login := new(request.UserLoginRequest)
	if err := ctx.Bind().Body(login); err != nil {
		zaplog.Zap.Error(fmt.Sprintf("bind body failed: %v", err))
		return exceptions.ErrBadRequest
	}
	if err := ctx.App().State().MustGet(utils.ValidatorSTATENAME).(*utils.StructValidator).Validate(login); err != nil {
		return exceptions.ErrInvalidParameters
	}
	if userInfo, err := ctx.App().State().MustGet(database.STATENAME).(database.Service).Login(login, ctx); err != nil {
		return err
	} else {
		return ctx.Status(http.StatusOK).JSON(response.Success("登录成功", userInfo))
	}
}
func  UserLogout(ctx fiber.Ctx) error {
	if err := ctx.App().State().MustGet(database.STATENAME).(database.Service).Logout(ctx); err != nil {
		return err
	}
	return ctx.Status(http.StatusOK).JSON(response.Success("登出成功", nil))
}

func  UserUploadAvatar(ctx fiber.Ctx) error {
	request := new(request.UserUploadAvatarRequest)
	if err := ctx.Bind().Body(request);err != nil {
		zaplog.Zap.Error(fmt.Sprintf("bind body failed: %v", err))
		return exceptions.ErrBadRequest
	}
	if err := ctx.App().State().MustGet(utils.ValidatorSTATENAME).(*utils.StructValidator).Validate(request); err != nil {
		return exceptions.ErrInvalidParameters
	}
	if err := ctx.App().State().MustGet(database.STATENAME).(database.Service).UploadAvatar(ctx,request);err != nil{
		return err
	}
	return ctx.Status(http.StatusOK).JSON(response.Success("上传成功", nil))
}

func GetUserDetails(ctx fiber.Ctx) error {
	searchinfo := ctx.Params("searchinfo")
	if searchinfo == "" {
		return exceptions.ErrBadRequest
	}
	userInfo, err := ctx.App().State().MustGet(database.STATENAME).(database.Service).GetUserDetails(ctx, searchinfo)
	if err != nil {
		return err
	}
	return ctx.Status(http.StatusOK).JSON(response.Success("获取用户信息成功", userInfo))
}