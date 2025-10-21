package queue

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestTask(id string, jobID string, priority int) *Task {
	return &Task{
		ID:         id,
		JobID:      jobID,
		Priority:   priority,
		CreatedAt:  time.Now(),
		MaxRetries: 3,
	}
}

func TestNewMemoryQueue(t *testing.T) {
	queue := NewMemoryQueue(nil)
	assert.NotNil(t, queue)
	assert.Equal(t, 10000, queue.maxSize)
	assert.Equal(t, 0, queue.Size())
}

func TestNewMemoryQueueWithConfig(t *testing.T) {
	config := &Config{MaxSize: 100}
	queue := NewMemoryQueue(config)
	assert.Equal(t, 100, queue.maxSize)
}

func TestEnqueueDequeue(t *testing.T) {
	queue := NewMemoryQueue(nil)
	task := createTestTask("task-1", "job-1", 5)

	err := queue.Enqueue(task)
	require.NoError(t, err)
	assert.Equal(t, 1, queue.Size())

	ctx := context.Background()
	dequeued, err := queue.Dequeue(ctx)
	require.NoError(t, err)
	assert.Equal(t, task.ID, dequeued.ID)
	assert.Equal(t, 0, queue.Size())
}

func TestEnqueueMultiple(t *testing.T) {
	queue := NewMemoryQueue(nil)

	for i := 0; i < 5; i++ {
		task := createTestTask(string(rune('a'+i)), "job-1", i)
		err := queue.Enqueue(task)
		require.NoError(t, err)
	}

	assert.Equal(t, 5, queue.Size())
}

func TestPriorityOrdering(t *testing.T) {
	queue := NewMemoryQueue(nil)

	// Enqueue tasks with different priorities
	queue.Enqueue(createTestTask("low", "job-1", 1))
	queue.Enqueue(createTestTask("high", "job-1", 10))
	queue.Enqueue(createTestTask("medium", "job-1", 5))

	ctx := context.Background()

	// Should dequeue in priority order: high, medium, low
	task1, _ := queue.Dequeue(ctx)
	assert.Equal(t, "high", task1.ID)

	task2, _ := queue.Dequeue(ctx)
	assert.Equal(t, "medium", task2.ID)

	task3, _ := queue.Dequeue(ctx)
	assert.Equal(t, "low", task3.ID)
}

func TestEnqueueFullQueue(t *testing.T) {
	config := &Config{MaxSize: 2}
	queue := NewMemoryQueue(config)

	err := queue.Enqueue(createTestTask("task-1", "job-1", 1))
	require.NoError(t, err)

	err = queue.Enqueue(createTestTask("task-2", "job-1", 1))
	require.NoError(t, err)

	// Queue is full
	err = queue.Enqueue(createTestTask("task-3", "job-1", 1))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "queue is full")
}

func TestDequeueEmptyQueue(t *testing.T) {
	queue := NewMemoryQueue(nil)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Start dequeue in goroutine
	done := make(chan bool)
	go func() {
		_, err := queue.Dequeue(ctx)
		assert.Error(t, err)
		done <- true
	}()

	// Wait a bit then add a task to unblock
	time.Sleep(10 * time.Millisecond)
	queue.Close() // Close to unblock

	select {
	case <-done:
		// Success
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Dequeue did not return")
	}
}

func TestDequeueWithCancellation(t *testing.T) {
	queue := NewMemoryQueue(nil)

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately
	cancel()

	_, err := queue.Dequeue(ctx)
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestClear(t *testing.T) {
	queue := NewMemoryQueue(nil)

	for i := 0; i < 5; i++ {
		queue.Enqueue(createTestTask(string(rune('a'+i)), "job-1", i))
	}

	assert.Equal(t, 5, queue.Size())

	err := queue.Clear()
	require.NoError(t, err)
	assert.Equal(t, 0, queue.Size())
}

func TestClose(t *testing.T) {
	queue := NewMemoryQueue(nil)

	err := queue.Close()
	require.NoError(t, err)
	assert.True(t, queue.IsClosed())

	// Cannot enqueue to closed queue
	err = queue.Enqueue(createTestTask("task-1", "job-1", 1))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "queue is closed")
}

func TestCloseAlreadyClosed(t *testing.T) {
	queue := NewMemoryQueue(nil)

	err := queue.Close()
	require.NoError(t, err)

	err = queue.Close()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already closed")
}

func TestPeek(t *testing.T) {
	queue := NewMemoryQueue(nil)

	task := createTestTask("task-1", "job-1", 5)
	queue.Enqueue(task)

	peeked, err := queue.Peek()
	require.NoError(t, err)
	assert.Equal(t, task.ID, peeked.ID)

	// Task should still be in queue
	assert.Equal(t, 1, queue.Size())
}

func TestPeekEmptyQueue(t *testing.T) {
	queue := NewMemoryQueue(nil)

	_, err := queue.Peek()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "queue is empty")
}

