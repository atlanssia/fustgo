package jobmanager

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	"github.com/atlanssia/fustgo/internal/database"
	"github.com/atlanssia/fustgo/internal/logger"
	"github.com/atlanssia/fustgo/internal/models"
)

// Manager handles job lifecycle management with state machine
type Manager struct {
	mu      sync.RWMutex
	store   database.MetadataStore
	jobs    map[string]*JobInstance // jobID -> instance
	running map[string]context.CancelFunc // jobID -> cancel function
}

// JobInstance represents a running job instance
type JobInstance struct {
	Job       *models.Job
	Status    models.JobStatus
	UpdatedAt time.Time
	Ctx       context.Context
	Cancel    context.CancelFunc
}

// NewManager creates a new job manager
func NewManager(store database.MetadataStore) *Manager {
	return &Manager{
		store:   store,
		jobs:    make(map[string]*JobInstance),
		running: make(map[string]context.CancelFunc),
	}
}

// CreateJob creates a new job
func (m *Manager) CreateJob(job *models.Job) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Generate job ID if not provided
	if job.JobID == "" {
		job.JobID = uuid.New().String()
	}

	// Validate state transition
	if job.Status == "" {
		job.Status = models.JobStatusDraft
	}

	// Set timestamps
	now := time.Now()
	job.CreatedAt = now
	job.UpdatedAt = now

	// Validate configuration
	if err := m.validateJobConfig(job); err != nil {
		return fmt.Errorf("invalid job configuration: %w", err)
	}

	// Save to database
	if err := m.store.SaveJob(job); err != nil {
		return fmt.Errorf("failed to save job: %w", err)
	}

	// Add to in-memory cache
	m.jobs[job.JobID] = &JobInstance{
		Job:       job,
		Status:    job.Status,
		UpdatedAt: now,
	}

	logger.Info("Created job %s (%s)", job.JobID, job.JobName)
	return nil
}

// GetJob retrieves a job by ID
func (m *Manager) GetJob(jobID string) (*models.Job, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Try in-memory cache first
	if instance, exists := m.jobs[jobID]; exists {
		return instance.Job, nil
	}

	// Fallback to database
	job, err := m.store.GetJob(jobID)
	if err != nil {
		return nil, fmt.Errorf("job not found: %w", err)
	}

	// Update cache
	m.jobs[jobID] = &JobInstance{
		Job:       job,
		Status:    job.Status,
		UpdatedAt: job.UpdatedAt,
	}

	return job, nil
}

// ListJobs lists all jobs with optional filters
func (m *Manager) ListJobs(filter map[string]interface{}) ([]*models.Job, error) {
	jobs, err := m.store.ListJobs(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}

	// Update cache
	m.mu.Lock()
	for _, job := range jobs {
		if _, exists := m.jobs[job.JobID]; !exists {
			m.jobs[job.JobID] = &JobInstance{
				Job:       job,
				Status:    job.Status,
				UpdatedAt: job.UpdatedAt,
			}
		}
	}
	m.mu.Unlock()

	return jobs, nil
}

// UpdateJob updates an existing job
func (m *Manager) UpdateJob(job *models.Job) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if job exists
	existing, err := m.store.GetJob(job.JobID)
	if err != nil {
		return fmt.Errorf("job not found: %w", err)
	}

	// Validate state transition (allow same-state transitions for updates)
	if job.Status != existing.Status {
		if err := m.validateStateTransition(existing.Status, job.Status); err != nil {
			return err
		}
	}

	// Update timestamp
	job.UpdatedAt = time.Now()

	// Validate configuration if changed
	if job.ConfigYAML != existing.ConfigYAML {
		if err := m.validateJobConfig(job); err != nil {
			return fmt.Errorf("invalid job configuration: %w", err)
		}
	}

	// Update in database
	if err := m.store.UpdateJob(job); err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	// Update cache
	if instance, exists := m.jobs[job.JobID]; exists {
		instance.Job = job
		instance.Status = job.Status
		instance.UpdatedAt = job.UpdatedAt
	}

	logger.Info("Updated job %s (%s) to status %s", job.JobID, job.JobName, job.Status)
	return nil
}

