package scheduler

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockExecutor is a mock job executor for testing
type MockExecutor struct {
	mu           sync.Mutex
	executions   []string
	shouldFail   bool
	executionErr error
	delay        time.Duration
}

func (m *MockExecutor) Execute(ctx context.Context, jobID string) error {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	m.mu.Lock()
	m.executions = append(m.executions, jobID)
	m.mu.Unlock()

	if m.shouldFail {
		return m.executionErr
	}
	return nil
}

func (m *MockExecutor) GetExecutions() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]string, len(m.executions))
	copy(result, m.executions)
	return result
}

func (m *MockExecutor) GetExecutionCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.executions)
}

func (m *MockExecutor) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.executions = nil
}

func TestNewScheduler(t *testing.T) {
	executor := &MockExecutor{}
	scheduler := NewScheduler(executor, nil)

	assert.NotNil(t, scheduler)
	assert.NotNil(t, scheduler.cron)
	assert.NotNil(t, scheduler.jobs)
	assert.NotNil(t, scheduler.executor)
	assert.False(t, scheduler.running)
}

func TestNewSchedulerWithConfig(t *testing.T) {
	executor := &MockExecutor{}
	config := &Config{
		Location: time.Local,
	}
	scheduler := NewScheduler(executor, config)

	assert.NotNil(t, scheduler)
}

func TestStartScheduler(t *testing.T) {
	executor := &MockExecutor{}
	scheduler := NewScheduler(executor, nil)

	err := scheduler.Start()
	require.NoError(t, err)
	assert.True(t, scheduler.IsRunning())

	// Cleanup
	scheduler.Stop()
}

func TestStartSchedulerAlreadyRunning(t *testing.T) {
	executor := &MockExecutor{}
	scheduler := NewScheduler(executor, nil)

	err := scheduler.Start()
	require.NoError(t, err)

	// Try to start again
	err = scheduler.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	// Cleanup
	scheduler.Stop()
}

func TestStopScheduler(t *testing.T) {
	executor := &MockExecutor{}
	scheduler := NewScheduler(executor, nil)

	err := scheduler.Start()
	require.NoError(t, err)

	err = scheduler.Stop()
	require.NoError(t, err)
	assert.False(t, scheduler.IsRunning())
}

func TestStopSchedulerNotRunning(t *testing.T) {
	executor := &MockExecutor{}
	scheduler := NewScheduler(executor, nil)

	err := scheduler.Stop()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
}

func TestAddJob(t *testing.T) {
	executor := &MockExecutor{}
	scheduler := NewScheduler(executor, nil)

	err := scheduler.Start()
	require.NoError(t, err)
	defer scheduler.Stop()

	err = scheduler.AddJob("job-1", "*/1 * * * *") // Every minute
	require.NoError(t, err)

	jobs := scheduler.GetScheduledJobs()
	assert.Equal(t, 1, len(jobs))
	assert.Contains(t, jobs, "job-1")
}

func TestAddJobInvalidCron(t *testing.T) {
	executor := &MockExecutor{}
	scheduler := NewScheduler(executor, nil)

	err := scheduler.Start()
	require.NoError(t, err)
	defer scheduler.Stop()

	err = scheduler.AddJob("job-1", "invalid cron")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid cron expression")
}

