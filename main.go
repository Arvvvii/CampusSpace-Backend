package main

import (
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"

	"github.com/Arvvvii/CampusSpace-Backend/app/repository"
	"github.com/Arvvvii/CampusSpace-Backend/app/service"
	"github.com/Arvvvii/CampusSpace-Backend/config"
	"github.com/Arvvvii/CampusSpace-Backend/database"
	"github.com/Arvvvii/CampusSpace-Backend/route"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	db, err := database.NewPostgresDB(cfg.Database)
	if err != nil {
		return fmt.Errorf("failed to connect to postgresql: %w", err)
	}
	defer db.Close()

	roomRepo := repository.NewRoomRepository(db)
	roomService := service.NewRoomService(roomRepo)

	app := fiber.New(fiber.Config{
		AppName: cfg.App.Name,
	})

	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Welcome to CampusSpace API",
			"status":  "success",
		})
	})

	route.RegisterRoomRoutes(app, roomService)

	addr := fmt.Sprintf(":%d", cfg.App.Port)
	if err := app.Listen(addr); err != nil {
		return fmt.Errorf("failed to start server on %s: %w", addr, err)
	}

	return nil
}
