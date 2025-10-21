package pipeline

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/atlanssia/fustgo/internal/checkpoint"
	"github.com/atlanssia/fustgo/internal/logger"
	"github.com/atlanssia/fustgo/pkg/types"
)

// ConcurrentPipeline represents a high-performance pipeline with buffered channels and backpressure
type ConcurrentPipeline struct {
	input       types.InputPlugin
	processors  []types.ProcessorPlugin
	output      types.OutputPlugin
	batchSize   int
	
	// Channel configuration
	inputBufferSize      int
	processorBufferSize  int
	outputBufferSize     int
	
	// Backpressure thresholds
	backpressureThreshold float64 // 0.0 to 1.0, triggers when buffer is this full
	
	// Checkpoint management
	checkpointManager *checkpoint.Manager
	
	// Error handling
	errorChan chan error
	
	// Metrics
	mu                sync.RWMutex
	totalBatches      int64
	totalRecords      int64
	failedRecords     int64
	startTime         time.Time
	endTime           time.Time
}

// ConcurrentPipelineConfig holds configuration for concurrent pipeline
type ConcurrentPipelineConfig struct {
	BatchSize             int
	InputBufferSize       int
	ProcessorBufferSize   int
	OutputBufferSize      int
	BackpressureThreshold float64
	JobID                 string
	CheckpointConfig      *checkpoint.Config
}

// DefaultConcurrentConfig returns default configuration
func DefaultConcurrentConfig() *ConcurrentPipelineConfig {
	return &ConcurrentPipelineConfig{
		BatchSize:             1000,
		InputBufferSize:       10,   // Buffer 10 batches
		ProcessorBufferSize:   10,
		OutputBufferSize:      5,
		BackpressureThreshold: 0.8,  // Trigger backpressure at 80% full
	}
}

// NewConcurrentPipeline creates a new concurrent pipeline
func NewConcurrentPipeline(
	input types.InputPlugin,
	processors []types.ProcessorPlugin,
	output types.OutputPlugin,
	config *ConcurrentPipelineConfig,
) *ConcurrentPipeline {
	if config == nil {
		config = DefaultConcurrentConfig()
	}
	
	pipeline := &ConcurrentPipeline{
		input:                 input,
		processors:            processors,
		output:                output,
		batchSize:             config.BatchSize,
		inputBufferSize:       config.InputBufferSize,
		processorBufferSize:   config.ProcessorBufferSize,
		outputBufferSize:      config.OutputBufferSize,
		backpressureThreshold: config.BackpressureThreshold,
		errorChan:             make(chan error, 10),
	}
	
	// Initialize checkpoint manager if configured
	if config.CheckpointConfig != nil && config.CheckpointConfig.Enabled {
		if config.JobID == "" {
			config.JobID = fmt.Sprintf("pipeline-%d", time.Now().Unix())
		}
		
		manager, err := checkpoint.NewManager(config.JobID, config.CheckpointConfig)
		if err != nil {
			logger.Warn("Failed to create checkpoint manager: %v", err)
		} else {
			pipeline.checkpointManager = manager
			logger.Info("Checkpoint manager enabled for job %s", config.JobID)
		}
	}
	
	return pipeline
}

// Execute executes the pipeline with concurrent processing
func (p *ConcurrentPipeline) Execute(ctx context.Context) error {
	p.startTime = time.Now()
	logger.Info("Starting concurrent pipeline execution")
	
	// Connect input
	if err := p.input.Connect(); err != nil {
		return fmt.Errorf("failed to connect input: %w", err)
	}
	defer p.input.Close()
	
	// Connect output
	if err := p.output.Connect(); err != nil {
		return fmt.Errorf("failed to connect output: %w", err)
	}
	defer p.output.Close()
	
	// Create buffered channels
	inputChan := make(chan *types.DataBatch, p.inputBufferSize)
	processorChans := make([]chan *types.DataBatch, len(p.processors)+1)
	processorChans[0] = inputChan
	for i := 0; i < len(p.processors); i++ {
		processorChans[i+1] = make(chan *types.DataBatch, p.processorBufferSize)
	}
	outputChan := processorChans[len(p.processors)]
	
	// WaitGroup for goroutines
	var wg sync.WaitGroup
	
	// Context for cancellation
	pipelineCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	
	// Start input reader
	wg.Add(1)
	go p.runInputReader(pipelineCtx, inputChan, &wg)
	
	// Start processors
	for i, processor := range p.processors {
		wg.Add(1)
		go p.runProcessor(pipelineCtx, processor, processorChans[i], processorChans[i+1], i, &wg)
	}
	
	// Start output writer
	wg.Add(1)
	go p.runOutputWriter(pipelineCtx, outputChan, &wg)
	
	// Wait for completion or error
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		logger.Info("Pipeline completed successfully")
	case err := <-p.errorChan:
		cancel()
		wg.Wait()
		return fmt.Errorf("pipeline error: %w", err)
	case <-ctx.Done():
		cancel()
		wg.Wait()
		return fmt.Errorf("pipeline cancelled: %w", ctx.Err())
	}
	
	// Flush output
	if err := p.output.Flush(); err != nil {
		return fmt.Errorf("failed to flush output: %w", err)
	}
	
	p.endTime = time.Now()
	p.logStatistics()
	
	return nil
}