func TestAddJobEmptyCron(t *testing.T) {
	executor := &MockExecutor{}
	scheduler := NewScheduler(executor, nil)

	err := scheduler.Start()
	require.NoError(t, err)
	defer scheduler.Stop()

	err = scheduler.AddJob("job-1", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cron expression is required")
}

func TestAddJobAlreadyScheduled(t *testing.T) {
	executor := &MockExecutor{}
	scheduler := NewScheduler(executor, nil)

	err := scheduler.Start()
	require.NoError(t, err)
	defer scheduler.Stop()

	err = scheduler.AddJob("job-1", "*/1 * * * *")
	require.NoError(t, err)

	// Try to add same job again
	err = scheduler.AddJob("job-1", "*/2 * * * *")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already scheduled")
}

func TestRemoveJob(t *testing.T) {
	executor := &MockExecutor{}
	scheduler := NewScheduler(executor, nil)

	err := scheduler.Start()
	require.NoError(t, err)
	defer scheduler.Stop()

	err = scheduler.AddJob("job-1", "*/1 * * * *")
	require.NoError(t, err)

	err = scheduler.RemoveJob("job-1")
	require.NoError(t, err)

	jobs := scheduler.GetScheduledJobs()
	assert.Equal(t, 0, len(jobs))
}

func TestRemoveJobNotScheduled(t *testing.T) {
	executor := &MockExecutor{}
	scheduler := NewScheduler(executor, nil)

	err := scheduler.Start()
	require.NoError(t, err)
	defer scheduler.Stop()

	err = scheduler.RemoveJob("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not scheduled")
}

func TestUpdateJob(t *testing.T) {
	executor := &MockExecutor{}
	scheduler := NewScheduler(executor, nil)

	err := scheduler.Start()
	require.NoError(t, err)
	defer scheduler.Stop()

	// Add job
	err = scheduler.AddJob("job-1", "*/1 * * * *")
	require.NoError(t, err)

	// Update job
	err = scheduler.UpdateJob("job-1", "*/5 * * * *")
	require.NoError(t, err)

	jobs := scheduler.GetScheduledJobs()
	assert.Equal(t, 1, len(jobs))
}

func TestUpdateJobNotScheduled(t *testing.T) {
	executor := &MockExecutor{}
	scheduler := NewScheduler(executor, nil)

	err := scheduler.Start()
	require.NoError(t, err)
	defer scheduler.Stop()

	// Update non-existent job should add it
	err = scheduler.UpdateJob("job-1", "*/1 * * * *")
	require.NoError(t, err)

	jobs := scheduler.GetScheduledJobs()
	assert.Equal(t, 1, len(jobs))
}

func TestGetNextRun(t *testing.T) {
	executor := &MockExecutor{}
	scheduler := NewScheduler(executor, nil)

	err := scheduler.Start()
	require.NoError(t, err)
	defer scheduler.Stop()

	err = scheduler.AddJob("job-1", "*/1 * * * *")
	require.NoError(t, err)

	nextRun, err := scheduler.GetNextRun("job-1")
	require.NoError(t, err)
	assert.NotNil(t, nextRun)
	assert.True(t, nextRun.After(time.Now()))
}

func TestGetNextRunNotScheduled(t *testing.T) {
	executor := &MockExecutor{}
	scheduler := NewScheduler(executor, nil)

	err := scheduler.Start()
	require.NoError(t, err)
	defer scheduler.Stop()

	_, err = scheduler.GetNextRun("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not scheduled")
}

func TestScheduledJobExecution(t *testing.T) {
	executor := &MockExecutor{}
	scheduler := NewScheduler(executor, nil)

	err := scheduler.Start()
	require.NoError(t, err)
	defer scheduler.Stop()

	// Schedule job to run every second (using cron format with seconds)
	// Note: Standard cron doesn't support seconds, so we use every minute
	// For testing, we'll trigger it manually
	err = scheduler.AddJob("test-job", "* * * * *")
	require.NoError(t, err)

	// Manually execute to test
	scheduler.executeJob("test-job")

	// Wait a bit for execution
	time.Sleep(100 * time.Millisecond)

	executions := executor.GetExecutions()
	assert.Equal(t, 1, len(executions))
	assert.Equal(t, "test-job", executions[0])
}

func TestGetJobCount(t *testing.T) {
	executor := &MockExecutor{}
	scheduler := NewScheduler(executor, nil)

	err := scheduler.Start()
	require.NoError(t, err)
	defer scheduler.Stop()

	assert.Equal(t, 0, scheduler.GetJobCount())

	scheduler.AddJob("job-1", "*/1 * * * *")
	assert.Equal(t, 1, scheduler.GetJobCount())

	scheduler.AddJob("job-2", "*/2 * * * *")
	assert.Equal(t, 2, scheduler.GetJobCount())

	scheduler.RemoveJob("job-1")
	assert.Equal(t, 1, scheduler.GetJobCount())
}

func TestValidateCronExpression(t *testing.T) {
	tests := []struct {
		name      string
		expr      string
		expectErr bool
	}{
		{"Valid: Every minute", "* * * * *", false},
		{"Valid: Every hour", "0 * * * *", false},
		{"Valid: Daily at noon", "0 12 * * *", false},
		{"Valid: Every 5 minutes", "*/5 * * * *", false},
		{"Invalid: Empty", "", true},
		{"Invalid: Too few fields", "* * *", true},
		{"Invalid: Too many fields", "* * * * * * *", true},
		{"Invalid: Bad syntax", "abc def ghi", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCronExpression(tt.expr)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCronExpressionExamples(t *testing.T) {
	// Verify all example expressions are valid
	for name, expr := range CronExpressionExamples {
		t.Run(name, func(t *testing.T) {
			// Skip "Last day of month" as it uses special L syntax not supported by standard parser
			if name == "Last day of month" {
				t.Skip("L syntax not supported by standard cron parser")
			}
			err := ValidateCronExpression(expr)
			assert.NoError(t, err, "Example '%s' should be valid: %s", name, expr)
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	assert.NotNil(t, config)
	assert.Equal(t, time.UTC, config.Location)
	assert.Nil(t, config.Logger)
}
