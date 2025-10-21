package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/atlanssia/fustgo/internal/logger"
	"github.com/atlanssia/fustgo/internal/models"
)

// JobExecutor defines the interface for executing jobs
type JobExecutor interface {
	Execute(ctx context.Context, jobID string) error
}

// Scheduler manages scheduled job executions
type Scheduler struct {
	mu         sync.RWMutex
	cron       *cron.Cron
	jobs       map[string]cron.EntryID // jobID -> cron entry ID
	executor   JobExecutor
	running    bool
	ctx        context.Context
	cancel     context.CancelFunc
}

// Config holds scheduler configuration
type Config struct {
	Location *time.Location
	Logger   cron.Logger
}

// DefaultConfig returns default scheduler configuration
func DefaultConfig() *Config {
	return &Config{
		Location: time.UTC,
		Logger:   nil, // Use default logger
	}
}

// NewScheduler creates a new job scheduler
func NewScheduler(executor JobExecutor, config *Config) *Scheduler {
	if config == nil {
		config = DefaultConfig()
	}

	opts := []cron.Option{
		cron.WithLocation(config.Location),
	}

	if config.Logger != nil {
		opts = append(opts, cron.WithLogger(config.Logger))
	}

	return &Scheduler{
		cron:     cron.New(opts...),
		jobs:     make(map[string]cron.EntryID),
		executor: executor,
	}
}

// Start starts the scheduler
func (s *Scheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("scheduler is already running")
	}

	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.cron.Start()
	s.running = true

	logger.Info("Scheduler started")
	return nil
}

// Stop stops the scheduler
func (s *Scheduler) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return fmt.Errorf("scheduler is not running")
	}

	ctx := s.cron.Stop()
	<-ctx.Done() // Wait for all running jobs to complete

	if s.cancel != nil {
		s.cancel()
	}

	s.running = false
	logger.Info("Scheduler stopped")
	return nil
}

// AddJob adds a job to the scheduler
func (s *Scheduler) AddJob(jobID string, cronExpr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if job is already scheduled
	if _, exists := s.jobs[jobID]; exists {
		return fmt.Errorf("job %s is already scheduled", jobID)
	}

	// Parse and validate cron expression
	if cronExpr == "" {
		return fmt.Errorf("cron expression is required")
	}

	// Add job to cron
	entryID, err := s.cron.AddFunc(cronExpr, func() {
		s.executeJob(jobID)
	})
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	s.jobs[jobID] = entryID
	logger.Info("Scheduled job %s with cron expression: %s", jobID, cronExpr)
	return nil
}

// RemoveJob removes a job from the scheduler
func (s *Scheduler) RemoveJob(jobID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entryID, exists := s.jobs[jobID]
	if !exists {
		return fmt.Errorf("job %s is not scheduled", jobID)
	}

	s.cron.Remove(entryID)
	delete(s.jobs, jobID)

	logger.Info("Removed job %s from scheduler", jobID)
	return nil
}

// UpdateJob updates a job's schedule
func (s *Scheduler) UpdateJob(jobID string, cronExpr string) error {
	// Remove existing schedule
	if err := s.RemoveJob(jobID); err != nil {
		// If job doesn't exist, just add it
		if err.Error() != fmt.Sprintf("job %s is not scheduled", jobID) {
			return err
		}
	}

	// Add with new schedule
	return s.AddJob(jobID, cronExpr)
}

// GetScheduledJobs returns all scheduled job IDs
func (s *Scheduler) GetScheduledJobs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jobs := make([]string, 0, len(s.jobs))
	for jobID := range s.jobs {
		jobs = append(jobs, jobID)
	}
	return jobs
}

// GetNextRun returns the next scheduled run time for a job
func (s *Scheduler) GetNextRun(jobID string) (*time.Time, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entryID, exists := s.jobs[jobID]
	if !exists {
		return nil, fmt.Errorf("job %s is not scheduled", jobID)
	}

	entry := s.cron.Entry(entryID)
	if entry.ID == 0 {
		return nil, fmt.Errorf("job entry not found")
	}

	nextRun := entry.Next
	return &nextRun, nil
}

// IsRunning returns whether the scheduler is running
func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// executeJob executes a scheduled job
func (s *Scheduler) executeJob(jobID string) {
	logger.Info("Executing scheduled job: %s", jobID)

	ctx := s.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	// Create a timeout context for job execution
	execCtx, cancel := context.WithTimeout(ctx, 1*time.Hour)
	defer cancel()

	if err := s.executor.Execute(execCtx, jobID); err != nil {
		logger.Error("Failed to execute scheduled job %s: %v", jobID, err)
	} else {
		logger.Info("Successfully executed scheduled job: %s", jobID)
	}
}

// GetJobCount returns the number of scheduled jobs
func (s *Scheduler) GetJobCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.jobs)
}

// SchedulingConfig represents scheduling configuration for a job
type SchedulingConfig struct {
	Enabled      bool   `json:"enabled" yaml:"enabled"`
	CronExpr     string `json:"cron_expr" yaml:"cron_expr"`
	Timezone     string `json:"timezone,omitempty" yaml:"timezone,omitempty"`
	MaxRetries   int    `json:"max_retries,omitempty" yaml:"max_retries,omitempty"`
	RetryDelay   string `json:"retry_delay,omitempty" yaml:"retry_delay,omitempty"`
	Timeout      string `json:"timeout,omitempty" yaml:"timeout,omitempty"`
}

// ParseSchedulingConfig parses a scheduling configuration from a job
func ParseSchedulingConfig(job *models.Job) (*SchedulingConfig, error) {
	if job.SchedulingConfig == "" {
		return nil, fmt.Errorf("no scheduling configuration found")
	}

	// For simplicity, assume the SchedulingConfig is a cron expression
	// In production, this would parse JSON/YAML
	return &SchedulingConfig{
		Enabled:  true,
		CronExpr: job.SchedulingConfig,
	}, nil
}

// ValidateCronExpression validates a cron expression
func ValidateCronExpression(expr string) error {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	_, err := parser.Parse(expr)
	return err
}

// CronExpressionExamples provides common cron expression examples
var CronExpressionExamples = map[string]string{
	"Every minute":             "* * * * *",
	"Every 5 minutes":          "*/5 * * * *",
	"Every 15 minutes":         "*/15 * * * *",
	"Every 30 minutes":         "*/30 * * * *",
	"Every hour":               "0 * * * *",
	"Every 2 hours":            "0 */2 * * *",
	"Every day at midnight":    "0 0 * * *",
	"Every day at noon":        "0 12 * * *",
	"Every Monday at 9 AM":     "0 9 * * 1",
	"Every weekday at 6 PM":    "0 18 * * 1-5",
	"First day of month":       "0 0 1 * *",
	"Last day of month":        "0 0 L * *",
	"Every Sunday at midnight": "0 0 * * 0",
}
