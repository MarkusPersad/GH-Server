package handler

import (
	"GH-Server/pkg/exceptions"
	"GH-Server/pkg/request"
	"GH-Server/pkg/response"
	"GH-Server/pkg/zaplog"
	"fmt"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v3"
)

type UserHandler interface {
	UserRegister(ctx fiber.Ctx) error
	UserLogin(ctx fiber.Ctx) error
	UserLogout(ctx fiber.Ctx) error
	UserUploadAvatar(ctx fiber.Ctx) error
}

func (handler *Handler) SendVerifyMail(ctx fiber.Ctx) error {
	mailVerify := new(request.UserMailVerifyRequest)
	if err := ctx.Bind().Body(mailVerify); err != nil {
		zaplog.Zap.Error(fmt.Sprintf("bind body failed: %v", err))
		return exceptions.ErrBadRequest
	}
	if err := handler.Validate(mailVerify); err != nil {
		return exceptions.ErrInvalidParameters
	}
	if err := handler.SendVerifyCode(mailVerify, ctx); err != nil {
		return err
	}
	return ctx.Status(http.StatusOK).JSON(response.Success("发送成功", nil))
}
func (handler *Handler) UserRegister(ctx fiber.Ctx) error {
	register := new(request.UserRegisterRequest)
	if err := ctx.Bind().Body(register); err != nil {
		zaplog.Zap.Error(fmt.Sprintf("bind body failed: %v", err))
		return exceptions.ErrBadRequest
	}
	if err := handler.Validate(register); err != nil {
		return exceptions.ErrInvalidParameters
	}
	if err := handler.Register(register, ctx); err != nil {
		return err
	}
	return ctx.Status(http.StatusOK).JSON(response.Success("注册成功", nil))
}

func (handler *Handler) UserLogin(ctx fiber.Ctx) error {
	login := new(request.UserLoginRequest)
	if err := ctx.Bind().Body(login); err != nil {
		zaplog.Zap.Error(fmt.Sprintf("bind body failed: %v", err))
		return exceptions.ErrBadRequest
	}
	if err := handler.Validate(login); err != nil {
		return exceptions.ErrInvalidParameters
	}
	if userInfo,err := handler.Login(login,ctx);err != nil {
		return err
	} else {
		return ctx.Status(http.StatusOK).JSON(response.Success("登录成功",userInfo))
	}
}
func (handler *Handler) UserLogout(ctx fiber.Ctx) error {
	if err := handler.Logout(ctx);err != nil {
		return err
	}
	return ctx.Status(http.StatusOK).JSON(response.Success("登出成功",nil))
}

func (handler *Handler) UserUploadAvatar(ctx fiber.Ctx) error {
	header,exists := ctx.GetHeaders()["Email"]
	if !exists || len(header) == 0 {
		zaplog.Zap.Error(fmt.Sprintf("email not found:%v",ctx.GetHeaders()))
		return exceptions.ErrBadRequest
	}
	fileHeader,err := ctx.FormFile("file")
	if err != nil {
		zaplog.Zap.Error(fmt.Sprintf("get file failed: %v", err))
		return exceptions.ErrBadRequest
	}
	file,err := fileHeader.Open()
	if err != nil {
		zaplog.Zap.Error(fmt.Sprintf("open file failed: %v", err))
		return err
	}
	defer file.Close()
	mpo,err := handler.RustFSService.UploadFile(ctx,fmt.Sprintf("avatar/%s",header[0]),file,strings.Split(fileHeader.Filename,".")[1])
	if err != nil {
		zaplog.Zap.Error(fmt.Sprintf("upload file failed: %v", err))
		return err
	}
	if err := handler.Service.UploadAvatar(ctx,header[0],mpo.Location); err != nil {
		return err
	}

	return ctx.Status(http.StatusOK).JSON(response.Success("上传成功",nil))
}
