package handler

import "github.com/gofiber/fiber/v3"

// Health responds with a simple liveness payload.
func Health(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok"})
}
