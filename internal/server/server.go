package server

import (
	"GH-Server/internal/database"
	"GH-Server/internal/handler"
	"GH-Server/internal/middleware"
	"GH-Server/pkg/utils"
	"os"

	"github.com/bytedance/sonic"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"

	_ "github.com/joho/godotenv/autoload"
)

type FiberServer struct {
	*fiber.App
	*handler.Handler
}

func New() *FiberServer {
	return &FiberServer{
		App: fiber.New(fiber.Config{
			AppName:      os.Getenv("APP_NAME"),
			ServerHeader: os.Getenv("APP_NAME"),
			JSONEncoder:  sonic.Marshal,
			JSONDecoder:  sonic.Unmarshal,
			ErrorHandler: middleware.ErrorHandler,
		}),
		Handler: &handler.Handler{
			Service: database.New(),
			StructValidator: &utils.StructValidator{
				Validator: validator.New(),
			},
		},
	}
}
