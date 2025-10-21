package pipeline

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/atlanssia/fustgo/pkg/types"
)

// Mock plugins for testing

type mockInputPlugin struct {
	batches []*types.DataBatch
	index   int
	delay   time.Duration
}

func (m *mockInputPlugin) Name() string                              { return "mock-input" }
func (m *mockInputPlugin) Type() types.PluginType                    { return types.PluginTypeInput }
func (m *mockInputPlugin) Initialize(config map[string]interface{}) error { return nil }
func (m *mockInputPlugin) Validate() error                           { return nil }
func (m *mockInputPlugin) Close() error                              { return nil }
func (m *mockInputPlugin) GetMetadata() types.PluginMetadata {
	return types.PluginMetadata{Name: "mock-input", Version: "1.0.0"}
}
func (m *mockInputPlugin) Connect() error { return nil }

func (m *mockInputPlugin) ReadBatch(batchSize int) (*types.DataBatch, error) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	
	if m.index >= len(m.batches) {
		return nil, io.EOF
	}
	
	batch := m.batches[m.index]
	m.index++
	return batch, nil
}

func (m *mockInputPlugin) HasNext() bool {
	return m.index < len(m.batches)
}

func (m *mockInputPlugin) GetProgress() *types.Progress {
	return &types.Progress{
		TotalRecords:     int64(len(m.batches) * 100),
		ProcessedRecords: int64(m.index * 100),
	}
}

type mockProcessorPlugin struct {
	name      string
	filterAll bool
	delay     time.Duration
}

func (m *mockProcessorPlugin) Name() string                              { return m.name }
func (m *mockProcessorPlugin) Type() types.PluginType                    { return types.PluginTypeProcessor }
func (m *mockProcessorPlugin) Initialize(config map[string]interface{}) error { return nil }
func (m *mockProcessorPlugin) Validate() error                           { return nil }
func (m *mockProcessorPlugin) Close() error                              { return nil }
func (m *mockProcessorPlugin) GetMetadata() types.PluginMetadata {
	return types.PluginMetadata{Name: m.name, Version: "1.0.0"}
}

func (m *mockProcessorPlugin) Process(input *types.DataBatch) (*types.DataBatch, error) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	
	if m.filterAll {
		return &types.DataBatch{
			Schema:  input.Schema,
			Records: []types.Record{},
		}, nil
	}
	
	return input, nil
}

func (m *mockProcessorPlugin) GetStatistics() *types.ProcessStatistics {
	return &types.ProcessStatistics{
		RecordsIn:  100,
		RecordsOut: 100,
		Filtered:   0,
	}
}

type mockOutputPlugin struct {
	batches []*types.DataBatch
	delay   time.Duration
}

func (m *mockOutputPlugin) Name() string                              { return "mock-output" }
func (m *mockOutputPlugin) Type() types.PluginType                    { return types.PluginTypeOutput }
func (m *mockOutputPlugin) Initialize(config map[string]interface{}) error { return nil }
func (m *mockOutputPlugin) Validate() error                           { return nil }
func (m *mockOutputPlugin) Close() error                              { return nil }
func (m *mockOutputPlugin) GetMetadata() types.PluginMetadata {
	return types.PluginMetadata{Name: "mock-output", Version: "1.0.0"}
}
func (m *mockOutputPlugin) Connect() error { return nil }

func (m *mockOutputPlugin) WriteBatch(data *types.DataBatch) error {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	m.batches = append(m.batches, data)
	return nil
}

func (m *mockOutputPlugin) Flush() error { return nil }

func (m *mockOutputPlugin) GetWriteStatistics() *types.WriteStatistics {
	totalRecords := int64(0)
	for _, batch := range m.batches {
		totalRecords += int64(batch.Size())
	}
	return &types.WriteStatistics{
		RecordsWritten: totalRecords,
		RecordsFailed:  0,
	}
}

// Helper function to create test batches
func createTestBatch(numRecords int) *types.DataBatch {
	schema := types.Schema{
		Columns: []types.Column{
			{Name: "id", DataType: types.DataTypeInt},
			{Name: "name", DataType: types.DataTypeString},
		},
	}
	
	records := make([]types.Record, numRecords)
	for i := 0; i < numRecords; i++ {
		records[i] = types.Record{
			Values: []interface{}{i, "test"},
		}
	}
	
	return &types.DataBatch{
		Schema:  schema,
		Records: records,
	}
}

func TestNewConcurrentPipeline(t *testing.T) {
	input := &mockInputPlugin{}
	processors := []types.ProcessorPlugin{&mockProcessorPlugin{name: "proc1"}}
	output := &mockOutputPlugin{}
	
	pipeline := NewConcurrentPipeline(input, processors, output, nil)
	
	assert.NotNil(t, pipeline)
	assert.Equal(t, 1000, pipeline.batchSize)
	assert.Equal(t, 10, pipeline.inputBufferSize)
	assert.Equal(t, 10, pipeline.processorBufferSize)
	assert.Equal(t, 5, pipeline.outputBufferSize)
	assert.Equal(t, 0.8, pipeline.backpressureThreshold)
}

