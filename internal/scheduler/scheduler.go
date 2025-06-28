package scheduler

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/time/rate"

	"payer-status-io/internal/config"
)

// Scheduler manages the scheduling of probe tasks
type Scheduler struct {
	heap      *TaskHeap
	limiters  map[string]*rate.Limiter // Per-endpoint rate limiters
	taskChan  chan *Task
	logger    *zap.Logger
	mu        sync.RWMutex
	jitterPct float64 // Jitter percentage (0.1 = 10%)
}

// New creates a new scheduler
func New(logger *zap.Logger, taskChanSize int) *Scheduler {
	return &Scheduler{
		heap:      NewTaskHeap(),
		limiters:  make(map[string]*rate.Limiter),
		taskChan:  make(chan *Task, taskChanSize),
		logger:    logger,
		jitterPct: 0.1, // 10% jitter as per .windsurfrules
	}
}

// LoadConfig loads endpoints from configuration and creates tasks
func (s *Scheduler) LoadConfig(cfg *config.Config) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Clear existing tasks and limiters
	s.heap = NewTaskHeap()
	s.limiters = make(map[string]*rate.Limiter)

	now := time.Now()
	
	for _, payer := range cfg.Payers {
		for _, endpoint := range payer.Endpoints {
			// Create task
			task := &Task{
				Payer:    payer.Name,
				Endpoint: endpoint,
				NextRun:  now.Add(s.addJitter(endpoint.GetSchedule())),
				Interval: endpoint.GetSchedule(),
			}
			
			s.heap.PushTask(task)

			// Create rate limiter for this endpoint
			limiterKey := s.getLimiterKey(payer.Name, endpoint.Type)
			interval := endpoint.GetSchedule()
			
			// Ensure minimum interval of 1 minute as per .windsurfrules
			if interval < time.Minute {
				interval = time.Minute
			}
			
			s.limiters[limiterKey] = rate.NewLimiter(rate.Every(interval), 1)
		}
	}

	s.logger.Info("Scheduler loaded configuration",
		zap.Int("total_tasks", s.heap.Len()),
		zap.Int("rate_limiters", len(s.limiters)))
}

// Start begins the scheduling loop
func (s *Scheduler) Start(ctx context.Context) error {
	s.logger.Info("Starting scheduler")
	
	ticker := time.NewTicker(100 * time.Millisecond) // Check every 100ms
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Scheduler stopping due to context cancellation")
			close(s.taskChan)
			return ctx.Err()
			
		case <-ticker.C:
			s.processNextTasks(ctx)
		}
	}
}

// GetTaskChannel returns the channel for receiving scheduled tasks
func (s *Scheduler) GetTaskChannel() <-chan *Task {
	return s.taskChan
}

// processNextTasks checks for and processes ready tasks
func (s *Scheduler) processNextTasks(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	
	for {
		task := s.heap.PeekTask()
		if task == nil || task.NextRun.After(now) {
			break // No more ready tasks
		}

		// Remove task from heap
		task = s.heap.PopTask()
		
		// Check rate limiter
		limiterKey := s.getLimiterKey(task.Payer, task.Endpoint.Type)
		limiter, exists := s.limiters[limiterKey]
		
		if exists && limiter.Allow() {
			// Send task to workers (non-blocking)
			select {
			case s.taskChan <- task:
				s.logger.Debug("Task scheduled for execution",
					zap.String("payer", task.Payer),
					zap.String("type", task.Endpoint.Type))
			default:
				s.logger.Warn("Task channel full, dropping task",
					zap.String("payer", task.Payer),
					zap.String("type", task.Endpoint.Type))
			}
		} else if !exists {
			s.logger.Error("No rate limiter found for task",
				zap.String("payer", task.Payer),
				zap.String("type", task.Endpoint.Type))
		}

		// Reschedule task for next run
		task.NextRun = now.Add(s.addJitter(task.Interval))
		s.heap.PushTask(task)
	}
}

// addJitter adds random jitter to prevent thundering herd
func (s *Scheduler) addJitter(duration time.Duration) time.Duration {
	if s.jitterPct <= 0 {
		return duration
	}
	
	// Add ±jitterPct random variation
	jitter := float64(duration) * s.jitterPct * (2*rand.Float64() - 1)
	return duration + time.Duration(jitter)
}

// getLimiterKey creates a unique key for rate limiters
func (s *Scheduler) getLimiterKey(payer, endpointType string) string {
	return payer + ":" + endpointType
}

// GetStats returns scheduler statistics
func (s *Scheduler) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return map[string]interface{}{
		"total_tasks":     s.heap.Len(),
		"rate_limiters":   len(s.limiters),
		"task_chan_size":  len(s.taskChan),
		"task_chan_cap":   cap(s.taskChan),
	}
}
