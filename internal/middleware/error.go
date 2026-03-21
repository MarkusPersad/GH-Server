package middleware

import (
	"GH-Server/pkg/exceptions"
	"GH-Server/pkg/response"
	"errors"
	"strconv"

	"net/http"

	"github.com/gofiber/fiber/v3"
)

const (
	XHR = "XHR"
	ACCESS_STATUS = "Access-Status"
)


func ErrorHandler(ctx fiber.Ctx, err error) error {
	var exception *exceptions.Exception
	if errors.As(err, &exception) {
		if exception.Code == exceptions.ErrTokenExpired.Code {
			if ok,_:=strconv.ParseBool(ctx.Get(XHR));ok {
				return ctx.SendStatus(exception.Code)
			}
			ctx.Set(ACCESS_STATUS,"true")
			return ctx.SendStatus(fiber.StatusOK)	
		}
		return ctx.Status(http.StatusOK).JSON(response.Error(exception, nil))
	}
	return ctx.Status(http.StatusInternalServerError).JSON(response.Error(exceptions.ErrInternalServerError,nil))
}
