package validator

import (
	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
}

// Validate - валидация структуры
func Validate(s interface{}) error {
	return validate.Struct(s)
}

// GetValidator - получить валидатор для кастомной конфигурации
func GetValidator() *validator.Validate {
	return validate
}
