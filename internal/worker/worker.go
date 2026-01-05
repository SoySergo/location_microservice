package worker

import (
	"context"
)

// Worker интерфейс для всех воркеров
type Worker interface {
	// Start запускает воркер
	Start(ctx context.Context) error

	// Stop останавливает воркер
	Stop() error

	// Name возвращает имя воркера
	Name() string
}
