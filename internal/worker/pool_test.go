package worker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/atlanssia/fustgo/internal/database"
	"github.com/atlanssia/fustgo/internal/models"
)

func setupTestPool(t *testing.T) *Pool {
	tmpDB := t.TempDir() + "/test.db"
	store, err := database.NewSQLiteStore(tmpDB)
	require.NoError(t, err)

	return NewPool(store, nil)
}

func TestNewPool(t *testing.T) {
	pool := setupTestPool(t)
	assert.NotNil(t, pool)
	assert.NotNil(t, pool.store)
	assert.NotNil(t, pool.workers)
	assert.Equal(t, 30*time.Second, pool.heartbeatInterval)
	assert.Equal(t, 90*time.Second, pool.heartbeatTimeout)
}

func TestNewPoolWithConfig(t *testing.T) {
	tmpDB := t.TempDir() + "/test.db"
	store, err := database.NewSQLiteStore(tmpDB)
	require.NoError(t, err)

	config := &Config{
		HeartbeatInterval: 10 * time.Second,
		HeartbeatTimeout:  30 * time.Second,
	}

	pool := NewPool(store, config)
	assert.Equal(t, 10*time.Second, pool.heartbeatInterval)
	assert.Equal(t, 30*time.Second, pool.heartbeatTimeout)
}

func TestRegisterWorker(t *testing.T) {
	pool := setupTestPool(t)

	worker, err := pool.RegisterWorker("test-host", 8080)
	require.NoError(t, err)

	assert.NotEmpty(t, worker.WorkerID)
	assert.Equal(t, "test-host", worker.Hostname)
	assert.Equal(t, 8080, worker.Port)
	assert.Equal(t, models.WorkerStatusOnline, worker.Status)
	assert.True(t, worker.CPUCores > 0)
	assert.True(t, worker.MemoryMB > 0)
	assert.False(t, worker.RegisteredAt.IsZero())
	assert.False(t, worker.LastHeartbeat.IsZero())
}

func TestUnregisterWorker(t *testing.T) {
	pool := setupTestPool(t)

	// Register worker
	worker, err := pool.RegisterWorker("test-host", 8080)
	require.NoError(t, err)

	// Unregister worker
	err = pool.UnregisterWorker(worker.WorkerID)
	require.NoError(t, err)

	// Verify worker is removed
	_, err = pool.GetWorker(worker.WorkerID)
	assert.Error(t, err)
}

