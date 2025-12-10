package server

import (
	"GH-Server/internal/database"
	"GH-Server/internal/middleware"
	"os"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	_ "github.com/joho/godotenv/autoload"
)

type FiberServer struct {
	*fiber.App
	database.Service
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
		Service: database.New(),
	}
}