// DeleteJob deletes a job
func (m *Manager) DeleteJob(jobID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get job to check status
	job, err := m.store.GetJob(jobID)
	if err != nil {
		return fmt.Errorf("job not found: %w", err)
	}

	// Cannot delete running jobs
	if job.Status == models.JobStatusRunning {
		return fmt.Errorf("cannot delete running job, stop it first")
	}

	// Delete from database
	if err := m.store.DeleteJob(jobID); err != nil {
		return fmt.Errorf("failed to delete job: %w", err)
	}

	// Remove from cache
	delete(m.jobs, jobID)

	logger.Info("Deleted job %s (%s)", jobID, job.JobName)
	return nil
}

// StartJob starts a job execution
func (m *Manager) StartJob(jobID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, err := m.store.GetJob(jobID)
	if err != nil {
		return fmt.Errorf("job not found: %w", err)
	}

	// Validate state transition
	if job.Status == models.JobStatusRunning {
		return fmt.Errorf("cannot start job from status %s", job.Status)
	}
	if err := m.validateStateTransition(job.Status, models.JobStatusRunning); err != nil {
		return err
	}

	// Check if already running
	if _, running := m.running[jobID]; running {
		return fmt.Errorf("job is already running")
	}

	// Update job status
	job.Status = models.JobStatusRunning
	job.UpdatedAt = time.Now()
	if err := m.store.UpdateJob(job); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	// Create context for job execution
	ctx, cancel := context.WithCancel(context.Background())
	m.running[jobID] = cancel

	// Update cache
	if instance, exists := m.jobs[jobID]; exists {
		instance.Job = job
		instance.Status = job.Status
		instance.UpdatedAt = job.UpdatedAt
		instance.Ctx = ctx
		instance.Cancel = cancel
	} else {
		m.jobs[jobID] = &JobInstance{
			Job:       job,
			Status:    job.Status,
			UpdatedAt: job.UpdatedAt,
			Ctx:       ctx,
			Cancel:    cancel,
		}
	}

	logger.Info("Started job %s (%s)", jobID, job.JobName)
	return nil
}

// StopJob stops a running job
func (m *Manager) StopJob(jobID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, err := m.store.GetJob(jobID)
	if err != nil {
		return fmt.Errorf("job not found: %w", err)
	}

	// Validate state transition
	if job.Status != models.JobStatusRunning && job.Status != models.JobStatusPaused {
		return fmt.Errorf("cannot stop job in status %s", job.Status)
	}

	// Cancel job execution
	if cancel, exists := m.running[jobID]; exists {
		cancel()
		delete(m.running, jobID)
	}

	// Update job status
	job.Status = models.JobStatusCompleted
	job.UpdatedAt = time.Now()
	if err := m.store.UpdateJob(job); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	// Update cache
	if instance, exists := m.jobs[jobID]; exists {
		instance.Job = job
		instance.Status = job.Status
		instance.UpdatedAt = job.UpdatedAt
		instance.Ctx = nil
		instance.Cancel = nil
	}

	logger.Info("Stopped job %s (%s)", jobID, job.JobName)
	return nil
}

// PauseJob pauses a running job
func (m *Manager) PauseJob(jobID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, err := m.store.GetJob(jobID)
	if err != nil {
		return fmt.Errorf("job not found: %w", err)
	}

	// Validate state transition
	if err := m.validateStateTransition(job.Status, models.JobStatusPaused); err != nil {
		return err
	}

	// Update job status
	job.Status = models.JobStatusPaused
	job.UpdatedAt = time.Now()
	if err := m.store.UpdateJob(job); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	// Update cache
	if instance, exists := m.jobs[jobID]; exists {
		instance.Job = job
		instance.Status = job.Status
		instance.UpdatedAt = job.UpdatedAt
	}

	logger.Info("Paused job %s (%s)", jobID, job.JobName)
	return nil
}

