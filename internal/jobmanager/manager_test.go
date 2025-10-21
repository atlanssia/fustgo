package jobmanager

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/atlanssia/fustgo/internal/database"
	"github.com/atlanssia/fustgo/internal/models"
)

func setupTestManager(t *testing.T) *Manager {
	tmpDB := t.TempDir() + "/test.db"
	store, err := database.NewSQLiteStore(tmpDB)
	require.NoError(t, err)

	return NewManager(store)
}

func createTestJob() *models.Job {
	return &models.Job{
		JobName: "test-job",
		JobType: models.JobTypeETL,
		Description: "Test job description",
		ConfigYAML: `
input:
  type: csv
  path: /data/input.csv
output:
  type: csv
  path: /data/output.csv
`,
		Enabled:  true,
		Priority: 1,
	}
}

func TestNewManager(t *testing.T) {
	manager := setupTestManager(t)
	assert.NotNil(t, manager)
	assert.NotNil(t, manager.store)
	assert.NotNil(t, manager.jobs)
	assert.NotNil(t, manager.running)
}

func TestCreateJob(t *testing.T) {
	manager := setupTestManager(t)
	job := createTestJob()

	err := manager.CreateJob(job)
	require.NoError(t, err)

	assert.NotEmpty(t, job.JobID)
	assert.Equal(t, models.JobStatusDraft, job.Status)
	assert.False(t, job.CreatedAt.IsZero())
	assert.False(t, job.UpdatedAt.IsZero())
}

func TestCreateJobWithInvalidConfig(t *testing.T) {
	manager := setupTestManager(t)
	job := createTestJob()
	job.ConfigYAML = "invalid yaml: [[[" // Invalid YAML

	err := manager.CreateJob(job)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid YAML configuration")
}

func TestCreateJobMissingInput(t *testing.T) {
	manager := setupTestManager(t)
	job := createTestJob()
	job.ConfigYAML = `
output:
  type: csv
  path: /data/output.csv
`

	err := manager.CreateJob(job)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "input")
}

func TestCreateJobMissingOutput(t *testing.T) {
	manager := setupTestManager(t)
	job := createTestJob()
	job.ConfigYAML = `
input:
  type: csv
  path: /data/input.csv
`

	err := manager.CreateJob(job)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "output")
}

func TestGetJob(t *testing.T) {
	manager := setupTestManager(t)
	job := createTestJob()

	err := manager.CreateJob(job)
	require.NoError(t, err)

	retrieved, err := manager.GetJob(job.JobID)
	require.NoError(t, err)
	assert.Equal(t, job.JobID, retrieved.JobID)
	assert.Equal(t, job.JobName, retrieved.JobName)
}

