package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

// CORS - middleware для настройки Cross-Origin Resource Sharing.
// Разрешает запросы со всех источников для корректной работы тайлов и API на фронтенде.
func CORS() fiber.Handler {
	return cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Content-Type,Accept,Accept-Language,Authorization,If-None-Match",
	})
}