// ResumeJob resumes a paused job
func (m *Manager) ResumeJob(jobID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, err := m.store.GetJob(jobID)
	if err != nil {
		return fmt.Errorf("job not found: %w", err)
	}

	// Validate state transition
	if err := m.validateStateTransition(job.Status, models.JobStatusRunning); err != nil {
		return err
	}

	// Update job status
	job.Status = models.JobStatusRunning
	job.UpdatedAt = time.Now()
	if err := m.store.UpdateJob(job); err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	// Update cache
	if instance, exists := m.jobs[jobID]; exists {
		instance.Job = job
		instance.Status = job.Status
		instance.UpdatedAt = job.UpdatedAt
	}

	logger.Info("Resumed job %s (%s)", jobID, job.JobName)
	return nil
}

// GetJobContext returns the context for a running job
func (m *Manager) GetJobContext(jobID string) (context.Context, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	instance, exists := m.jobs[jobID]
	if !exists {
		return nil, fmt.Errorf("job not found")
	}

	if instance.Ctx == nil {
		return nil, fmt.Errorf("job is not running")
	}

	return instance.Ctx, nil
}

// validateStateTransition validates job state transitions
func (m *Manager) validateStateTransition(from, to models.JobStatus) error {
	validTransitions := map[models.JobStatus][]models.JobStatus{
		models.JobStatusDraft: {
			models.JobStatusReady,
		},
		models.JobStatusReady: {
			models.JobStatusRunning,
			models.JobStatusDraft,
		},
		models.JobStatusRunning: {
			models.JobStatusPaused,
			models.JobStatusCompleted,
			models.JobStatusFailed,
		},
		models.JobStatusPaused: {
			models.JobStatusRunning,
			models.JobStatusCompleted,
		},
		models.JobStatusCompleted: {
			models.JobStatusReady, // Can restart
		},
		models.JobStatusFailed: {
			models.JobStatusReady, // Can retry
		},
	}

	allowed, exists := validTransitions[from]
	if !exists {
		return fmt.Errorf("no valid transitions from status %s", from)
	}

	for _, status := range allowed {
		if status == to {
			return nil
		}
	}

	return fmt.Errorf("invalid state transition from %s to %s", from, to)
}

// validateJobConfig validates the job configuration
func (m *Manager) validateJobConfig(job *models.Job) error {
	if job.JobName == "" {
		return fmt.Errorf("job name is required")
	}

	if job.ConfigYAML == "" {
		return fmt.Errorf("configuration is required")
	}

	// Parse and validate YAML
	var config map[string]interface{}
	if err := yaml.Unmarshal([]byte(job.ConfigYAML), &config); err != nil {
		return fmt.Errorf("invalid YAML configuration: %w", err)
	}

	// Store parsed config
	job.Config = config

	// Validate required fields
	if _, hasInput := config["input"]; !hasInput {
		return fmt.Errorf("configuration must have 'input' section")
	}

	if _, hasOutput := config["output"]; !hasOutput {
		return fmt.Errorf("configuration must have 'output' section")
	}

	return nil
}

// GetJobStats returns statistics for all jobs
func (m *Manager) GetJobStats() map[string]int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := map[string]int{
		"total":     len(m.jobs),
		"draft":     0,
		"ready":     0,
		"running":   0,
		"paused":    0,
		"completed": 0,
		"failed":    0,
	}

	for _, instance := range m.jobs {
		switch instance.Status {
		case models.JobStatusDraft:
			stats["draft"]++
		case models.JobStatusReady:
			stats["ready"]++
		case models.JobStatusRunning:
			stats["running"]++
		case models.JobStatusPaused:
			stats["paused"]++
		case models.JobStatusCompleted:
			stats["completed"]++
		case models.JobStatusFailed:
			stats["failed"]++
		}
	}

	return stats
}
