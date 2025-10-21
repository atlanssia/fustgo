package checkpoint

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/atlanssia/fustgo/internal/logger"
	"github.com/atlanssia/fustgo/pkg/types"
)

// Manager handles checkpoint persistence for fault recovery
type Manager struct {
	mu          sync.RWMutex
	jobID       string
	checkpoints map[string]*types.Checkpoint // stage name -> checkpoint
	storage     Storage
	interval    time.Duration
	enabled     bool
}

// Storage defines the interface for checkpoint persistence
type Storage interface {
	Save(jobID string, stage string, checkpoint *types.Checkpoint) error
	Load(jobID string, stage string) (*types.Checkpoint, error)
	List(jobID string) (map[string]*types.Checkpoint, error)
	Delete(jobID string, stage string) error
	Clear(jobID string) error
}

// Config holds configuration for checkpoint manager
type Config struct {
	Enabled     bool
	StorageType string        // "file" or "database"
	StoragePath string        // For file storage
	Interval    time.Duration // Auto-save interval
}

// DefaultConfig returns default checkpoint configuration
func DefaultConfig() *Config {
	return &Config{
		Enabled:     true,
		StorageType: "file",
		StoragePath: "./data/checkpoints",
		Interval:    30 * time.Second,
	}
}

// NewManager creates a new checkpoint manager
func NewManager(jobID string, config *Config) (*Manager, error) {
	if config == nil {
		config = DefaultConfig()
	}

	var storage Storage
	var err error

	switch config.StorageType {
	case "file":
		storage, err = NewFileStorage(config.StoragePath)
		if err != nil {
			return nil, fmt.Errorf("failed to create file storage: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", config.StorageType)
	}

	manager := &Manager{
		jobID:       jobID,
		checkpoints: make(map[string]*types.Checkpoint),
		storage:     storage,
		interval:    config.Interval,
		enabled:     config.Enabled,
	}

	// Load existing checkpoints
	if config.Enabled {
		existing, err := storage.List(jobID)
		if err != nil {
			logger.Warn("Failed to load existing checkpoints: %v", err)
		} else {
			manager.checkpoints = existing
			logger.Info("Loaded %d existing checkpoints for job %s", len(existing), jobID)
		}
	}

	return manager, nil
}

// SaveCheckpoint saves a checkpoint for a specific stage
func (m *Manager) SaveCheckpoint(stage string, checkpoint *types.Checkpoint) error {
	if !m.enabled {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	checkpoint.Timestamp = time.Now()
	m.checkpoints[stage] = checkpoint

	if err := m.storage.Save(m.jobID, stage, checkpoint); err != nil {
		return fmt.Errorf("failed to save checkpoint: %w", err)
	}

	logger.Debug("Saved checkpoint for stage %s", stage)
	return nil
}

// LoadCheckpoint loads a checkpoint for a specific stage
func (m *Manager) LoadCheckpoint(stage string) (*types.Checkpoint, error) {
	if !m.enabled {
		return nil, nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	checkpoint, exists := m.checkpoints[stage]
	if !exists {
		return nil, nil
	}

	logger.Debug("Loaded checkpoint for stage %s from %v", stage, checkpoint.Timestamp)
	return checkpoint, nil
}

// GetAllCheckpoints returns all checkpoints
func (m *Manager) GetAllCheckpoints() map[string]*types.Checkpoint {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*types.Checkpoint)
	for k, v := range m.checkpoints {
		result[k] = v
	}
	return result
}

// DeleteCheckpoint deletes a checkpoint for a specific stage
func (m *Manager) DeleteCheckpoint(stage string) error {
	if !m.enabled {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.checkpoints, stage)

	if err := m.storage.Delete(m.jobID, stage); err != nil {
		return fmt.Errorf("failed to delete checkpoint: %w", err)
	}

	logger.Debug("Deleted checkpoint for stage %s", stage)
	return nil
}

// Clear clears all checkpoints
func (m *Manager) Clear() error {
	if !m.enabled {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.checkpoints = make(map[string]*types.Checkpoint)

	if err := m.storage.Clear(m.jobID); err != nil {
		return fmt.Errorf("failed to clear checkpoints: %w", err)
	}

	logger.Info("Cleared all checkpoints for job %s", m.jobID)
	return nil
}

// IsEnabled returns whether checkpointing is enabled
func (m *Manager) IsEnabled() bool {
	return m.enabled
}

// FileStorage implements file-based checkpoint storage
type FileStorage struct {
	basePath string
}

// NewFileStorage creates a new file-based checkpoint storage
func NewFileStorage(basePath string) (*FileStorage, error) {
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create checkpoint directory: %w", err)
	}

	return &FileStorage{
		basePath: basePath,
	}, nil
}

// Save saves a checkpoint to file
func (fs *FileStorage) Save(jobID string, stage string, checkpoint *types.Checkpoint) error {
	jobDir := filepath.Join(fs.basePath, jobID)
	if err := os.MkdirAll(jobDir, 0755); err != nil {
		return fmt.Errorf("failed to create job directory: %w", err)
	}

	filename := filepath.Join(jobDir, fmt.Sprintf("%s.json", stage))
	data, err := json.MarshalIndent(checkpoint, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write checkpoint file: %w", err)
	}

	return nil
}

// Load loads a checkpoint from file
func (fs *FileStorage) Load(jobID string, stage string) (*types.Checkpoint, error) {
	filename := filepath.Join(fs.basePath, jobID, fmt.Sprintf("%s.json", stage))

	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read checkpoint file: %w", err)
	}

	var checkpoint types.Checkpoint
	if err := json.Unmarshal(data, &checkpoint); err != nil {
		return nil, fmt.Errorf("failed to unmarshal checkpoint: %w", err)
	}

	return &checkpoint, nil
}

// List lists all checkpoints for a job
func (fs *FileStorage) List(jobID string) (map[string]*types.Checkpoint, error) {
	jobDir := filepath.Join(fs.basePath, jobID)

	if _, err := os.Stat(jobDir); os.IsNotExist(err) {
		return make(map[string]*types.Checkpoint), nil
	}

	entries, err := os.ReadDir(jobDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read job directory: %w", err)
	}

	checkpoints := make(map[string]*types.Checkpoint)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		stage := entry.Name()[:len(entry.Name())-5] // Remove .json extension
		checkpoint, err := fs.Load(jobID, stage)
		if err != nil {
			logger.Warn("Failed to load checkpoint %s: %v", stage, err)
			continue
		}

		if checkpoint != nil {
			checkpoints[stage] = checkpoint
		}
	}

	return checkpoints, nil
}

// Delete deletes a checkpoint file
func (fs *FileStorage) Delete(jobID string, stage string) error {
	filename := filepath.Join(fs.basePath, jobID, fmt.Sprintf("%s.json", stage))

	if err := os.Remove(filename); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete checkpoint file: %w", err)
	}

	return nil
}

// Clear clears all checkpoints for a job
func (fs *FileStorage) Clear(jobID string) error {
	jobDir := filepath.Join(fs.basePath, jobID)

	if err := os.RemoveAll(jobDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove job directory: %w", err)
	}

	return nil
}
