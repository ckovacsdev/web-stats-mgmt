package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"

	"github.com/ckovacs/web-stats-mgmt/internal/pagespeed"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app := newApp(pagespeed.New())
	err := app.Listen(":"+port, fiber.ListenConfig{
		DisableStartupMessage: true,
		GracefulContext:       ctx,
		ShutdownTimeout:       10 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}
}

func newApp(psi *pagespeed.Client) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      "web-stats-mgmt",
		ReadTimeout:  10 * time.Second,
		ErrorHandler: jsonErrorHandler,
	})
	app.Use(recover.New())

	app.Post("/scores", scoresHandler(psi))

	return app
}

func jsonErrorHandler(c fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	msg := "internal error"
	var fe *fiber.Error
	if errors.As(err, &fe) {
		code, msg = fe.Code, fe.Message
	}
	return c.Status(code).JSON(fiber.Map{"error": msg})
}