func TestGetTasks(t *testing.T) {
	queue := NewMemoryQueue(nil)

	task1 := createTestTask("task-1", "job-1", 5)
	task2 := createTestTask("task-2", "job-1", 3)

	queue.Enqueue(task1)
	queue.Enqueue(task2)

	tasks := queue.GetTasks()
	assert.Equal(t, 2, len(tasks))
}

func TestRemoveTask(t *testing.T) {
	queue := NewMemoryQueue(nil)

	task1 := createTestTask("task-1", "job-1", 5)
	task2 := createTestTask("task-2", "job-1", 3)

	queue.Enqueue(task1)
	queue.Enqueue(task2)

	err := queue.RemoveTask("task-1")
	require.NoError(t, err)
	assert.Equal(t, 1, queue.Size())

	// Verify task-2 is still there
	tasks := queue.GetTasks()
	assert.Equal(t, "task-2", tasks[0].ID)
}

func TestRemoveTaskNotFound(t *testing.T) {
	queue := NewMemoryQueue(nil)

	err := queue.RemoveTask("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetStatistics(t *testing.T) {
	queue := NewMemoryQueue(nil)

	// Enqueue some tasks
	queue.Enqueue(createTestTask("task-1", "job-1", 1))
	queue.Enqueue(createTestTask("task-2", "job-1", 1))

	stats := queue.GetStatistics()

	assert.Equal(t, 2, stats["current_size"])
	assert.Equal(t, 10000, stats["max_size"])
	assert.Equal(t, int64(2), stats["enqueued"])
	assert.Equal(t, int64(0), stats["dequeued"])

	// Dequeue a task
	queue.Dequeue(context.Background())

	stats = queue.GetStatistics()
	assert.Equal(t, 1, stats["current_size"])
	assert.Equal(t, int64(1), stats["dequeued"])
}

func TestConcurrentEnqueueDequeue(t *testing.T) {
	queue := NewMemoryQueue(nil)
	done := make(chan bool)

	// Producer
	go func() {
		for i := 0; i < 100; i++ {
			task := createTestTask(string(rune(i)), "job-1", i%10)
			queue.Enqueue(task)
			time.Sleep(1 * time.Millisecond)
		}
		done <- true
	}()

	// Consumer
	dequeued := 0
	go func() {
		ctx := context.Background()
		for dequeued < 100 {
			_, err := queue.Dequeue(ctx)
			if err == nil {
				dequeued++
			}
		}
		done <- true
	}()

	// Wait for both to complete
	<-done
	<-done

	assert.Equal(t, 0, queue.Size())
	assert.Equal(t, 100, dequeued)
}

func TestNewPriorityQueue(t *testing.T) {
	priorities := []int{1, 5, 10}
	pq := NewPriorityQueue(priorities, nil)

	assert.NotNil(t, pq)
	assert.Equal(t, 3, len(pq.queues))
}

func TestPriorityQueueEnqueueDequeue(t *testing.T) {
	priorities := []int{1, 5, 10}
	pq := NewPriorityQueue(priorities, nil)

	// Enqueue tasks with different priorities
	pq.Enqueue(createTestTask("low", "job-1", 1))
	pq.Enqueue(createTestTask("high", "job-1", 10))
	pq.Enqueue(createTestTask("medium", "job-1", 5))

	// Should dequeue highest priority first
	ctx := context.Background()
	task1, err := pq.Dequeue(ctx)
	require.NoError(t, err)
	assert.Equal(t, 10, task1.Priority)

	task2, err := pq.Dequeue(ctx)
	require.NoError(t, err)
	assert.Equal(t, 5, task2.Priority)

	task3, err := pq.Dequeue(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, task3.Priority)
}

func TestPriorityQueueSize(t *testing.T) {
	priorities := []int{1, 5, 10}
	pq := NewPriorityQueue(priorities, nil)

	pq.Enqueue(createTestTask("task-1", "job-1", 1))
	pq.Enqueue(createTestTask("task-2", "job-1", 5))
	pq.Enqueue(createTestTask("task-3", "job-1", 10))

	assert.Equal(t, 3, pq.Size())
}

func TestPriorityQueueClose(t *testing.T) {
	priorities := []int{1, 5, 10}
	pq := NewPriorityQueue(priorities, nil)

	err := pq.Close()
	require.NoError(t, err)

	// All queues should be closed
	for _, queue := range pq.queues {
		assert.True(t, queue.IsClosed())
	}
}

func TestPriorityQueueInvalidPriority(t *testing.T) {
	priorities := []int{1, 5, 10}
	pq := NewPriorityQueue(priorities, nil)

	err := pq.Enqueue(createTestTask("task-1", "job-1", 99))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no queue for priority")
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	assert.NotNil(t, config)
	assert.Equal(t, 10000, config.MaxSize)
}
