package middleware

import (
	"GH-Server/pkg/exceptions"
	"GH-Server/pkg/response"
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v3"
)



func ErrorHandler(ctx fiber.Ctx, err error) error {
	var exception *exceptions.Exception
	if errors.As(err, &exception) {
		return ctx.Status(http.StatusOK).JSON(response.Error(exception, nil))
	}
	return ctx.Status(http.StatusInternalServerError).JSON(response.Error(exceptions.ErrInternalServerError,nil))
}
