package main

import (
	"GH-Server/internal/server"
	"GH-Server/pkg/zaplog"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	_ "github.com/joho/godotenv/autoload"
)

var (
	port, _ = strconv.Atoi(os.Getenv("PORT"))
)

func main() {
	servers := server.New()
	servers.RegisterRoutes()
	done := make(chan bool, 1)
	go func() {
		if err := servers.Listen(fmt.Sprintf(":%d", port), fiber.ListenConfig{
			EnablePrefork: true,
		}); err != nil {
			zaplog.Zap.Panic(fmt.Sprintf("Http Error: %v", err))
		}
	}()
	go gracefulShutdown(servers, done)
	<-done
	zaplog.Zap.Info("Graceful shutdown complete.")
}

func gracefulShutdown(fiberServer *server.FiberServer, done chan bool) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
	zaplog.Zap.Info("shutting down gracefully, press Ctrl+C again to force")
	stop()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := fiberServer.ShutdownWithContext(ctx); err != nil {
		zaplog.Zap.Error(fmt.Sprintf("Server forced to shutdown with error: %v", err))
	}
	zaplog.Zap.Info("Server exiting")
	done <- true
}
