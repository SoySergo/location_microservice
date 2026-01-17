package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	// shutdownTimeout - максимальное время ожидания завершения воркеров
	shutdownTimeout = 30 * time.Second
)

// WorkerManager управляет несколькими воркерами
type WorkerManager struct {
	workers []Worker
	logger  *zap.Logger
	wg      sync.WaitGroup
	mu      sync.Mutex
}

// NewWorkerManager создает новый WorkerManager
func NewWorkerManager(logger *zap.Logger) *WorkerManager {
	return &WorkerManager{
		workers: make([]Worker, 0),
		logger:  logger,
	}
}

// Register регистрирует воркер
func (m *WorkerManager) Register(w Worker) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.workers = append(m.workers, w)
	m.logger.Info("Worker registered", zap.String("name", w.Name()))
}

// Start запускает все зарегистрированные воркеры
func (m *WorkerManager) Start(ctx context.Context) error {
	m.mu.Lock()
	workers := make([]Worker, len(m.workers))
	copy(workers, m.workers)
	m.mu.Unlock()

	if len(workers) == 0 {
		return fmt.Errorf("no workers registered")
	}

	m.logger.Info("Starting workers", zap.Int("count", len(workers)))

	// Запускаем каждый воркер в отдельной горутине
	for _, worker := range workers {
		m.wg.Add(1)
		go func(w Worker) {
			defer m.wg.Done()

			m.logger.Info("Starting worker", zap.String("name", w.Name()))
			if err := w.Start(ctx); err != nil {
				m.logger.Error("Worker failed",
					zap.String("name", w.Name()),
					zap.Error(err))
			}
		}(worker)
	}

	return nil
}

// Stop останавливает все воркеры с timeout
func (m *WorkerManager) Stop() error {
	m.mu.Lock()
	workers := make([]Worker, len(m.workers))
	copy(workers, m.workers)
	m.mu.Unlock()

	m.logger.Info("Stopping workers", zap.Int("count", len(workers)))

	// Останавливаем все воркеры (сигнализируем о завершении)
	for _, worker := range workers {
		if err := worker.Stop(); err != nil {
			m.logger.Error("Failed to stop worker",
				zap.String("name", worker.Name()),
				zap.Error(err))
		}
	}

	// Ждём завершения с timeout
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		m.logger.Info("All workers stopped gracefully")
	case <-time.After(shutdownTimeout):
		m.logger.Warn("Workers shutdown timed out, some tasks may not have completed",
			zap.Duration("timeout", shutdownTimeout))
		return fmt.Errorf("workers shutdown timed out after %v", shutdownTimeout)
	}

	return nil
}
