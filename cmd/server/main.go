package main

import (
	"context"
	"errors"
	"log"
	"os/signal"
	"syscall"

	"go-yt-dlp/internal/download"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// main inicia el servidor HTTP y el manager de descargas.
func main() {
	cfg := download.LoadConfig()

	manager, err := download.NewManager(cfg)
	if err != nil {
		log.Fatalf("no se pudo iniciar manager: %v", err)
	}

	app := fiber.New(fiber.Config{
		BodyLimit: cfg.MaxBodyBytes,
	})
	app.Use(logger.New())
	app.Use(recover.New())

	download.RegisterRoutes(app, manager, websocket.Config{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	})

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	manager.Start(ctx)

	go func() {
		<-ctx.Done()
		if err := app.Shutdown(); err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("error al cerrar app: %v", err)
		}
	}()

	log.Printf("servidor escuchando en %s", cfg.ListenAddr)
	if err := app.Listen(cfg.ListenAddr); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("error en servidor: %v", err)
	}
}
