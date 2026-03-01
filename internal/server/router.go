package server

import (
	"GH-Server/internal/database"
	"GH-Server/internal/handler"
	"GH-Server/internal/middleware"
	"GH-Server/pkg/utils"
	"GH-Server/pkg/zaplog"
	"os"
	"strings"

	jwtware "github.com/gofiber/contrib/v3/jwt"
	"github.com/gofiber/contrib/v3/monitor"
	middlewareZap "github.com/gofiber/contrib/v3/zap"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/extractors"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"go.uber.org/zap/zapcore"
)


var (
	jwtSecret = os.Getenv("JWT_SECRET")
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
	server.App.Use(jwtware.New(jwtware.Config{
		SigningKey: jwtware.SigningKey{Key: []byte(jwtSecret)},
		Extractor: extractors.FromAuthHeader("Bearer"),
		Claims: &utils.AccessClaims{},
		ErrorHandler: middleware.JwtErrorHandler,
		SuccessHandler: middleware.JwtSuccessHandler,
		Next: func(ctx fiber.Ctx) bool {
			return strings.Contains(ctx.Path(),"/register") ||
				strings.Contains(ctx.Path(),"/login") ||
				strings.Contains(ctx.Path(),"/sendVerifyCode") ||
				strings.Contains(ctx.Path(),"/uploadAvatar") ||
				strings.Contains(ctx.Path(),"/refresh")
		},
	}))


	// Database  health
	server.App.Get("/health",middleware.CheckRole("admin"),server.healthHandler)

	// Metrics
	server.App.Get("/metrics",middleware.CheckRole("admin"), monitor.New(monitor.Config{
		Title: "GH-Server Monitor",
	}))

	server.Get("/refresh",middleware.RefreshTokenHandler)
	server.Get("/tick/:UserID",middleware.CheckRole("admin"),middleware.Tick)

	userRoute := server.App.Group("/user")
	userRoute.Post("/register", handler.UserRegister)
	userRoute.Post("/sendVerifyCode", handler.SendVerifyMail)
	userRoute.Post("/login",handler.UserLogin)
	userRoute.Get("/logout", handler.UserLogout)
	userRoute.Post("/uploadAvatar",handler.UserUploadAvatar)
	userRoute.Get("/details", handler.GetUserDetails)
	userRoute.Get("/search", handler.Search)
}

func (server *FiberServer) healthHandler(ctx fiber.Ctx) error {
	return ctx.JSON(server.State().MustGet(database.STATENAME).(database.Service).Health())
}
