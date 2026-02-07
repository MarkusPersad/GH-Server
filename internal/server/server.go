package server

import (
	"GH-Server/internal/database"
	"GH-Server/internal/fileServer"
	"GH-Server/internal/middleware"
	"GH-Server/pkg/utils"
	"os"

	"github.com/bytedance/sonic"
	"github.com/gofiber/fiber/v3"

	_ "github.com/joho/godotenv/autoload"
)

type FiberServer struct {
	*fiber.App
}

func New() *FiberServer {
	app := &FiberServer{
		App: fiber.New(fiber.Config{
			AppName:      os.Getenv("APP_NAME"),
			ServerHeader: os.Getenv("APP_NAME"),
			JSONEncoder:  sonic.Marshal,
			JSONDecoder:  sonic.Unmarshal,
			ErrorHandler: middleware.ErrorHandler,
		}),
	}
	app.State().Set(database.STATENAME,database.New())
	app.State().Set(fileServer.STATENAME,fileServer.New())
	app.State().Set(utils.ValidatorSTATENAME,utils.NewValidator())
	return app
}
