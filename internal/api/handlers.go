package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/atlanssia/fustgo/internal/database"
	"github.com/atlanssia/fustgo/internal/jobmanager"
	"github.com/atlanssia/fustgo/internal/models"
	"github.com/atlanssia/fustgo/internal/plugin"
	"github.com/atlanssia/fustgo/internal/worker"
)

// Handler holds dependencies for API handlers
type Handler struct {
	jobManager *jobmanager.Manager
	workerPool *worker.Pool
	registry   *plugin.Registry
	store      database.MetadataStore
}

// NewHandler creates a new API handler
func NewHandler(
	jobManager *jobmanager.Manager,
	workerPool *worker.Pool,
	registry *plugin.Registry,
	store database.MetadataStore,
) *Handler {
	return &Handler{
		jobManager: jobManager,
		workerPool: workerPool,
		registry:   registry,
		store:      store,
	}
}

// Job Management Handlers

type CreateJobRequest struct {
	JobName          string `json:"job_name" binding:"required"`
	JobType          string `json:"job_type" binding:"required"`
	Description      string `json:"description"`
	ConfigYAML       string `json:"config_yaml" binding:"required"`
	SchedulingConfig string `json:"scheduling_config"`
	Priority         int    `json:"priority"`
}

func (h *Handler) CreateJob(c *gin.Context) {
	var req CreateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	job := &models.Job{
		JobName:          req.JobName,
		JobType:          models.JobType(req.JobType),
		Description:      req.Description,
		ConfigYAML:       req.ConfigYAML,
		SchedulingConfig: req.SchedulingConfig,
		Priority:         req.Priority,
		Enabled:          true,
	}

	if err := h.jobManager.CreateJob(job); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"job": job})
}

func (h *Handler) ListJobs(c *gin.Context) {
	// Get query parameters for filtering
	filter := make(map[string]interface{})
	
	if status := c.Query("status"); status != "" {
		filter["status"] = status
	}

	jobs, err := h.jobManager.ListJobs(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"jobs": jobs,
		"total": len(jobs),
	})
}

func (h *Handler) GetJob(c *gin.Context) {
	jobID := c.Param("id")

	job, err := h.jobManager.GetJob(jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"job": job})
}

type UpdateJobRequest struct {
	JobName          string `json:"job_name"`
	Description      string `json:"description"`
	ConfigYAML       string `json:"config_yaml"`
	SchedulingConfig string `json:"scheduling_config"`
	Priority         int    `json:"priority"`
	Enabled          *bool  `json:"enabled"`
}

func (h *Handler) UpdateJob(c *gin.Context) {
	jobID := c.Param("id")

	var req UpdateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get existing job
	job, err := h.jobManager.GetJob(jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	// Update fields
	if req.JobName != "" {
		job.JobName = req.JobName
	}
	if req.Description != "" {
		job.Description = req.Description
	}
	if req.ConfigYAML != "" {
		job.ConfigYAML = req.ConfigYAML
	}
	if req.SchedulingConfig != "" {
		job.SchedulingConfig = req.SchedulingConfig
	}
	if req.Priority != 0 {
		job.Priority = req.Priority
	}
	if req.Enabled != nil {
		job.Enabled = *req.Enabled
	}

	if err := h.jobManager.UpdateJob(job); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"job": job})
}

func (h *Handler) DeleteJob(c *gin.Context) {
	jobID := c.Param("id")

	if err := h.jobManager.DeleteJob(jobID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "job deleted successfully"})
}

func (h *Handler) StartJob(c *gin.Context) {
	jobID := c.Param("id")

	if err := h.jobManager.StartJob(jobID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "job started successfully"})
}

func (h *Handler) StopJob(c *gin.Context) {
	jobID := c.Param("id")

	if err := h.jobManager.StopJob(jobID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "job stopped successfully"})
}

func (h *Handler) PauseJob(c *gin.Context) {
	jobID := c.Param("id")

	if err := h.jobManager.PauseJob(jobID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "job paused successfully"})
}

func (h *Handler) ResumeJob(c *gin.Context) {
	jobID := c.Param("id")

	if err := h.jobManager.ResumeJob(jobID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "job resumed successfully"})
}

// Plugin Management Handlers

func (h *Handler) ListPlugins(c *gin.Context) {
	pluginType := c.Query("type")

	var plugins []gin.H

	if pluginType == "" || pluginType == "input" {
		inputs := h.registry.ListInputs()
		for _, name := range inputs {
			plugins = append(plugins, gin.H{
				"name": name,
				"type": "input",
			})
		}
	}

	if pluginType == "" || pluginType == "processor" {
		processors := h.registry.ListProcessors()
		for _, name := range processors {
			plugins = append(plugins, gin.H{
				"name": name,
				"type": "processor",
			})
		}
	}

	if pluginType == "" || pluginType == "output" {
		outputs := h.registry.ListOutputs()
		for _, name := range outputs {
			plugins = append(plugins, gin.H{
				"name": name,
				"type": "output",
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"plugins": plugins,
		"total":   len(plugins),
	})
}

func (h *Handler) GetPlugin(c *gin.Context) {
	name := c.Param("name")
	pluginType := c.Query("type")

	var plugin interface{}
	var err error

	switch pluginType {
	case "input":
		plugin, err = h.registry.GetInput(name)
	case "processor":
		plugin, err = h.registry.GetProcessor(name)
	case "output":
		plugin, err = h.registry.GetOutput(name)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "plugin type is required"})
		return
	}

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "plugin not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"plugin": plugin})
}

// Worker Management Handlers

func (h *Handler) ListWorkers(c *gin.Context) {
	workers, err := h.workerPool.ListWorkers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"workers": workers,
		"total":   len(workers),
	})
}

func (h *Handler) GetWorker(c *gin.Context) {
	workerID := c.Param("id")

	worker, err := h.workerPool.GetWorker(workerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "worker not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"worker": worker})
}

// Monitoring Handlers

func (h *Handler) GetStats(c *gin.Context) {
	jobStats := h.jobManager.GetJobStats()
	poolStats := h.workerPool.GetPoolStats()

	c.JSON(http.StatusOK, gin.H{
		"jobs":    jobStats,
		"workers": poolStats,
	})
}

func (h *Handler) GetMetrics(c *gin.Context) {
	// TODO: Implement Prometheus metrics
	c.JSON(http.StatusOK, gin.H{
		"metrics": map[string]interface{}{
			"placeholder": "metrics endpoint",
		},
	})
}
