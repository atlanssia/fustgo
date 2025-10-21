package worker

import (
	"fmt"
	"net"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/atlanssia/fustgo/internal/database"
	"github.com/atlanssia/fustgo/internal/logger"
	"github.com/atlanssia/fustgo/internal/models"
)

// Pool manages a pool of workers for distributed task execution
type Pool struct {
	mu                sync.RWMutex
	store             database.MetadataStore
	workers           map[string]*models.Worker // workerID -> worker
	heartbeatInterval time.Duration
	heartbeatTimeout  time.Duration
	stopChan          chan struct{}
	running           bool
}

// Config holds worker pool configuration
type Config struct {
	HeartbeatInterval time.Duration
	HeartbeatTimeout  time.Duration
}

// DefaultConfig returns default worker pool configuration
func DefaultConfig() *Config {
	return &Config{
		HeartbeatInterval: 30 * time.Second,
		HeartbeatTimeout:  90 * time.Second, // 3x heartbeat interval
	}
}

// NewPool creates a new worker pool
func NewPool(store database.MetadataStore, config *Config) *Pool {
	if config == nil {
		config = DefaultConfig()
	}

	return &Pool{
		store:             store,
		workers:           make(map[string]*models.Worker),
		heartbeatInterval: config.HeartbeatInterval,
		heartbeatTimeout:  config.HeartbeatTimeout,
		stopChan:          make(chan struct{}),
	}
}

// RegisterWorker registers a new worker in the pool
func (p *Pool) RegisterWorker(hostname string, port int) (*models.Worker, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Get IP address
	ipAddress, err := getLocalIP()
	if err != nil {
		ipAddress = "127.0.0.1"
		logger.Warn("Failed to get local IP, using 127.0.0.1: %v", err)
	}

	// Create worker
	worker := &models.Worker{
		WorkerID:      uuid.New().String(),
		Hostname:      hostname,
		IPAddress:     ipAddress,
		Port:          port,
		Status:        models.WorkerStatusOnline,
		CPUCores:      runtime.NumCPU(),
		MemoryMB:      getMemoryMB(),
		LastHeartbeat: time.Now(),
		RegisteredAt:  time.Now(),
	}

	// Save to database
	if err := p.store.RegisterWorker(worker); err != nil {
		return nil, fmt.Errorf("failed to register worker: %w", err)
	}

	// Add to in-memory map
	p.workers[worker.WorkerID] = worker

	logger.Info("Registered worker %s (%s:%d) with %d CPU cores, %d MB memory",
		worker.WorkerID, worker.IPAddress, worker.Port, worker.CPUCores, worker.MemoryMB)

	return worker, nil
}

// UnregisterWorker removes a worker from the pool
func (p *Pool) UnregisterWorker(workerID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.workers[workerID]; !exists {
		return fmt.Errorf("worker %s not found", workerID)
	}

	// Remove from database
	if err := p.store.UnregisterWorker(workerID); err != nil {
		return fmt.Errorf("failed to unregister worker: %w", err)
	}

	// Remove from memory
	delete(p.workers, workerID)

	logger.Info("Unregistered worker %s", workerID)
	return nil
}

// UpdateHeartbeat updates the heartbeat timestamp for a worker
func (p *Pool) UpdateHeartbeat(workerID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	worker, exists := p.workers[workerID]
	if !exists {
		// Try to load from database
		dbWorker, err := p.store.GetWorker(workerID)
		if err != nil {
			return fmt.Errorf("worker %s not found", workerID)
		}
		worker = dbWorker
		p.workers[workerID] = worker
	}

	// Update heartbeat
	worker.LastHeartbeat = time.Now()

	// Update in database
	if err := p.store.UpdateWorkerHeartbeat(workerID); err != nil {
		return fmt.Errorf("failed to update heartbeat: %w", err)
	}

	logger.Debug("Updated heartbeat for worker %s", workerID)
	return nil
}

// GetWorker retrieves a worker by ID
func (p *Pool) GetWorker(workerID string) (*models.Worker, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if worker, exists := p.workers[workerID]; exists {
		return worker, nil
	}

	// Try to load from database
	worker, err := p.store.GetWorker(workerID)
	if err != nil {
		return nil, fmt.Errorf("worker not found: %w", err)
	}

	return worker, nil
}