func TestGetJobNotFound(t *testing.T) {
	manager := setupTestManager(t)

	_, err := manager.GetJob("non-existent-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestListJobs(t *testing.T) {
	manager := setupTestManager(t)

	// Create multiple jobs
	for i := 0; i < 3; i++ {
		job := createTestJob()
		job.JobName = "test-job-" + string(rune('a'+i))
		err := manager.CreateJob(job)
		require.NoError(t, err)
	}

	jobs, err := manager.ListJobs(nil)
	require.NoError(t, err)
	assert.Equal(t, 3, len(jobs))
}

func TestUpdateJob(t *testing.T) {
	manager := setupTestManager(t)
	job := createTestJob()

	err := manager.CreateJob(job)
	require.NoError(t, err)

	// Update job description
	job.Description = "Updated description"
	err = manager.UpdateJob(job)
	require.NoError(t, err)

	// Verify update
	retrieved, err := manager.GetJob(job.JobID)
	require.NoError(t, err)
	assert.Equal(t, "Updated description", retrieved.Description)
}

func TestDeleteJob(t *testing.T) {
	manager := setupTestManager(t)
	job := createTestJob()

	err := manager.CreateJob(job)
	require.NoError(t, err)

	err = manager.DeleteJob(job.JobID)
	require.NoError(t, err)

	_, err = manager.GetJob(job.JobID)
	assert.Error(t, err)
}

func TestDeleteRunningJob(t *testing.T) {
	manager := setupTestManager(t)
	job := createTestJob()
	job.Status = models.JobStatusReady

	err := manager.CreateJob(job)
	require.NoError(t, err)

	// Start job
	err = manager.StartJob(job.JobID)
	require.NoError(t, err)

	// Try to delete running job
	err = manager.DeleteJob(job.JobID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot delete running job")
}

func TestStartJob(t *testing.T) {
	manager := setupTestManager(t)
	job := createTestJob()
	job.Status = models.JobStatusReady

	err := manager.CreateJob(job)
	require.NoError(t, err)

	err = manager.StartJob(job.JobID)
	require.NoError(t, err)

	// Verify job is running
	retrieved, err := manager.GetJob(job.JobID)
	require.NoError(t, err)
	assert.Equal(t, models.JobStatusRunning, retrieved.Status)

	// Verify context exists
	ctx, err := manager.GetJobContext(job.JobID)
	require.NoError(t, err)
	assert.NotNil(t, ctx)
}

func TestStartJobAlreadyRunning(t *testing.T) {
	manager := setupTestManager(t)
	job := createTestJob()
	job.Status = models.JobStatusReady

	err := manager.CreateJob(job)
	require.NoError(t, err)

	err = manager.StartJob(job.JobID)
	require.NoError(t, err)

	// Try to start again
	err = manager.StartJob(job.JobID)
	assert.Error(t, err)
	// The error could be either "already running" or "cannot start job from status running"
	assert.True(t, 
		assert.ObjectsAreEqual(err.Error(), "job is already running") ||
		assert.ObjectsAreEqual(err.Error(), "cannot start job from status running"),
		"Expected error about already running job, got: %v", err,
	)
}

func TestStopJob(t *testing.T) {
	manager := setupTestManager(t)
	job := createTestJob()
	job.Status = models.JobStatusReady

	err := manager.CreateJob(job)
	require.NoError(t, err)

	err = manager.StartJob(job.JobID)
	require.NoError(t, err)

	err = manager.StopJob(job.JobID)
	require.NoError(t, err)

	// Verify job is stopped
	retrieved, err := manager.GetJob(job.JobID)
	require.NoError(t, err)
	assert.Equal(t, models.JobStatusCompleted, retrieved.Status)
}

func TestPauseJob(t *testing.T) {
	manager := setupTestManager(t)
	job := createTestJob()
	job.Status = models.JobStatusReady

	err := manager.CreateJob(job)
	require.NoError(t, err)

	err = manager.StartJob(job.JobID)
	require.NoError(t, err)

	err = manager.PauseJob(job.JobID)
	require.NoError(t, err)

	// Verify job is paused
	retrieved, err := manager.GetJob(job.JobID)
	require.NoError(t, err)
	assert.Equal(t, models.JobStatusPaused, retrieved.Status)
}

func TestResumeJob(t *testing.T) {
	manager := setupTestManager(t)
	job := createTestJob()
	job.Status = models.JobStatusReady

	err := manager.CreateJob(job)
	require.NoError(t, err)

	err = manager.StartJob(job.JobID)
	require.NoError(t, err)

	err = manager.PauseJob(job.JobID)
	require.NoError(t, err)

	err = manager.ResumeJob(job.JobID)
	require.NoError(t, err)

	// Verify job is running
	retrieved, err := manager.GetJob(job.JobID)
	require.NoError(t, err)
	assert.Equal(t, models.JobStatusRunning, retrieved.Status)
}

func TestValidateStateTransition(t *testing.T) {
	manager := setupTestManager(t)

	tests := []struct {
		name      string
		from      models.JobStatus
		to        models.JobStatus
		expectErr bool
	}{
		{"Draft to Ready", models.JobStatusDraft, models.JobStatusReady, false},
		{"Ready to Running", models.JobStatusReady, models.JobStatusRunning, false},
		{"Running to Paused", models.JobStatusRunning, models.JobStatusPaused, false},
		{"Paused to Running", models.JobStatusPaused, models.JobStatusRunning, false},
		{"Running to Completed", models.JobStatusRunning, models.JobStatusCompleted, false},
		{"Completed to Ready", models.JobStatusCompleted, models.JobStatusReady, false},
		{"Failed to Ready", models.JobStatusFailed, models.JobStatusReady, false},
		{"Draft to Running", models.JobStatusDraft, models.JobStatusRunning, true}, // Invalid
		{"Completed to Running", models.JobStatusCompleted, models.JobStatusRunning, true}, // Invalid
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.validateStateTransition(tt.from, tt.to)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetJobStats(t *testing.T) {
	manager := setupTestManager(t)

	// Create jobs with different statuses
	statuses := []models.JobStatus{
		models.JobStatusDraft,
		models.JobStatusReady,
		models.JobStatusRunning,
		models.JobStatusPaused,
		models.JobStatusCompleted,
		models.JobStatusFailed,
	}

	for _, status := range statuses {
		job := createTestJob()
		job.Status = status
		err := manager.CreateJob(job)
		require.NoError(t, err)

		// Start running jobs to update status properly
		if status == models.JobStatusRunning {
			job.Status = models.JobStatusReady
			manager.store.UpdateJob(job)
			manager.StartJob(job.JobID)
		}
	}

	stats := manager.GetJobStats()
	assert.Equal(t, 6, stats["total"])
	assert.Equal(t, 1, stats["draft"])
	assert.Equal(t, 1, stats["ready"])
	assert.Equal(t, 1, stats["running"])
	assert.Equal(t, 1, stats["paused"])
	assert.Equal(t, 1, stats["completed"])
	assert.Equal(t, 1, stats["failed"])
}

func TestGetJobContext(t *testing.T) {
	manager := setupTestManager(t)
	job := createTestJob()
	job.Status = models.JobStatusReady

	err := manager.CreateJob(job)
	require.NoError(t, err)

	// Before starting, context should not exist
	_, err = manager.GetJobContext(job.JobID)
	assert.Error(t, err)

	// Start job
	err = manager.StartJob(job.JobID)
	require.NoError(t, err)

	// Now context should exist
	ctx, err := manager.GetJobContext(job.JobID)
	require.NoError(t, err)
	assert.NotNil(t, ctx)

	// Context should be active
	select {
	case <-ctx.Done():
		t.Fatal("Context should not be done")
	default:
		// Good
	}

	// Stop job
	err = manager.StopJob(job.JobID)
	require.NoError(t, err)

	// Context should be cancelled
	time.Sleep(10 * time.Millisecond) // Give it time to cancel
	select {
	case <-ctx.Done():
		// Good
	default:
		t.Fatal("Context should be done after stopping job")
	}
}