// runInputReader reads batches from input and sends to channel
func (p *ConcurrentPipeline) runInputReader(ctx context.Context, outputChan chan<- *types.DataBatch, wg *sync.WaitGroup) {
	defer wg.Done()
	defer close(outputChan)
	
	logger.Info("Input reader started")
	batchCount := 0
	
	for {
		select {
		case <-ctx.Done():
			logger.Info("Input reader cancelled")
			return
		default:
			// Check backpressure
			if p.isBackpressure(outputChan, p.inputBufferSize) {
				logger.Debug("Backpressure detected in input reader, slowing down")
				time.Sleep(100 * time.Millisecond)
				continue
			}
			
			// Read batch
			batch, err := p.input.ReadBatch(p.batchSize)
			if err == io.EOF {
				logger.Info("Input reader reached end of input")
				return
			}
			if err != nil {
				p.errorChan <- fmt.Errorf("failed to read batch: %w", err)
				return
			}
			
			if batch == nil || batch.IsEmpty() {
				logger.Info("Input reader received empty batch, stopping")
				return
			}
			
			batchCount++
			logger.Debug("Input reader produced batch %d with %d records", batchCount, batch.Size())
			
			// Save checkpoint if enabled
			if p.checkpointManager != nil && batch.Checkpoint != nil {
				if err := p.checkpointManager.SaveCheckpoint("input", batch.Checkpoint); err != nil {
					logger.Warn("Failed to save input checkpoint: %v", err)
				}
			}
			
			// Send to channel
			select {
			case outputChan <- batch:
				p.incrementBatches()
			case <-ctx.Done():
				logger.Info("Input reader cancelled while sending batch")
				return
			}
		}
	}
}

// runProcessor processes batches from input channel and sends to output channel
func (p *ConcurrentPipeline) runProcessor(
	ctx context.Context,
	processor types.ProcessorPlugin,
	inputChan <-chan *types.DataBatch,
	outputChan chan<- *types.DataBatch,
	index int,
	wg *sync.WaitGroup,
) {
	defer wg.Done()
	defer close(outputChan)
	
	logger.Info("Processor %d (%s) started", index, processor.Name())
	processedCount := 0
	
	for {
		select {
		case <-ctx.Done():
			logger.Info("Processor %d cancelled", index)
			return
		case batch, ok := <-inputChan:
			if !ok {
				logger.Info("Processor %d input channel closed", index)
				return
			}
			
			// Check backpressure
			if p.isBackpressure(outputChan, p.processorBufferSize) {
				logger.Debug("Backpressure detected in processor %d, slowing down", index)
				time.Sleep(50 * time.Millisecond)
			}
			
			// Process batch
			processed, err := processor.Process(batch)
			if err != nil {
				p.errorChan <- fmt.Errorf("processor %d (%s) failed: %w", index, processor.Name(), err)
				return
			}
			
			processedCount++
			
			if processed.IsEmpty() {
				logger.Debug("Processor %d filtered out all records in batch %d", index, processedCount)
				continue
			}
			
			logger.Debug("Processor %d processed batch %d: %d records", index, processedCount, processed.Size())
			
			// Send to next stage
			select {
			case outputChan <- processed:
				// Success
			case <-ctx.Done():
				logger.Info("Processor %d cancelled while sending batch", index)
				return
			}
		}
	}
}

