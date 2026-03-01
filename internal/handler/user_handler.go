package handler

import (
	"GH-Server/internal/database"
	"GH-Server/internal/fileServer"
	"GH-Server/pkg/exceptions"
	"GH-Server/pkg/request"
	"GH-Server/pkg/response"
	"GH-Server/pkg/utils"
	"GH-Server/pkg/zaplog"
	"bytes"
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
	emailHeader, exists := ctx.GetHeaders()["Email"]
	if !exists || len(emailHeader) == 0 {
		zaplog.Zap.Error(fmt.Sprintf("email not found:%v", ctx.GetHeaders()))
		return exceptions.ErrBadRequest
	}
	contentTypeHeader, exists := ctx.GetHeaders()["Content-Type"]
	if !exists || len(contentTypeHeader) == 0 {
		zaplog.Zap.Error(fmt.Sprintf("content-type not found:%v", ctx.GetHeaders()))
		return exceptions.ErrBadRequest
	}
	mpo, err := ctx.App().State().MustGet(fileServer.STATENAME).(fileServer.RustFSService).UploadSingleFile(ctx, fmt.Sprintf("avatar/%s", emailHeader[0]), bytes.NewReader(ctx.BodyRaw()), contentTypeHeader[0])
	if err != nil {
		zaplog.Zap.Error(fmt.Sprintf("upload file failed: %v", err))
		return err
	}
	if err := ctx.App().State().MustGet(database.STATENAME).(database.Service).UploadAvatar(ctx, emailHeader[0], mpo.Location); err != nil {
		return err
	}

	return ctx.Status(http.StatusOK).JSON(response.Success("上传成功", nil))
}

func GetUserDetails(ctx fiber.Ctx) error {
	searchinfo := ctx.Query("searchinfo")
	if searchinfo == "" {
		return exceptions.ErrBadRequest
	}
	userInfo, err := ctx.App().State().MustGet(database.STATENAME).(database.Service).GetUserDetails(ctx, searchinfo)
	if err != nil {
		return err
	}
	return ctx.Status(http.StatusOK).JSON(response.Success("获取用户信息成功", userInfo))
}

func Search(ctx fiber.Ctx) error {
	searchinfo := ctx.Query("searchinfo")
	if searchinfo == "" {
		return exceptions.ErrBadRequest
	}
	userInfo, err := ctx.App().State().MustGet(database.STATENAME).(database.Service).Search(ctx, searchinfo)
	if err != nil {
		return err
	}
	return ctx.Status(http.StatusOK).JSON(response.Success("搜索成功", userInfo))
}

func GetUserList(ctx fiber.Ctx) error{
	userList, err := ctx.App().State().MustGet(database.STATENAME).(database.Service).GetUserList(ctx)
	if err != nil {
		return err
	}
	return ctx.Status(http.StatusOK).JSON(response.Success("获取用户列表成功", userList))
}

func AddFriend(ctx fiber.Ctx) error {
	addFriend := new(request.AddFriendRequest)
	if err := ctx.Bind().Body(addFriend); err != nil {
		zaplog.Zap.Error(fmt.Sprintf("bind body failed: %v", err))
		return exceptions.ErrBadRequest
	}
	if err := ctx.App().State().MustGet(utils.ValidatorSTATENAME).(*utils.StructValidator).Validate(addFriend); err != nil {
		return exceptions.ErrInvalidParameters
	}
	if err := ctx.App().State().MustGet(database.STATENAME).(database.Service).AddFriend(addFriend, ctx); err != nil {
		return err
	}
	return ctx.Status(http.StatusOK).JSON(response.Success("发送好友请求成功", nil))
}