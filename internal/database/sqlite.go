package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/atlanssia/fustgo/internal/models"
)

// MetadataStore defines the interface for metadata storage
type MetadataStore interface {
	// Initialize initializes the database connection and schema
	Initialize(config map[string]interface{}) error

	// Close closes the database connection
	Close() error

	// Job operations
	SaveJob(job *models.Job) error
	GetJob(jobID string) (*models.Job, error)
	ListJobs(filter map[string]interface{}) ([]*models.Job, error)
	UpdateJob(job *models.Job) error
	DeleteJob(jobID string) error

	// Execution operations
	SaveExecution(exec *models.Execution) error
	GetExecution(executionID string) (*models.Execution, error)
	GetExecutions(jobID string, limit int) ([]*models.Execution, error)
	UpdateExecution(exec *models.Execution) error

	// Worker operations
	RegisterWorker(worker *models.Worker) error
	UpdateWorkerHeartbeat(workerID string) error
	GetWorker(workerID string) (*models.Worker, error)
	ListWorkers() ([]*models.Worker, error)
	UnregisterWorker(workerID string) error

	// Plugin operations
	RegisterPlugin(plugin *models.Plugin) error
	GetPlugin(pluginName string) (*models.Plugin, error)
	ListPlugins(pluginType string) ([]*models.Plugin, error)
	UpdatePluginStatus(pluginName string, enabled bool) error
}

// SQLiteStore implements MetadataStore using SQLite
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore creates a new SQLite metadata store
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode for better concurrency
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	store := &SQLiteStore{db: db}
	if err := store.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return store, nil
}

// Initialize implements MetadataStore.Initialize
func (s *SQLiteStore) Initialize(config map[string]interface{}) error {
	return s.initSchema()
}

// initSchema creates the database schema
func (s *SQLiteStore) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS jobs (
		job_id TEXT PRIMARY KEY,
		job_name TEXT NOT NULL,
		job_type TEXT NOT NULL,
		description TEXT,
		config_yaml TEXT NOT NULL,
		flow_diagram TEXT,
		scheduling_config TEXT,
		status TEXT NOT NULL,
		created_by TEXT,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL,
		enabled BOOLEAN NOT NULL DEFAULT 1,
		priority INTEGER DEFAULT 0,
		retry_policy TEXT
	);

	CREATE TABLE IF NOT EXISTS executions (
		execution_id TEXT PRIMARY KEY,
		job_id TEXT NOT NULL,
		status TEXT NOT NULL,
		start_time TIMESTAMP NOT NULL,
		end_time TIMESTAMP,
		records_read INTEGER DEFAULT 0,
		records_written INTEGER DEFAULT 0,
		records_failed INTEGER DEFAULT 0,
		bytes_transferred INTEGER DEFAULT 0,
		error_message TEXT,
		worker_id TEXT,
		checkpoint_data TEXT,
		FOREIGN KEY (job_id) REFERENCES jobs(job_id)
	);

	CREATE TABLE IF NOT EXISTS workers (
		worker_id TEXT PRIMARY KEY,
		hostname TEXT NOT NULL,
		ip_address TEXT NOT NULL,
		port INTEGER NOT NULL,
		status TEXT NOT NULL,
		cpu_cores INTEGER,
		memory_mb INTEGER,
		last_heartbeat TIMESTAMP NOT NULL,
		registered_at TIMESTAMP NOT NULL
	);

	CREATE TABLE IF NOT EXISTS plugins (
		plugin_id TEXT PRIMARY KEY,
		plugin_name TEXT UNIQUE NOT NULL,
		plugin_type TEXT NOT NULL,
		version TEXT NOT NULL,
		data_source_type TEXT,
		config_schema TEXT NOT NULL,
		metadata TEXT,
		enabled BOOLEAN NOT NULL DEFAULT 1,
		installed_at TIMESTAMP NOT NULL
	);

	CREATE TABLE IF NOT EXISTS alert_rules (
		alert_id TEXT PRIMARY KEY,
		alert_name TEXT NOT NULL,
		alert_type TEXT NOT NULL,
		condition TEXT NOT NULL,
		action TEXT NOT NULL,
		enabled BOOLEAN NOT NULL DEFAULT 1,
		created_at TIMESTAMP NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);
	CREATE INDEX IF NOT EXISTS idx_executions_job_id ON executions(job_id);
	CREATE INDEX IF NOT EXISTS idx_executions_status ON executions(status);
	CREATE INDEX IF NOT EXISTS idx_workers_status ON workers(status);
	`

	_, err := s.db.Exec(schema)
	return err
}

// Close implements MetadataStore.Close
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// SaveJob implements MetadataStore.SaveJob
func (s *SQLiteStore) SaveJob(job *models.Job) error {
	query := `
		INSERT INTO jobs (job_id, job_name, job_type, description, config_yaml, 
			flow_diagram, scheduling_config, status, created_by, created_at, 
			updated_at, enabled, priority, retry_policy)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.Exec(query,
		job.JobID, job.JobName, job.JobType, job.Description, job.ConfigYAML,
		job.FlowDiagram, job.SchedulingConfig, job.Status, job.CreatedBy,
		job.CreatedAt, job.UpdatedAt, job.Enabled, job.Priority, job.RetryPolicy,
	)
	return err
}

