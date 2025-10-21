package checkpoint

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/atlanssia/fustgo/pkg/types"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.True(t, config.Enabled)
	assert.Equal(t, "file", config.StorageType)
	assert.Equal(t, "./data/checkpoints", config.StoragePath)
	assert.Equal(t, 30*time.Second, config.Interval)
}

func TestNewManager(t *testing.T) {
	tmpDir := t.TempDir()

	config := &Config{
		Enabled:     true,
		StorageType: "file",
		StoragePath: tmpDir,
		Interval:    10 * time.Second,
	}

	manager, err := NewManager("test-job-1", config)
	require.NoError(t, err)
	assert.NotNil(t, manager)
	assert.True(t, manager.enabled)
	assert.Equal(t, "test-job-1", manager.jobID)
}

func TestNewManagerUnsupportedStorage(t *testing.T) {
	config := &Config{
		Enabled:     true,
		StorageType: "redis", // Unsupported
		StoragePath: "/tmp",
	}

	_, err := NewManager("test-job", config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported storage type")
}

func TestSaveAndLoadCheckpoint(t *testing.T) {
	tmpDir := t.TempDir()

	config := &Config{
		Enabled:     true,
		StorageType: "file",
		StoragePath: tmpDir,
	}

	manager, err := NewManager("test-job-2", config)
	require.NoError(t, err)

	// Create a checkpoint
	checkpoint := &types.Checkpoint{
		Position: "12345", // Use string to avoid JSON type conversion issues
		Metadata: map[string]string{
			"file":   "data.csv",
			"offset": "12345",
		},
	}

	// Save checkpoint
	err = manager.SaveCheckpoint("input", checkpoint)
	require.NoError(t, err)

	// Load checkpoint
	loaded, err := manager.LoadCheckpoint("input")
	require.NoError(t, err)
	assert.NotNil(t, loaded)
	assert.Equal(t, "12345", loaded.Position)
	assert.Equal(t, "data.csv", loaded.Metadata["file"])
	assert.Equal(t, "12345", loaded.Metadata["offset"])
}

func TestSaveCheckpointDisabled(t *testing.T) {
	tmpDir := t.TempDir()

	config := &Config{
		Enabled:     false,
		StorageType: "file",
		StoragePath: tmpDir,
	}

	manager, err := NewManager("test-job-3", config)
	require.NoError(t, err)

	checkpoint := &types.Checkpoint{
		Position: 100,
	}

	// Should not error even when disabled
	err = manager.SaveCheckpoint("input", checkpoint)
	assert.NoError(t, err)

	// Should return nil when loading from disabled manager
	loaded, err := manager.LoadCheckpoint("input")
	assert.NoError(t, err)
	assert.Nil(t, loaded)
}

func TestGetAllCheckpoints(t *testing.T) {
	tmpDir := t.TempDir()

	config := &Config{
		Enabled:     true,
		StorageType: "file",
		StoragePath: tmpDir,
	}

	manager, err := NewManager("test-job-4", config)
	require.NoError(t, err)

	// Save multiple checkpoints
	checkpoints := map[string]*types.Checkpoint{
		"input": {
			Position: 100,
			Metadata: map[string]string{"stage": "input"},
		},
		"processor1": {
			Position: 200,
			Metadata: map[string]string{"stage": "processor1"},
		},
		"output": {
			Position: 300,
			Metadata: map[string]string{"stage": "output"},
		},
	}

	for stage, checkpoint := range checkpoints {
		err := manager.SaveCheckpoint(stage, checkpoint)
		require.NoError(t, err)
	}

	// Get all checkpoints
	all := manager.GetAllCheckpoints()
	assert.Equal(t, 3, len(all))
	assert.NotNil(t, all["input"])
	assert.NotNil(t, all["processor1"])
	assert.NotNil(t, all["output"])
}

func TestDeleteCheckpoint(t *testing.T) {
	tmpDir := t.TempDir()

	config := &Config{
		Enabled:     true,
		StorageType: "file",
		StoragePath: tmpDir,
	}

	manager, err := NewManager("test-job-5", config)
	require.NoError(t, err)

	// Save checkpoint
	checkpoint := &types.Checkpoint{Position: 100}
	err = manager.SaveCheckpoint("input", checkpoint)
	require.NoError(t, err)

	// Verify it exists
	loaded, err := manager.LoadCheckpoint("input")
	require.NoError(t, err)
	assert.NotNil(t, loaded)

	// Delete checkpoint
	err = manager.DeleteCheckpoint("input")
	require.NoError(t, err)

	// Verify it's gone
	loaded, err = manager.LoadCheckpoint("input")
	require.NoError(t, err)
	assert.Nil(t, loaded)
}

func TestClearCheckpoints(t *testing.T) {
	tmpDir := t.TempDir()

	config := &Config{
		Enabled:     true,
		StorageType: "file",
		StoragePath: tmpDir,
	}

	manager, err := NewManager("test-job-6", config)
	require.NoError(t, err)

	// Save multiple checkpoints
	for i := 0; i < 3; i++ {
		checkpoint := &types.Checkpoint{Position: i * 100}
		err := manager.SaveCheckpoint(string(rune('a'+i)), checkpoint)
		require.NoError(t, err)
	}

	// Verify they exist
	all := manager.GetAllCheckpoints()
	assert.Equal(t, 3, len(all))

	// Clear all
	err = manager.Clear()
	require.NoError(t, err)

	// Verify all are gone
	all = manager.GetAllCheckpoints()
	assert.Equal(t, 0, len(all))
}

func TestFileStorageSave(t *testing.T) {
	tmpDir := t.TempDir()

	storage, err := NewFileStorage(tmpDir)
	require.NoError(t, err)

	checkpoint := &types.Checkpoint{
		Position:  12345,
		Timestamp: time.Now(),
		Metadata: map[string]string{
			"key": "value",
		},
	}

	err = storage.Save("job-1", "stage-1", checkpoint)
	require.NoError(t, err)

	// Verify file exists
	filename := filepath.Join(tmpDir, "job-1", "stage-1.json")
	_, err = os.Stat(filename)
	assert.NoError(t, err)
}

func TestFileStorageLoad(t *testing.T) {
	tmpDir := t.TempDir()

	storage, err := NewFileStorage(tmpDir)
	require.NoError(t, err)

	original := &types.Checkpoint{
		Position: 99999,
		Metadata: map[string]string{
			"test": "data",
		},
	}

	err = storage.Save("job-2", "stage-2", original)
	require.NoError(t, err)

	loaded, err := storage.Load("job-2", "stage-2")
	require.NoError(t, err)
	assert.NotNil(t, loaded)
	assert.Equal(t, float64(99999), loaded.Position)
	assert.Equal(t, "data", loaded.Metadata["test"])
}

func TestFileStorageLoadNonExistent(t *testing.T) {
	tmpDir := t.TempDir()

	storage, err := NewFileStorage(tmpDir)
	require.NoError(t, err)

	loaded, err := storage.Load("non-existent-job", "non-existent-stage")
	require.NoError(t, err)
	assert.Nil(t, loaded)
}

func TestFileStorageList(t *testing.T) {
	tmpDir := t.TempDir()

	storage, err := NewFileStorage(tmpDir)
	require.NoError(t, err)

	// Save multiple checkpoints
	for i := 0; i < 3; i++ {
		checkpoint := &types.Checkpoint{Position: i * 100}
		stage := string(rune('a' + i))
		err := storage.Save("job-3", stage, checkpoint)
		require.NoError(t, err)
	}

	// List checkpoints
	checkpoints, err := storage.List("job-3")
	require.NoError(t, err)
	assert.Equal(t, 3, len(checkpoints))
	assert.NotNil(t, checkpoints["a"])
	assert.NotNil(t, checkpoints["b"])
	assert.NotNil(t, checkpoints["c"])
}

func TestFileStorageListNonExistentJob(t *testing.T) {
	tmpDir := t.TempDir()

	storage, err := NewFileStorage(tmpDir)
	require.NoError(t, err)

	checkpoints, err := storage.List("non-existent-job")
	require.NoError(t, err)
	assert.Equal(t, 0, len(checkpoints))
}

func TestFileStorageDelete(t *testing.T) {
	tmpDir := t.TempDir()

	storage, err := NewFileStorage(tmpDir)
	require.NoError(t, err)

	checkpoint := &types.Checkpoint{Position: 100}
	err = storage.Save("job-4", "stage-4", checkpoint)
	require.NoError(t, err)

	// Verify it exists
	filename := filepath.Join(tmpDir, "job-4", "stage-4.json")
	_, err = os.Stat(filename)
	assert.NoError(t, err)

	// Delete
	err = storage.Delete("job-4", "stage-4")
	require.NoError(t, err)

	// Verify it's gone
	_, err = os.Stat(filename)
	assert.True(t, os.IsNotExist(err))
}

func TestFileStorageClear(t *testing.T) {
	tmpDir := t.TempDir()

	storage, err := NewFileStorage(tmpDir)
	require.NoError(t, err)

	// Save multiple checkpoints
	for i := 0; i < 3; i++ {
		checkpoint := &types.Checkpoint{Position: i * 100}
		stage := string(rune('a' + i))
		err := storage.Save("job-5", stage, checkpoint)
		require.NoError(t, err)
	}

	// Verify directory exists
	jobDir := filepath.Join(tmpDir, "job-5")
	_, err = os.Stat(jobDir)
	assert.NoError(t, err)

	// Clear
	err = storage.Clear("job-5")
	require.NoError(t, err)

	// Verify directory is gone
	_, err = os.Stat(jobDir)
	assert.True(t, os.IsNotExist(err))
}

func TestManagerLoadExistingCheckpoints(t *testing.T) {
	tmpDir := t.TempDir()

	// Create storage and save checkpoints
	storage, err := NewFileStorage(tmpDir)
	require.NoError(t, err)

	checkpoint1 := &types.Checkpoint{Position: 100}
	checkpoint2 := &types.Checkpoint{Position: 200}

	err = storage.Save("job-6", "input", checkpoint1)
	require.NoError(t, err)
	err = storage.Save("job-6", "output", checkpoint2)
	require.NoError(t, err)

	// Create manager - should load existing checkpoints
	config := &Config{
		Enabled:     true,
		StorageType: "file",
		StoragePath: tmpDir,
	}

	manager, err := NewManager("job-6", config)
	require.NoError(t, err)

	// Verify checkpoints were loaded
	all := manager.GetAllCheckpoints()
	assert.Equal(t, 2, len(all))
	assert.NotNil(t, all["input"])
	assert.NotNil(t, all["output"])
}
