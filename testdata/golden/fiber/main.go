package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"testproj/internal/handler"

	"github.com/gofiber/fiber/v3"
)

func main() {
	app := fiber.New()

	app.Get("/health", handler.Health)

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	// Listen for SIGINT/SIGTERM so the server can drain in-flight requests.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := app.Listen(":" + port); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down server...")

	if err := app.Shutdown(); err != nil {
		log.Fatalf("graceful shutdown failed: %v", err)
	}
	log.Println("server stopped")
}
