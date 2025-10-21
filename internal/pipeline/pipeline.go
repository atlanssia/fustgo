package pipeline

import (
	"fmt"
	"io"

	"github.com/atlanssia/fustgo/internal/logger"
	"github.com/atlanssia/fustgo/pkg/types"
)

// Pipeline represents a data processing pipeline
type Pipeline struct {
	input      types.InputPlugin
	processors []types.ProcessorPlugin
	output     types.OutputPlugin
	batchSize  int
}

// NewPipeline creates a new pipeline
func NewPipeline(input types.InputPlugin, processors []types.ProcessorPlugin, output types.OutputPlugin) *Pipeline {
	return &Pipeline{
		input:      input,
		processors: processors,
		output:     output,
		batchSize:  1000, // Default batch size
	}
}

// SetBatchSize sets the batch size for reading
func (p *Pipeline) SetBatchSize(size int) {
	p.batchSize = size
}

// Execute executes the pipeline
func (p *Pipeline) Execute() error {
	logger.Info("Starting pipeline execution")
	
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
	
	totalRecords := int64(0)
	batchCount := 0
	
	// Process data in batches
	for {
		// Read batch from input
		batch, err := p.input.ReadBatch(p.batchSize)
		if err == io.EOF {
			logger.Info("Reached end of input")
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read batch: %w", err)
		}
		
		if batch == nil || batch.IsEmpty() {
			break
		}
		
		batchCount++
		logger.Info("Processing batch %d with %d records", batchCount, batch.Size())
		
		// Process through all processors
		processedBatch := batch
		for i, processor := range p.processors {
			processedBatch, err = processor.Process(processedBatch)
			if err != nil {
				return fmt.Errorf("processor %d failed: %w", i, err)
			}
			
			if processedBatch.IsEmpty() {
				logger.Info("All records filtered out by processor %d", i)
				break
			}
		}
		
		// Write to output
		if !processedBatch.IsEmpty() {
			if err := p.output.WriteBatch(processedBatch); err != nil {
				return fmt.Errorf("failed to write batch: %w", err)
			}
			totalRecords += int64(processedBatch.Size())
		}
	}
	
	// Flush output
	if err := p.output.Flush(); err != nil {
		return fmt.Errorf("failed to flush output: %w", err)
	}
	
	logger.Info("Pipeline execution completed: %d batches, %d total records", batchCount, totalRecords)
	
	return nil
}

// GetStatistics returns pipeline statistics
func (p *Pipeline) GetStatistics() map[string]interface{} {
	stats := make(map[string]interface{})
	
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
