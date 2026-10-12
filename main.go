package main

import (
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/ckovacs/web-stats-mgmt/internal/pagespeed"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	app := newApp(pagespeed.New())

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		<-sig
		_ = app.ShutdownWithTimeout(10 * time.Second)
	}()

	if err := app.Listen(":" + port); err != nil {
		log.Fatal(err)
	}
}

func newApp(psi *pagespeed.Client) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:               "web-stats-mgmt",
		DisableStartupMessage: true,
		ReadTimeout:           10 * time.Second,
		ErrorHandler:          jsonErrorHandler,
	})
	app.Use(recover.New())

	app.Post("/scores", scoresHandler(psi))

	return app
}

func jsonErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	msg := "internal error"
	var fe *fiber.Error
	if errors.As(err, &fe) {
		code, msg = fe.Code, fe.Message
	}
	return c.Status(code).JSON(fiber.Map{"error": msg})
}
