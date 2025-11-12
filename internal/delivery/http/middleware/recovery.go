package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// Recovery - middleware для восстановления после паники
func Recovery() fiber.Handler {
	return recover.New(recover.Config{
		EnableStackTrace: true,
	})
}