// ListWorkers returns all workers
func (p *Pool) ListWorkers() ([]*models.Worker, error) {
	workers, err := p.store.ListWorkers()
	if err != nil {
		return nil, fmt.Errorf("failed to list workers: %w", err)
	}

	// Update in-memory cache
	p.mu.Lock()
	for _, worker := range workers {
		p.workers[worker.WorkerID] = worker
	}
	p.mu.Unlock()

	return workers, nil
}

// GetHealthyWorkers returns all healthy workers
func (p *Pool) GetHealthyWorkers() []*models.Worker {
	p.mu.RLock()
	defer p.mu.RUnlock()

	healthy := make([]*models.Worker, 0)
	for _, worker := range p.workers {
		if worker.IsHealthy(p.heartbeatTimeout) {
			healthy = append(healthy, worker)
		}
	}

	return healthy
}

// GetWorkerCount returns the number of registered workers
func (p *Pool) GetWorkerCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.workers)
}

// Start starts the worker pool monitor
func (p *Pool) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return fmt.Errorf("worker pool is already running")
	}

	// Load existing workers from database
	workers, err := p.store.ListWorkers()
	if err != nil {
		logger.Warn("Failed to load existing workers: %v", err)
	} else {
		for _, worker := range workers {
			p.workers[worker.WorkerID] = worker
		}
		logger.Info("Loaded %d existing workers", len(workers))
	}

	p.running = true
	p.stopChan = make(chan struct{})

	// Start health check goroutine
	go p.monitorWorkers()

	logger.Info("Worker pool started with heartbeat interval: %v, timeout: %v",
		p.heartbeatInterval, p.heartbeatTimeout)

	return nil
}

// Stop stops the worker pool monitor
func (p *Pool) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return fmt.Errorf("worker pool is not running")
	}

	close(p.stopChan)
	p.running = false

	logger.Info("Worker pool stopped")
	return nil
}

// IsRunning returns whether the pool is running
func (p *Pool) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}

// monitorWorkers monitors worker health and removes unhealthy workers
func (p *Pool) monitorWorkers() {
	ticker := time.NewTicker(p.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.checkWorkerHealth()
		case <-p.stopChan:
			return
		}
	}
}

// checkWorkerHealth checks all workers and marks unhealthy ones as offline
func (p *Pool) checkWorkerHealth() {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	offlineCount := 0

	for workerID, worker := range p.workers {
		if now.Sub(worker.LastHeartbeat) > p.heartbeatTimeout {
			// Mark as offline
			if worker.Status != models.WorkerStatusOffline {
				worker.Status = models.WorkerStatusOffline
				offlineCount++
				logger.Warn("Worker %s (%s) marked as offline (no heartbeat for %v)",
					workerID, worker.Hostname, now.Sub(worker.LastHeartbeat))
			}
		} else if worker.Status == models.WorkerStatusOffline {
			// If worker was offline but now has recent heartbeat, mark as online
			worker.Status = models.WorkerStatusOnline
			logger.Info("Worker %s (%s) is back online", workerID, worker.Hostname)
		}
	}

	if offlineCount > 0 {
		logger.Info("Health check: %d workers marked offline", offlineCount)
	}
}

// GetPoolStats returns statistics about the worker pool
func (p *Pool) GetPoolStats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := map[string]interface{}{
		"total_workers":   len(p.workers),
		"online_workers":  0,
		"offline_workers": 0,
		"busy_workers":    0,
		"total_cpu_cores": 0,
		"total_memory_mb": 0,
	}

	for _, worker := range p.workers {
		switch worker.Status {
		case models.WorkerStatusOnline:
			stats["online_workers"] = stats["online_workers"].(int) + 1
		case models.WorkerStatusOffline:
			stats["offline_workers"] = stats["offline_workers"].(int) + 1
		case models.WorkerStatusBusy:
			stats["busy_workers"] = stats["busy_workers"].(int) + 1
		}

		stats["total_cpu_cores"] = stats["total_cpu_cores"].(int) + worker.CPUCores
		stats["total_memory_mb"] = stats["total_memory_mb"].(int) + worker.MemoryMB
	}

	return stats
}

// getLocalIP gets the local IP address
func getLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String(), nil
			}
		}
	}

	return "", fmt.Errorf("no valid IP address found")
}

// getMemoryMB returns approximate available memory in MB
func getMemoryMB() int {
	// This is a simple approximation
	// In production, you'd use runtime.MemStats or system calls
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return int(m.Sys / 1024 / 1024) // Convert bytes to MB
}

// GetWorkerHostname returns the current hostname
func GetWorkerHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}
