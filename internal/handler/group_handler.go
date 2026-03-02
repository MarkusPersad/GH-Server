package handler

import (
	"GH-Server/internal/database"
	"GH-Server/pkg/response"
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