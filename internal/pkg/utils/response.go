package utils

import (
	"github.com/gofiber/fiber/v2"
	"github.com/location-microservice/internal/pkg/errors"
)

type SuccessResponse struct {
	Data interface{} `json:"data"`
	Meta *Meta       `json:"meta,omitempty"`
}

type ErrorResponse struct {
	Error *errors.AppError `json:"error"`
}

type Meta struct {
	Total    int     `json:"total,omitempty"`
	Page     int     `json:"page,omitempty"`
	Limit    int     `json:"limit,omitempty"`
	TimeMSec float64 `json:"time_ms,omitempty"`
}

func SendSuccess(c *fiber.Ctx, data interface{}, meta *Meta) error {
	return c.JSON(SuccessResponse{
		Data: data,
		Meta: meta,
	})
}

func SendError(c *fiber.Ctx, err error) error {
	if appErr, ok := err.(*errors.AppError); ok {
		return c.Status(appErr.StatusCode).JSON(ErrorResponse{
			Error: appErr,
		})
	}

	// Unknown error - return 500
	return c.Status(500).JSON(ErrorResponse{
		Error: errors.ErrInternalServer,
	})
}
