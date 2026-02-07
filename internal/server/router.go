package server

import (
	"GH-Server/internal/database"
	"GH-Server/internal/handler"
	"GH-Server/internal/middleware"
	fibersatoken "GH-Server/internal/middleware/fiber-sa-token"
	"GH-Server/pkg/zaplog"

	"github.com/gofiber/contrib/v3/monitor"
	middlewareZap "github.com/gofiber/contrib/v3/zap"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"go.uber.org/zap/zapcore"
)



func (server *FiberServer) RegisterRoutes() {

	// CORS
	server.App.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false, // credentials require explicit origins
		MaxAge:           300,
	}))

	// Logger
	server.App.Use(middlewareZap.New(middlewareZap.Config{
		Logger: zaplog.Zap,
		Levels: []zapcore.Level{zapcore.ErrorLevel, zapcore.WarnLevel, zapcore.InfoLevel},
	}))

	// Recovery
	server.App.Use(recover.New(recover.ConfigDefault))
	
	//JWT
	// server.App.Use(middleware.NewJwtMiddleWare())

	// SaToken-Go+FiberV3--> SaTokenMiddleware
	middleware.SaTokenMiddleware(server.State().MustGet(database.STATENAME).(database.Service).GetRedisClient())
	saPlugin:= fibersatoken.NewPlugin(fibersatoken.GetManager())

	// Database  health
	server.App.Get("/health", saPlugin.AuthMiddleware(),server.healthHandler)

	// Metrics
	server.App.Get("/metrics", monitor.New(monitor.Config{
		Title: "GH-Server Monitor",
	}))

	userRoute := server.App.Group("/user")
	userRoute.Post("/register", handler.UserRegister)
	userRoute.Post("/sendVerifyCode", handler.SendVerifyMail)
	userRoute.Post("/login",handler.UserLogin)
	userRoute.Get("/logout",saPlugin.AuthMiddleware(), handler.UserLogout)
	userRoute.Post("/uploadAvatar",handler.UserUploadAvatar)
}

func (server *FiberServer) healthHandler(ctx fiber.Ctx) error {
	return ctx.JSON(server.State().MustGet(database.STATENAME).(database.Service).Health())
}
