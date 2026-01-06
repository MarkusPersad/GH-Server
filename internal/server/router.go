package server

import (
	"GH-Server/internal/middleware"
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
	server.App.Use(middleware.NewJwtMiddleWare())

	// Database  health
	server.App.Get("/health", server.healthHandler)

	// Metrics
	server.App.Get("/metrics", monitor.New(monitor.Config{
		Title: "GH-Server Monitor",
	}))

	userRoute := server.App.Group("/user")
	userRoute.Post("/register", server.Handler.UserRegister)
	userRoute.Post("/sendVerifyCode", server.Handler.SendVerifyMail)
	userRoute.Post("/login",server.UserLogin)
	userRoute.Get("/logout",server.Handler.UserLogout)
}

func (server *FiberServer) healthHandler(ctx fiber.Ctx) error {
	return ctx.JSON(server.Health())
}