func TestUnregisterWorkerNotFound(t *testing.T) {
	pool := setupTestPool(t)

	err := pool.UnregisterWorker("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestUpdateHeartbeat(t *testing.T) {
	pool := setupTestPool(t)

	// Register worker
	worker, err := pool.RegisterWorker("test-host", 8080)
	require.NoError(t, err)

	// Wait a bit
	time.Sleep(100 * time.Millisecond)
	oldHeartbeat := worker.LastHeartbeat

	// Update heartbeat
	err = pool.UpdateHeartbeat(worker.WorkerID)
	require.NoError(t, err)

	// Verify heartbeat was updated
	updated, err := pool.GetWorker(worker.WorkerID)
	require.NoError(t, err)
	assert.True(t, updated.LastHeartbeat.After(oldHeartbeat))
}

func TestUpdateHeartbeatNotFound(t *testing.T) {
	pool := setupTestPool(t)

	err := pool.UpdateHeartbeat("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetWorker(t *testing.T) {
	pool := setupTestPool(t)

	// Register worker
	worker, err := pool.RegisterWorker("test-host", 8080)
	require.NoError(t, err)

	// Get worker
	retrieved, err := pool.GetWorker(worker.WorkerID)
	require.NoError(t, err)
	assert.Equal(t, worker.WorkerID, retrieved.WorkerID)
	assert.Equal(t, worker.Hostname, retrieved.Hostname)
}

func TestListWorkers(t *testing.T) {
	pool := setupTestPool(t)

	// Register multiple workers
	for i := 0; i < 3; i++ {
		_, err := pool.RegisterWorker("test-host", 8080+i)
		require.NoError(t, err)
	}

	// List workers
	workers, err := pool.ListWorkers()
	require.NoError(t, err)
	assert.Equal(t, 3, len(workers))
}

func TestGetHealthyWorkers(t *testing.T) {
	pool := setupTestPool(t)

	// Register workers
	worker1, err := pool.RegisterWorker("worker-1", 8080)
	require.NoError(t, err)

	worker2, err := pool.RegisterWorker("worker-2", 8081)
	require.NoError(t, err)

	// Make worker2 stale (simulate missed heartbeats)
	pool.mu.Lock()
	pool.workers[worker2.WorkerID].LastHeartbeat = time.Now().Add(-2 * pool.heartbeatTimeout)
	pool.mu.Unlock()

	// Get healthy workers
	healthy := pool.GetHealthyWorkers()
	assert.Equal(t, 1, len(healthy))
	assert.Equal(t, worker1.WorkerID, healthy[0].WorkerID)
}

func TestGetWorkerCount(t *testing.T) {
	pool := setupTestPool(t)

	assert.Equal(t, 0, pool.GetWorkerCount())

	// Register workers
	pool.RegisterWorker("worker-1", 8080)
	assert.Equal(t, 1, pool.GetWorkerCount())

	pool.RegisterWorker("worker-2", 8081)
	assert.Equal(t, 2, pool.GetWorkerCount())
}

func TestStartPool(t *testing.T) {
	pool := setupTestPool(t)

	err := pool.Start()
	require.NoError(t, err)
	assert.True(t, pool.IsRunning())

	// Cleanup
	pool.Stop()
}

func TestStartPoolAlreadyRunning(t *testing.T) {
	pool := setupTestPool(t)

	err := pool.Start()
	require.NoError(t, err)

	// Try to start again
	err = pool.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	// Cleanup
	pool.Stop()
}

func TestStopPool(t *testing.T) {
	pool := setupTestPool(t)

	err := pool.Start()
	require.NoError(t, err)

	err = pool.Stop()
	require.NoError(t, err)
	assert.False(t, pool.IsRunning())
}

func TestStopPoolNotRunning(t *testing.T) {
	pool := setupTestPool(t)

	err := pool.Stop()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
}

func TestMonitorWorkers(t *testing.T) {
	tmpDB := t.TempDir() + "/test.db"
	store, err := database.NewSQLiteStore(tmpDB)
	require.NoError(t, err)

	// Use short intervals for testing
	config := &Config{
		HeartbeatInterval: 100 * time.Millisecond,
		HeartbeatTimeout:  200 * time.Millisecond,
	}

	pool := NewPool(store, config)

	// Start pool
	err = pool.Start()
	require.NoError(t, err)
	defer pool.Stop()

	// Register worker
	worker, err := pool.RegisterWorker("test-worker", 8080)
	require.NoError(t, err)

	// Make worker stale
	pool.mu.Lock()
	pool.workers[worker.WorkerID].LastHeartbeat = time.Now().Add(-300 * time.Millisecond)
	pool.mu.Unlock()

	// Wait for monitor to detect
	time.Sleep(250 * time.Millisecond)

	// Check worker status
	pool.mu.RLock()
	status := pool.workers[worker.WorkerID].Status
	pool.mu.RUnlock()

	assert.Equal(t, models.WorkerStatusOffline, status)
}

func TestGetPoolStats(t *testing.T) {
	pool := setupTestPool(t)

	// Register workers with different statuses
	worker1, err := pool.RegisterWorker("worker-1", 8080)
	require.NoError(t, err)

	worker2, err := pool.RegisterWorker("worker-2", 8081)
	require.NoError(t, err)

	// Make worker2 offline
	pool.mu.Lock()
	pool.workers[worker2.WorkerID].Status = models.WorkerStatusOffline
	pool.mu.Unlock()

	// Get stats
	stats := pool.GetPoolStats()

	assert.Equal(t, 2, stats["total_workers"])
	assert.Equal(t, 1, stats["online_workers"])
	assert.Equal(t, 1, stats["offline_workers"])
	assert.True(t, stats["total_cpu_cores"].(int) >= 2) // At least 2 cores
	assert.True(t, stats["total_memory_mb"].(int) > 0)

	// Cleanup
	pool.UnregisterWorker(worker1.WorkerID)
}

func TestGetLocalIP(t *testing.T) {
	ip, err := getLocalIP()
	
	// May fail in some test environments, so we check both cases
	if err != nil {
		t.Logf("Failed to get local IP (expected in some environments): %v", err)
	} else {
		assert.NotEmpty(t, ip)
		assert.NotEqual(t, "127.0.0.1", ip)
		t.Logf("Local IP: %s", ip)
	}
}

func TestGetMemoryMB(t *testing.T) {
	memory := getMemoryMB()
	assert.True(t, memory > 0)
	t.Logf("Memory: %d MB", memory)
}

func TestGetWorkerHostname(t *testing.T) {
	hostname := GetWorkerHostname()
	assert.NotEmpty(t, hostname)
	assert.NotEqual(t, "unknown", hostname)
	t.Logf("Hostname: %s", hostname)
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	assert.NotNil(t, config)
	assert.Equal(t, 30*time.Second, config.HeartbeatInterval)
	assert.Equal(t, 90*time.Second, config.HeartbeatTimeout)
}

func TestWorkerHealthRecovery(t *testing.T) {
	tmpDB := t.TempDir() + "/test.db"
	store, err := database.NewSQLiteStore(tmpDB)
	require.NoError(t, err)

	config := &Config{
		HeartbeatInterval: 50 * time.Millisecond,
		HeartbeatTimeout:  100 * time.Millisecond,
	}

	pool := NewPool(store, config)
	err = pool.Start()
	require.NoError(t, err)
	defer pool.Stop()

	// Register worker
	worker, err := pool.RegisterWorker("test-worker", 8080)
	require.NoError(t, err)

	// Make worker offline
	pool.mu.Lock()
	pool.workers[worker.WorkerID].LastHeartbeat = time.Now().Add(-150 * time.Millisecond)
	pool.workers[worker.WorkerID].Status = models.WorkerStatusOffline
	pool.mu.Unlock()

	// Wait for health check
	time.Sleep(100 * time.Millisecond)

	// Update heartbeat (worker comes back online)
	err = pool.UpdateHeartbeat(worker.WorkerID)
	require.NoError(t, err)

	// Wait for another health check
	time.Sleep(100 * time.Millisecond)

	// Verify worker is back online
	pool.mu.RLock()
	status := pool.workers[worker.WorkerID].Status
	pool.mu.RUnlock()

	assert.Equal(t, models.WorkerStatusOnline, status)
}
