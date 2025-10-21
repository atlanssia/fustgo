package models

import "time"

// JobStatus represents the status of a job
type JobStatus string

const (
	JobStatusDraft     JobStatus = "draft"
	JobStatusReady     JobStatus = "ready"
	JobStatusRunning   JobStatus = "running"
	JobStatusPaused    JobStatus = "paused"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
)

// JobType represents the type of ETL job
type JobType string

const (
	JobTypeETL JobType = "etl"
	JobTypeELT JobType = "elt"
)

// Job represents a data synchronization job
type Job struct {
	JobID            string                 `json:"job_id" db:"job_id"`
	JobName          string                 `json:"job_name" db:"job_name"`
	JobType          JobType                `json:"job_type" db:"job_type"`
	Description      string                 `json:"description" db:"description"`
	ConfigYAML       string                 `json:"config_yaml" db:"config_yaml"`
	FlowDiagram      string                 `json:"flow_diagram,omitempty" db:"flow_diagram"`
	SchedulingConfig string                 `json:"scheduling_config,omitempty" db:"scheduling_config"`
	Status           JobStatus              `json:"status" db:"status"`
	CreatedBy        string                 `json:"created_by" db:"created_by"`
	CreatedAt        time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at" db:"updated_at"`
	Enabled          bool                   `json:"enabled" db:"enabled"`
	Priority         int                    `json:"priority" db:"priority"`
	RetryPolicy      string                 `json:"retry_policy,omitempty" db:"retry_policy"`
	Config           map[string]interface{} `json:"config,omitempty" db:"-"` // Parsed config
}

// ExecutionStatus represents the status of a job execution
type ExecutionStatus string

const (
	ExecutionStatusPending   ExecutionStatus = "pending"
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusCompleted ExecutionStatus = "completed"
	ExecutionStatusFailed    ExecutionStatus = "failed"
	ExecutionStatusCancelled ExecutionStatus = "cancelled"
)

// Execution represents a single execution of a job
type Execution struct {
	ExecutionID      string          `json:"execution_id" db:"execution_id"`
	JobID            string          `json:"job_id" db:"job_id"`
	Status           ExecutionStatus `json:"status" db:"status"`
	StartTime        time.Time       `json:"start_time" db:"start_time"`
	EndTime          *time.Time      `json:"end_time,omitempty" db:"end_time"`
	RecordsRead      int64           `json:"records_read" db:"records_read"`
	RecordsWritten   int64           `json:"records_written" db:"records_written"`
	RecordsFailed    int64           `json:"records_failed" db:"records_failed"`
	BytesTransferred int64           `json:"bytes_transferred" db:"bytes_transferred"`
	ErrorMessage     string          `json:"error_message,omitempty" db:"error_message"`
	WorkerID         string          `json:"worker_id" db:"worker_id"`
	CheckpointData   string          `json:"checkpoint_data,omitempty" db:"checkpoint_data"`
}

// Duration returns the execution duration
func (e *Execution) Duration() time.Duration {
	if e.EndTime == nil {
		return time.Since(e.StartTime)
	}
	return e.EndTime.Sub(e.StartTime)
}

// WorkerStatus represents the status of a worker node
type WorkerStatus string

const (
	WorkerStatusOnline  WorkerStatus = "online"
	WorkerStatusOffline WorkerStatus = "offline"
	WorkerStatusBusy    WorkerStatus = "busy"
)

// Worker represents a worker node
type Worker struct {
	WorkerID      string       `json:"worker_id" db:"worker_id"`
	Hostname      string       `json:"hostname" db:"hostname"`
	IPAddress     string       `json:"ip_address" db:"ip_address"`
	Port          int          `json:"port" db:"port"`
	Status        WorkerStatus `json:"status" db:"status"`
	CPUCores      int          `json:"cpu_cores" db:"cpu_cores"`
	MemoryMB      int          `json:"memory_mb" db:"memory_mb"`
	LastHeartbeat time.Time    `json:"last_heartbeat" db:"last_heartbeat"`
	RegisteredAt  time.Time    `json:"registered_at" db:"registered_at"`
}

// IsHealthy checks if the worker is healthy based on last heartbeat
func (w *Worker) IsHealthy(timeout time.Duration) bool {
	return time.Since(w.LastHeartbeat) < timeout
}

// Plugin represents a registered plugin
type Plugin struct {
	PluginID       string    `json:"plugin_id" db:"plugin_id"`
	PluginName     string    `json:"plugin_name" db:"plugin_name"`
	PluginType     string    `json:"plugin_type" db:"plugin_type"`
	Version        string    `json:"version" db:"version"`
	DataSourceType string    `json:"data_source_type,omitempty" db:"data_source_type"`
	ConfigSchema   string    `json:"config_schema" db:"config_schema"`
	Metadata       string    `json:"metadata,omitempty" db:"metadata"`
	Enabled        bool      `json:"enabled" db:"enabled"`
	InstalledAt    time.Time `json:"installed_at" db:"installed_at"`
}

// AlertRule represents an alert configuration
type AlertRule struct {
	AlertID   string    `json:"alert_id" db:"alert_id"`
	AlertName string    `json:"alert_name" db:"alert_name"`
	AlertType string    `json:"alert_type" db:"alert_type"`
	Condition string    `json:"condition" db:"condition"`
	Action    string    `json:"action" db:"action"`
	Enabled   bool      `json:"enabled" db:"enabled"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}
