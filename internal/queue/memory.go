package queue

import (
	"container/list"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/atlanssia/fustgo/internal/logger"
)

// Task represents a task in the queue
type Task struct {
	ID        string
	JobID     string
	Priority  int
	CreatedAt time.Time
	Data      interface{}
	Retries   int
	MaxRetries int
}

// TaskQueue defines the interface for task queues
type TaskQueue interface {
	Enqueue(task *Task) error
	Dequeue(ctx context.Context) (*Task, error)
	Size() int
	Clear() error
	Close() error
}

// MemoryQueue implements an in-memory task queue with priority support
type MemoryQueue struct {
	mu         sync.RWMutex
	tasks      *list.List
	notEmpty   *sync.Cond
	maxSize    int
	closed     bool
	
	// Statistics
	enqueued   int64
	dequeued   int64
	failed     int64
}

// Config holds memory queue configuration
type Config struct {
	MaxSize int
}

// DefaultConfig returns default queue configuration
func DefaultConfig() *Config {
	return &Config{
		MaxSize: 10000,
	}
}

// NewMemoryQueue creates a new in-memory task queue
func NewMemoryQueue(config *Config) *MemoryQueue {
	if config == nil {
		config = DefaultConfig()
	}

	q := &MemoryQueue{
		tasks:   list.New(),
		maxSize: config.MaxSize,
	}
	q.notEmpty = sync.NewCond(&q.mu)

	logger.Info("Created memory queue with max size: %d", config.MaxSize)
	return q
}

// Enqueue adds a task to the queue
func (q *MemoryQueue) Enqueue(task *Task) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return fmt.Errorf("queue is closed")
	}

	if q.tasks.Len() >= q.maxSize {
		return fmt.Errorf("queue is full (max size: %d)", q.maxSize)
	}

	// Insert task based on priority (higher priority first)
	inserted := false
	for e := q.tasks.Front(); e != nil; e = e.Next() {
		t := e.Value.(*Task)
		if task.Priority > t.Priority {
			q.tasks.InsertBefore(task, e)
			inserted = true
			break
		}
	}

	if !inserted {
		q.tasks.PushBack(task)
	}

	q.enqueued++
	q.notEmpty.Signal()

	logger.Debug("Enqueued task %s (job: %s, priority: %d)", task.ID, task.JobID, task.Priority)
	return nil
}

// Dequeue removes and returns a task from the queue
func (q *MemoryQueue) Dequeue(ctx context.Context) (*Task, error) {
	for {
		// Check context first
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		q.mu.Lock()

		// Check if closed
		if q.closed {
			q.mu.Unlock()
			return nil, fmt.Errorf("queue is closed")
		}

		// Try to get task
		if q.tasks.Len() > 0 {
			element := q.tasks.Front()
			task := element.Value.(*Task)
			q.tasks.Remove(element)
			q.dequeued++
			q.mu.Unlock()

			logger.Debug("Dequeued task %s (job: %s)", task.ID, task.JobID)
			return task, nil
		}

		// Wait for task
		q.notEmpty.Wait()
		q.mu.Unlock()
	}
}

// Size returns the current queue size
func (q *MemoryQueue) Size() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.tasks.Len()
}

// Clear removes all tasks from the queue
func (q *MemoryQueue) Clear() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return fmt.Errorf("queue is closed")
	}

	count := q.tasks.Len()
	q.tasks.Init()
	
	logger.Info("Cleared %d tasks from queue", count)
	return nil
}

// Close closes the queue
func (q *MemoryQueue) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return fmt.Errorf("queue is already closed")
	}

	q.closed = true
	q.notEmpty.Broadcast() // Wake up all waiting dequeuers

	logger.Info("Closed memory queue")
	return nil
}

// IsClosed returns whether the queue is closed
func (q *MemoryQueue) IsClosed() bool {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.closed
}

