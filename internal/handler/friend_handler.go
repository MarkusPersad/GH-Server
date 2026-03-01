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

func AgreeFriend(ctx fiber.Ctx) error { 
	addFriend := new(request.AddFriendRequest)
	if err := ctx.Bind().Body(addFriend); err != nil {
		zaplog.Zap.Error(fmt.Sprintf("bind body failed: %v", err))
		return exceptions.ErrBadRequest
	}
	if err := ctx.App().State().MustGet(utils.ValidatorSTATENAME).(*utils.StructValidator).Validate(addFriend); err != nil {
		return exceptions.ErrInvalidParameters
	}
	if err := ctx.App().State().MustGet(database.STATENAME).(database.Service).AgreeFriend(addFriend, ctx); err != nil {
		return err
	}
	return ctx.Status(http.StatusOK).JSON(response.Success("发送好友请求成功", nil))
}