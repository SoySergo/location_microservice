package worker

import (
	"sync"

	"go.uber.org/zap"
)

// BaseWorker содержит общую логику для всех воркеров
type BaseWorker struct {
	name          string
	logger        *zap.Logger
	stopChan      chan struct{}
	stopped       bool
	mu            sync.Mutex
	consumerGroup string
}

// NewBaseWorker создает новый BaseWorker
func NewBaseWorker(name, consumerGroup string, logger *zap.Logger) *BaseWorker {
	return &BaseWorker{
		name:          name,
		logger:        logger,
		stopChan:      make(chan struct{}),
		consumerGroup: consumerGroup,
	}
}

// Name возвращает имя воркера
func (w *BaseWorker) Name() string {
	return w.name
}

// Stop останавливает воркер
func (w *BaseWorker) Stop() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.stopped {
		return nil
	}

	w.logger.Info("Stopping worker", zap.String("name", w.name))
	close(w.stopChan)
	w.stopped = true

	return nil
}

// IsStopped проверяет, остановлен ли воркер
func (w *BaseWorker) IsStopped() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.stopped
}

// StopChan возвращает канал остановки
func (w *BaseWorker) StopChan() <-chan struct{} {
	return w.stopChan
}

// ConsumerGroup возвращает имя consumer group
func (w *BaseWorker) ConsumerGroup() string {
	return w.consumerGroup
}

// Logger возвращает логгер
func (w *BaseWorker) Logger() *zap.Logger {
	return w.logger
}