func TestNewConcurrentPipelineWithCustomConfig(t *testing.T) {
	input := &mockInputPlugin{}
	processors := []types.ProcessorPlugin{&mockProcessorPlugin{name: "proc1"}}
	output := &mockOutputPlugin{}
	
	config := &ConcurrentPipelineConfig{
		BatchSize:             500,
		InputBufferSize:       5,
		ProcessorBufferSize:   5,
		OutputBufferSize:      3,
		BackpressureThreshold: 0.7,
	}
	
	pipeline := NewConcurrentPipeline(input, processors, output, config)
	
	assert.NotNil(t, pipeline)
	assert.Equal(t, 500, pipeline.batchSize)
	assert.Equal(t, 5, pipeline.inputBufferSize)
	assert.Equal(t, 0.7, pipeline.backpressureThreshold)
}

func TestConcurrentPipelineExecute(t *testing.T) {
	// Create test batches
	batches := []*types.DataBatch{
		createTestBatch(10),
		createTestBatch(20),
		createTestBatch(15),
	}
	
	input := &mockInputPlugin{batches: batches}
	processors := []types.ProcessorPlugin{
		&mockProcessorPlugin{name: "proc1"},
		&mockProcessorPlugin{name: "proc2"},
	}
	output := &mockOutputPlugin{}
	
	pipeline := NewConcurrentPipeline(input, processors, output, nil)
	
	ctx := context.Background()
	err := pipeline.Execute(ctx)
	
	assert.NoError(t, err)
	assert.Equal(t, 3, len(output.batches))
	
	// Verify statistics
	stats := pipeline.GetStatistics()
	assert.Equal(t, int64(3), stats["total_batches"])
	assert.Equal(t, int64(45), stats["total_records"]) // 10 + 20 + 15
}

func TestConcurrentPipelineExecuteWithFilter(t *testing.T) {
	// Create test batches
	batches := []*types.DataBatch{
		createTestBatch(10),
		createTestBatch(20),
	}
	
	input := &mockInputPlugin{batches: batches}
	processors := []types.ProcessorPlugin{
		&mockProcessorPlugin{name: "filter", filterAll: true}, // Filters everything
	}
	output := &mockOutputPlugin{}
	
	pipeline := NewConcurrentPipeline(input, processors, output, nil)
	
	ctx := context.Background()
	err := pipeline.Execute(ctx)
	
	assert.NoError(t, err)
	assert.Equal(t, 0, len(output.batches)) // All filtered out
}

func TestConcurrentPipelineExecuteWithTimeout(t *testing.T) {
	// Create test batches with delay
	batches := []*types.DataBatch{
		createTestBatch(10),
		createTestBatch(20),
	}
	
	input := &mockInputPlugin{batches: batches, delay: 500 * time.Millisecond}
	processors := []types.ProcessorPlugin{}
	output := &mockOutputPlugin{}
	
	pipeline := NewConcurrentPipeline(input, processors, output, nil)
	
	// Set timeout that should trigger
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	
	err := pipeline.Execute(ctx)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cancelled")
}

func TestConcurrentPipelineGetStatistics(t *testing.T) {
	batches := []*types.DataBatch{
		createTestBatch(10),
	}
	
	input := &mockInputPlugin{batches: batches}
	processors := []types.ProcessorPlugin{
		&mockProcessorPlugin{name: "proc1"},
	}
	output := &mockOutputPlugin{}
	
	pipeline := NewConcurrentPipeline(input, processors, output, nil)
	
	ctx := context.Background()
	err := pipeline.Execute(ctx)
	assert.NoError(t, err)
	
	stats := pipeline.GetStatistics()
	
	assert.NotNil(t, stats["total_batches"])
	assert.NotNil(t, stats["total_records"])
	assert.NotNil(t, stats["input_progress"])
	assert.NotNil(t, stats["processors"])
	assert.NotNil(t, stats["output"])
	assert.NotNil(t, stats["duration_seconds"])
	assert.NotNil(t, stats["throughput_records_per_second"])
}

func TestDefaultConcurrentConfig(t *testing.T) {
	config := DefaultConcurrentConfig()
	
	assert.NotNil(t, config)
	assert.Equal(t, 1000, config.BatchSize)
	assert.Equal(t, 10, config.InputBufferSize)
	assert.Equal(t, 10, config.ProcessorBufferSize)
	assert.Equal(t, 5, config.OutputBufferSize)
	assert.Equal(t, 0.8, config.BackpressureThreshold)
}

func TestConcurrentPipelineBackpressure(t *testing.T) {
	// Create many batches to trigger backpressure
	batches := make([]*types.DataBatch, 20)
	for i := 0; i < 20; i++ {
		batches[i] = createTestBatch(10)
	}
	
	input := &mockInputPlugin{batches: batches}
	processors := []types.ProcessorPlugin{
		&mockProcessorPlugin{name: "slow-proc", delay: 10 * time.Millisecond},
	}
	output := &mockOutputPlugin{delay: 10 * time.Millisecond}
	
	config := &ConcurrentPipelineConfig{
		BatchSize:             100,
		InputBufferSize:       3, // Small buffer to trigger backpressure
		ProcessorBufferSize:   3,
		OutputBufferSize:      2,
		BackpressureThreshold: 0.5,
	}
	
	pipeline := NewConcurrentPipeline(input, processors, output, config)
	
	ctx := context.Background()
	err := pipeline.Execute(ctx)
	
	assert.NoError(t, err)
	assert.Equal(t, 20, len(output.batches))
	
	stats := pipeline.GetStatistics()
	assert.Equal(t, int64(20), stats["total_batches"])
	assert.Equal(t, int64(200), stats["total_records"])
}