// GetStatistics returns queue statistics
func (q *MemoryQueue) GetStatistics() map[string]interface{} {
	q.mu.RLock()
	defer q.mu.RUnlock()

	return map[string]interface{}{
		"current_size": q.tasks.Len(),
		"max_size":     q.maxSize,
		"enqueued":     q.enqueued,
		"dequeued":     q.dequeued,
		"failed":       q.failed,
		"utilization":  float64(q.tasks.Len()) / float64(q.maxSize) * 100,
	}
}

// Peek returns the next task without removing it
func (q *MemoryQueue) Peek() (*Task, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if q.closed {
		return nil, fmt.Errorf("queue is closed")
	}

	if q.tasks.Len() == 0 {
		return nil, fmt.Errorf("queue is empty")
	}

	task := q.tasks.Front().Value.(*Task)
	return task, nil
}

// GetTasks returns all tasks (for inspection/debugging)
func (q *MemoryQueue) GetTasks() []*Task {
	q.mu.RLock()
	defer q.mu.RUnlock()

	tasks := make([]*Task, 0, q.tasks.Len())
	for e := q.tasks.Front(); e != nil; e = e.Next() {
		tasks = append(tasks, e.Value.(*Task))
	}
	return tasks
}

// RemoveTask removes a specific task by ID
func (q *MemoryQueue) RemoveTask(taskID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return fmt.Errorf("queue is closed")
	}

	for e := q.tasks.Front(); e != nil; e = e.Next() {
		task := e.Value.(*Task)
		if task.ID == taskID {
			q.tasks.Remove(e)
			logger.Debug("Removed task %s from queue", taskID)
			return nil
		}
	}

	return fmt.Errorf("task %s not found in queue", taskID)
}

// PriorityQueue is a helper type for managing task priorities
type PriorityQueue struct {
	queues map[int]*MemoryQueue // priority -> queue
	mu     sync.RWMutex
}

// NewPriorityQueue creates a new priority-based queue system
func NewPriorityQueue(priorities []int, config *Config) *PriorityQueue {
	pq := &PriorityQueue{
		queues: make(map[int]*MemoryQueue),
	}

	for _, priority := range priorities {
		pq.queues[priority] = NewMemoryQueue(config)
	}

	logger.Info("Created priority queue with %d priority levels", len(priorities))
	return pq
}

// Enqueue adds a task to the appropriate priority queue
func (pq *PriorityQueue) Enqueue(task *Task) error {
	pq.mu.RLock()
	queue, exists := pq.queues[task.Priority]
	pq.mu.RUnlock()

	if !exists {
		return fmt.Errorf("no queue for priority %d", task.Priority)
	}

	return queue.Enqueue(task)
}

// Dequeue retrieves a task from the highest priority non-empty queue
func (pq *PriorityQueue) Dequeue(ctx context.Context) (*Task, error) {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	// Try queues in descending priority order
	priorities := make([]int, 0, len(pq.queues))
	for p := range pq.queues {
		priorities = append(priorities, p)
	}

	// Sort priorities (descending)
	for i := 0; i < len(priorities); i++ {
		for j := i + 1; j < len(priorities); j++ {
			if priorities[i] < priorities[j] {
				priorities[i], priorities[j] = priorities[j], priorities[i]
			}
		}
	}

	// Try each priority queue
	for _, priority := range priorities {
		queue := pq.queues[priority]
		if queue.Size() > 0 {
			return queue.Dequeue(ctx)
		}
	}

	// All queues empty, wait on highest priority queue
	if len(priorities) > 0 {
		return pq.queues[priorities[0]].Dequeue(ctx)
	}

	return nil, fmt.Errorf("no queues available")
}

// Size returns the total size across all queues
func (pq *PriorityQueue) Size() int {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	total := 0
	for _, queue := range pq.queues {
		total += queue.Size()
	}
	return total
}

// Close closes all queues
func (pq *PriorityQueue) Close() error {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	for _, queue := range pq.queues {
		queue.Close()
	}
	return nil
}