// GetJob implements MetadataStore.GetJob
func (s *SQLiteStore) GetJob(jobID string) (*models.Job, error) {
	query := `
		SELECT job_id, job_name, job_type, description, config_yaml, 
			flow_diagram, scheduling_config, status, created_by, created_at, 
			updated_at, enabled, priority, retry_policy
		FROM jobs WHERE job_id = ?
	`
	job := &models.Job{}
	err := s.db.QueryRow(query, jobID).Scan(
		&job.JobID, &job.JobName, &job.JobType, &job.Description, &job.ConfigYAML,
		&job.FlowDiagram, &job.SchedulingConfig, &job.Status, &job.CreatedBy,
		&job.CreatedAt, &job.UpdatedAt, &job.Enabled, &job.Priority, &job.RetryPolicy,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}
	return job, err
}

// ListJobs implements MetadataStore.ListJobs
func (s *SQLiteStore) ListJobs(filter map[string]interface{}) ([]*models.Job, error) {
	query := `
		SELECT job_id, job_name, job_type, description, config_yaml, 
			flow_diagram, scheduling_config, status, created_by, created_at, 
			updated_at, enabled, priority, retry_policy
		FROM jobs ORDER BY created_at DESC
	`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*models.Job
	for rows.Next() {
		job := &models.Job{}
		err := rows.Scan(
			&job.JobID, &job.JobName, &job.JobType, &job.Description, &job.ConfigYAML,
			&job.FlowDiagram, &job.SchedulingConfig, &job.Status, &job.CreatedBy,
			&job.CreatedAt, &job.UpdatedAt, &job.Enabled, &job.Priority, &job.RetryPolicy,
		)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

// UpdateJob implements MetadataStore.UpdateJob
func (s *SQLiteStore) UpdateJob(job *models.Job) error {
	query := `
		UPDATE jobs SET job_name = ?, job_type = ?, description = ?, 
			config_yaml = ?, flow_diagram = ?, scheduling_config = ?, 
			status = ?, updated_at = ?, enabled = ?, priority = ?, retry_policy = ?
		WHERE job_id = ?
	`
	_, err := s.db.Exec(query,
		job.JobName, job.JobType, job.Description, job.ConfigYAML,
		job.FlowDiagram, job.SchedulingConfig, job.Status, time.Now(),
		job.Enabled, job.Priority, job.RetryPolicy, job.JobID,
	)
	return err
}

// DeleteJob implements MetadataStore.DeleteJob
func (s *SQLiteStore) DeleteJob(jobID string) error {
	_, err := s.db.Exec("DELETE FROM jobs WHERE job_id = ?", jobID)
	return err
}

// SaveExecution implements MetadataStore.SaveExecution
func (s *SQLiteStore) SaveExecution(exec *models.Execution) error {
	query := `
		INSERT INTO executions (execution_id, job_id, status, start_time, 
			end_time, records_read, records_written, records_failed, 
			bytes_transferred, error_message, worker_id, checkpoint_data)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.Exec(query,
		exec.ExecutionID, exec.JobID, exec.Status, exec.StartTime,
		exec.EndTime, exec.RecordsRead, exec.RecordsWritten, exec.RecordsFailed,
		exec.BytesTransferred, exec.ErrorMessage, exec.WorkerID, exec.CheckpointData,
	)
	return err
}

// GetExecution implements MetadataStore.GetExecution
func (s *SQLiteStore) GetExecution(executionID string) (*models.Execution, error) {
	query := `
		SELECT execution_id, job_id, status, start_time, end_time, 
			records_read, records_written, records_failed, bytes_transferred, 
			error_message, worker_id, checkpoint_data
		FROM executions WHERE execution_id = ?
	`
	exec := &models.Execution{}
	err := s.db.QueryRow(query, executionID).Scan(
		&exec.ExecutionID, &exec.JobID, &exec.Status, &exec.StartTime, &exec.EndTime,
		&exec.RecordsRead, &exec.RecordsWritten, &exec.RecordsFailed,
		&exec.BytesTransferred, &exec.ErrorMessage, &exec.WorkerID, &exec.CheckpointData,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("execution not found: %s", executionID)
	}
	return exec, err
}

// GetExecutions implements MetadataStore.GetExecutions
func (s *SQLiteStore) GetExecutions(jobID string, limit int) ([]*models.Execution, error) {
	query := `
		SELECT execution_id, job_id, status, start_time, end_time, 
			records_read, records_written, records_failed, bytes_transferred, 
			error_message, worker_id, checkpoint_data
		FROM executions WHERE job_id = ? ORDER BY start_time DESC LIMIT ?
	`
	rows, err := s.db.Query(query, jobID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var executions []*models.Execution
	for rows.Next() {
		exec := &models.Execution{}
		err := rows.Scan(
			&exec.ExecutionID, &exec.JobID, &exec.Status, &exec.StartTime, &exec.EndTime,
			&exec.RecordsRead, &exec.RecordsWritten, &exec.RecordsFailed,
			&exec.BytesTransferred, &exec.ErrorMessage, &exec.WorkerID, &exec.CheckpointData,
		)
		if err != nil {
			return nil, err
		}
		executions = append(executions, exec)
	}
	return executions, rows.Err()
}

// UpdateExecution implements MetadataStore.UpdateExecution
func (s *SQLiteStore) UpdateExecution(exec *models.Execution) error {
	query := `
		UPDATE executions SET status = ?, end_time = ?, records_read = ?, 
			records_written = ?, records_failed = ?, bytes_transferred = ?, 
			error_message = ?, checkpoint_data = ?
		WHERE execution_id = ?
	`
	_, err := s.db.Exec(query,
		exec.Status, exec.EndTime, exec.RecordsRead, exec.RecordsWritten,
		exec.RecordsFailed, exec.BytesTransferred, exec.ErrorMessage,
		exec.CheckpointData, exec.ExecutionID,
	)
	return err
}

// RegisterWorker implements MetadataStore.RegisterWorker
func (s *SQLiteStore) RegisterWorker(worker *models.Worker) error {
	query := `
		INSERT OR REPLACE INTO workers (worker_id, hostname, ip_address, port, 
			status, cpu_cores, memory_mb, last_heartbeat, registered_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.Exec(query,
		worker.WorkerID, worker.Hostname, worker.IPAddress, worker.Port,
		worker.Status, worker.CPUCores, worker.MemoryMB, worker.LastHeartbeat,
		worker.RegisteredAt,
	)
	return err
}

// UpdateWorkerHeartbeat implements MetadataStore.UpdateWorkerHeartbeat
func (s *SQLiteStore) UpdateWorkerHeartbeat(workerID string) error {
	query := "UPDATE workers SET last_heartbeat = ?, status = ? WHERE worker_id = ?"
	_, err := s.db.Exec(query, time.Now(), models.WorkerStatusOnline, workerID)
	return err
}

// GetWorker implements MetadataStore.GetWorker
func (s *SQLiteStore) GetWorker(workerID string) (*models.Worker, error) {
	query := `
		SELECT worker_id, hostname, ip_address, port, status, cpu_cores, 
			memory_mb, last_heartbeat, registered_at
		FROM workers WHERE worker_id = ?
	`
	worker := &models.Worker{}
	err := s.db.QueryRow(query, workerID).Scan(
		&worker.WorkerID, &worker.Hostname, &worker.IPAddress, &worker.Port,
		&worker.Status, &worker.CPUCores, &worker.MemoryMB,
		&worker.LastHeartbeat, &worker.RegisteredAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("worker not found: %s", workerID)
	}
	return worker, err
}

// ListWorkers implements MetadataStore.ListWorkers
func (s *SQLiteStore) ListWorkers() ([]*models.Worker, error) {
	query := `
		SELECT worker_id, hostname, ip_address, port, status, cpu_cores, 
			memory_mb, last_heartbeat, registered_at
		FROM workers ORDER BY registered_at DESC
	`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workers []*models.Worker
	for rows.Next() {
		worker := &models.Worker{}
		err := rows.Scan(
			&worker.WorkerID, &worker.Hostname, &worker.IPAddress, &worker.Port,
			&worker.Status, &worker.CPUCores, &worker.MemoryMB,
			&worker.LastHeartbeat, &worker.RegisteredAt,
		)
		if err != nil {
			return nil, err
		}
		workers = append(workers, worker)
	}
	return workers, rows.Err()
}

// UnregisterWorker implements MetadataStore.UnregisterWorker
func (s *SQLiteStore) UnregisterWorker(workerID string) error {
	_, err := s.db.Exec("DELETE FROM workers WHERE worker_id = ?", workerID)
	return err
}

// RegisterPlugin implements MetadataStore.RegisterPlugin
func (s *SQLiteStore) RegisterPlugin(plugin *models.Plugin) error {
	query := `
		INSERT OR REPLACE INTO plugins (plugin_id, plugin_name, plugin_type, 
			version, data_source_type, config_schema, metadata, enabled, installed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.Exec(query,
		plugin.PluginID, plugin.PluginName, plugin.PluginType, plugin.Version,
		plugin.DataSourceType, plugin.ConfigSchema, plugin.Metadata,
		plugin.Enabled, plugin.InstalledAt,
	)
	return err
}

// GetPlugin implements MetadataStore.GetPlugin
func (s *SQLiteStore) GetPlugin(pluginName string) (*models.Plugin, error) {
	query := `
		SELECT plugin_id, plugin_name, plugin_type, version, data_source_type, 
			config_schema, metadata, enabled, installed_at
		FROM plugins WHERE plugin_name = ?
	`
	plugin := &models.Plugin{}
	err := s.db.QueryRow(query, pluginName).Scan(
		&plugin.PluginID, &plugin.PluginName, &plugin.PluginType, &plugin.Version,
		&plugin.DataSourceType, &plugin.ConfigSchema, &plugin.Metadata,
		&plugin.Enabled, &plugin.InstalledAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("plugin not found: %s", pluginName)
	}
	return plugin, err
}

// ListPlugins implements MetadataStore.ListPlugins
func (s *SQLiteStore) ListPlugins(pluginType string) ([]*models.Plugin, error) {
	var query string
	var rows *sql.Rows
	var err error

	if pluginType != "" {
		query = `
			SELECT plugin_id, plugin_name, plugin_type, version, data_source_type, 
				config_schema, metadata, enabled, installed_at
			FROM plugins WHERE plugin_type = ? ORDER BY plugin_name
		`
		rows, err = s.db.Query(query, pluginType)
	} else {
		query = `
			SELECT plugin_id, plugin_name, plugin_type, version, data_source_type, 
				config_schema, metadata, enabled, installed_at
			FROM plugins ORDER BY plugin_name
		`
		rows, err = s.db.Query(query)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plugins []*models.Plugin
	for rows.Next() {
		plugin := &models.Plugin{}
		err := rows.Scan(
			&plugin.PluginID, &plugin.PluginName, &plugin.PluginType, &plugin.Version,
			&plugin.DataSourceType, &plugin.ConfigSchema, &plugin.Metadata,
			&plugin.Enabled, &plugin.InstalledAt,
		)
		if err != nil {
			return nil, err
		}
		plugins = append(plugins, plugin)
	}
	return plugins, rows.Err()
}

// UpdatePluginStatus implements MetadataStore.UpdatePluginStatus
func (s *SQLiteStore) UpdatePluginStatus(pluginName string, enabled bool) error {
	_, err := s.db.Exec("UPDATE plugins SET enabled = ? WHERE plugin_name = ?", enabled, pluginName)
	return err
}
