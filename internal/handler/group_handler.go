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

func GetGroups(ctx fiber.Ctx) error {
	groupList, err := ctx.App().State().MustGet(database.STATENAME).(database.Service).GetGroups(ctx)
	if err != nil {
		return err
	}
	return ctx.Status(http.StatusOK).JSON(response.Success("获取群组成功", groupList)) 
}

func CreateGroup(ctx fiber.Ctx) error {
	request := new(request.GroupCreateRequest)
	if err := ctx.Bind().Body(request); err != nil {
		zaplog.Zap.Error(fmt.Sprintf("bind body failed: %v", err))
		return exceptions.ErrBadRequest
	}
	if err := ctx.App().State().MustGet(utils.ValidatorSTATENAME).(*utils.StructValidator).Validate(request); err != nil {
		return exceptions.ErrInvalidParameters
	}
	if err := ctx.App().State().MustGet(database.STATENAME).(database.Service).CreateGroup(request, ctx); err != nil {
		return err
	}
	return ctx.Status(http.StatusOK).JSON(response.Success("创建群组成功", nil))
}