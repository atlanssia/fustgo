package plugin

import (
	"testing"

	"github.com/atlanssia/fustgo/pkg/types"
	"github.com/stretchr/testify/assert"
)

// Mock plugins for testing
type mockInputPlugin struct{}

func (m *mockInputPlugin) Name() string                                  { return "mock-input" }
func (m *mockInputPlugin) Type() types.PluginType                        { return types.PluginTypeInput }
func (m *mockInputPlugin) Initialize(config map[string]interface{}) error { return nil }
func (m *mockInputPlugin) Validate() error                               { return nil }
func (m *mockInputPlugin) Close() error                                  { return nil }
func (m *mockInputPlugin) GetMetadata() types.PluginMetadata {
	return types.PluginMetadata{Name: "mock-input", Type: types.PluginTypeInput}
}
func (m *mockInputPlugin) Connect() error                                  { return nil }
func (m *mockInputPlugin) ReadBatch(batchSize int) (*types.DataBatch, error) { return nil, nil }
func (m *mockInputPlugin) HasNext() bool                                   { return false }
func (m *mockInputPlugin) GetProgress() *types.Progress                    { return nil }

type mockProcessorPlugin struct{}

func (m *mockProcessorPlugin) Name() string                                  { return "mock-processor" }
func (m *mockProcessorPlugin) Type() types.PluginType                        { return types.PluginTypeProcessor }
func (m *mockProcessorPlugin) Initialize(config map[string]interface{}) error { return nil }
func (m *mockProcessorPlugin) Validate() error                               { return nil }
func (m *mockProcessorPlugin) Close() error                                  { return nil }
func (m *mockProcessorPlugin) GetMetadata() types.PluginMetadata {
	return types.PluginMetadata{Name: "mock-processor", Type: types.PluginTypeProcessor}
}
func (m *mockProcessorPlugin) Process(input *types.DataBatch) (*types.DataBatch, error) { return input, nil }
func (m *mockProcessorPlugin) GetStatistics() *types.ProcessStatistics                   { return nil }

type mockOutputPlugin struct{}

func (m *mockOutputPlugin) Name() string                                  { return "mock-output" }
func (m *mockOutputPlugin) Type() types.PluginType                        { return types.PluginTypeOutput }
func (m *mockOutputPlugin) Initialize(config map[string]interface{}) error { return nil }
func (m *mockOutputPlugin) Validate() error                               { return nil }
func (m *mockOutputPlugin) Close() error                                  { return nil }
func (m *mockOutputPlugin) GetMetadata() types.PluginMetadata {
	return types.PluginMetadata{Name: "mock-output", Type: types.PluginTypeOutput}
}
func (m *mockOutputPlugin) Connect() error                                 { return nil }
func (m *mockOutputPlugin) WriteBatch(data *types.DataBatch) error         { return nil }
func (m *mockOutputPlugin) Flush() error                                   { return nil }
func (m *mockOutputPlugin) GetWriteStatistics() *types.WriteStatistics     { return nil }

func TestRegistry_RegisterInput(t *testing.T) {
	registry := &Registry{
		inputs:     make(map[string]types.InputPlugin),
		processors: make(map[string]types.ProcessorPlugin),
		outputs:    make(map[string]types.OutputPlugin),
	}

	plugin := &mockInputPlugin{}
	err := registry.RegisterInput("test-input", plugin)
	assert.NoError(t, err)

	// Try to register again - should error
	err = registry.RegisterInput("test-input", plugin)
	assert.Error(t, err)
}

func TestRegistry_RegisterProcessor(t *testing.T) {
	registry := &Registry{
		inputs:     make(map[string]types.InputPlugin),
		processors: make(map[string]types.ProcessorPlugin),
		outputs:    make(map[string]types.OutputPlugin),
	}

	plugin := &mockProcessorPlugin{}
	err := registry.RegisterProcessor("test-processor", plugin)
	assert.NoError(t, err)

	// Try to register again - should error
	err = registry.RegisterProcessor("test-processor", plugin)
	assert.Error(t, err)
}

func TestRegistry_RegisterOutput(t *testing.T) {
	registry := &Registry{
		inputs:     make(map[string]types.InputPlugin),
		processors: make(map[string]types.ProcessorPlugin),
		outputs:    make(map[string]types.OutputPlugin),
	}

	plugin := &mockOutputPlugin{}
	err := registry.RegisterOutput("test-output", plugin)
	assert.NoError(t, err)

	// Try to register again - should error
	err = registry.RegisterOutput("test-output", plugin)
	assert.Error(t, err)
}

func TestRegistry_GetInput(t *testing.T) {
	registry := &Registry{
		inputs:     make(map[string]types.InputPlugin),
		processors: make(map[string]types.ProcessorPlugin),
		outputs:    make(map[string]types.OutputPlugin),
	}

	plugin := &mockInputPlugin{}
	registry.RegisterInput("test-input", plugin)

	// Test successful retrieval
	retrieved, err := registry.GetInput("test-input")
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)

	// Test not found
	_, err = registry.GetInput("non-existent")
	assert.Error(t, err)
}

func TestRegistry_ListInputs(t *testing.T) {
	registry := &Registry{
		inputs:     make(map[string]types.InputPlugin),
		processors: make(map[string]types.ProcessorPlugin),
		outputs:    make(map[string]types.OutputPlugin),
	}

	registry.RegisterInput("input1", &mockInputPlugin{})
	registry.RegisterInput("input2", &mockInputPlugin{})

	names := registry.ListInputs()
	assert.Len(t, names, 2)
	assert.Contains(t, names, "input1")
	assert.Contains(t, names, "input2")
}

func TestRegistry_ListProcessors(t *testing.T) {
	registry := &Registry{
		inputs:     make(map[string]types.InputPlugin),
		processors: make(map[string]types.ProcessorPlugin),
		outputs:    make(map[string]types.OutputPlugin),
	}

	registry.RegisterProcessor("proc1", &mockProcessorPlugin{})
	registry.RegisterProcessor("proc2", &mockProcessorPlugin{})

	names := registry.ListProcessors()
	assert.Len(t, names, 2)
	assert.Contains(t, names, "proc1")
	assert.Contains(t, names, "proc2")
}

func TestRegistry_ListOutputs(t *testing.T) {
	registry := &Registry{
		inputs:     make(map[string]types.InputPlugin),
		processors: make(map[string]types.ProcessorPlugin),
		outputs:    make(map[string]types.OutputPlugin),
	}

	registry.RegisterOutput("output1", &mockOutputPlugin{})
	registry.RegisterOutput("output2", &mockOutputPlugin{})

	names := registry.ListOutputs()
	assert.Len(t, names, 2)
	assert.Contains(t, names, "output1")
	assert.Contains(t, names, "output2")
}