// runOutputWriter writes batches from channel to output
func (p *ConcurrentPipeline) runOutputWriter(ctx context.Context, inputChan <-chan *types.DataBatch, wg *sync.WaitGroup) {
	defer wg.Done()
	
	logger.Info("Output writer started")
	batchCount := 0
	
	for {
		select {
		case <-ctx.Done():
			logger.Info("Output writer cancelled")
			return
		case batch, ok := <-inputChan:
			if !ok {
				logger.Info("Output writer input channel closed")
				return
			}
			
			// Write batch
			if err := p.output.WriteBatch(batch); err != nil {
				p.errorChan <- fmt.Errorf("failed to write batch: %w", err)
				return
			}
			
			batchCount++
			p.incrementRecords(int64(batch.Size()))
			logger.Debug("Output writer wrote batch %d with %d records", batchCount, batch.Size())
			
			// Save checkpoint if enabled
			if p.checkpointManager != nil && batch.Checkpoint != nil {
				if err := p.checkpointManager.SaveCheckpoint("output", batch.Checkpoint); err != nil {
					logger.Warn("Failed to save output checkpoint: %v", err)
				}
			}
		}
	}
}

// isBackpressure checks if backpressure should be applied
func (p *ConcurrentPipeline) isBackpressure(ch interface{}, capacity int) bool {
	// Use reflection to get channel length
	var length int
	switch c := ch.(type) {
	case chan *types.DataBatch:
		length = len(c)
	case chan<- *types.DataBatch:
		// Can't check send-only channel length directly
		return false
	default:
		return false
	}
	
	ratio := float64(length) / float64(capacity)
	return ratio >= p.backpressureThreshold
}

// incrementBatches increments batch counter
func (p *ConcurrentPipeline) incrementBatches() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.totalBatches++
}

// incrementRecords increments record counter
func (p *ConcurrentPipeline) incrementRecords(count int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.totalRecords += count
}

// GetStatistics returns pipeline statistics
func (p *ConcurrentPipeline) GetStatistics() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	stats := make(map[string]interface{})
	
	// Pipeline stats
	stats["total_batches"] = p.totalBatches
	stats["total_records"] = p.totalRecords
	stats["failed_records"] = p.failedRecords
	
	// Timing
	if !p.startTime.IsZero() {
		stats["start_time"] = p.startTime
		if !p.endTime.IsZero() {
			duration := p.endTime.Sub(p.startTime)
			stats["duration_seconds"] = duration.Seconds()
			stats["throughput_records_per_second"] = float64(p.totalRecords) / duration.Seconds()
		}
	}
	
	// Input progress
	stats["input_progress"] = p.input.GetProgress()
	
	// Processor statistics
	processorStats := make([]interface{}, len(p.processors))
	for i, proc := range p.processors {
		processorStats[i] = proc.GetStatistics()
	}
	stats["processors"] = processorStats
	
	// Output statistics
	stats["output"] = p.output.GetWriteStatistics()
	
	return stats
}

// logStatistics logs pipeline statistics
func (p *ConcurrentPipeline) logStatistics() {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	duration := p.endTime.Sub(p.startTime)
	throughput := float64(p.totalRecords) / duration.Seconds()
	
	logger.Info("Pipeline Statistics:")
	logger.Info("  Total Batches: %d", p.totalBatches)
	logger.Info("  Total Records: %d", p.totalRecords)
	logger.Info("  Failed Records: %d", p.failedRecords)
	logger.Info("  Duration: %.2f seconds", duration.Seconds())
	logger.Info("  Throughput: %.2f records/second", throughput)
}

// GetCheckpointManager returns the checkpoint manager
func (p *ConcurrentPipeline) GetCheckpointManager() *checkpoint.Manager {
	return p.checkpointManager
}

// LoadCheckpoint loads a checkpoint for a specific stage
func (p *ConcurrentPipeline) LoadCheckpoint(stage string) (*types.Checkpoint, error) {
	if p.checkpointManager == nil {
		return nil, nil
	}
	return p.checkpointManager.LoadCheckpoint(stage)
}

// ClearCheckpoints clears all checkpoints
func (p *ConcurrentPipeline) ClearCheckpoints() error {
	if p.checkpointManager == nil {
		return nil
	}
	return p.checkpointManager.Clear()
}
