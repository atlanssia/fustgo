package config

import (
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/atlanssia/fustgo/internal/pipeline"
	"github.com/atlanssia/fustgo/internal/plugin"
	"github.com/atlanssia/fustgo/pkg/types"
)

// PipelineConfig represents a YAML pipeline configuration
type PipelineConfig struct {
	Input      InputConfig       `yaml:"input"`
	Processors []ProcessorConfig `yaml:"processors,omitempty"`
	Output     OutputConfig      `yaml:"output"`
	Settings   SettingsConfig    `yaml:"settings,omitempty"`
}

// InputConfig represents input configuration
type InputConfig struct {
	Type   string                 `yaml:"type"`
	Config map[string]interface{} `yaml:"config,omitempty"`
}

// ProcessorConfig represents processor configuration
type ProcessorConfig struct {
	Type   string                 `yaml:"type"`
	Config map[string]interface{} `yaml:"config,omitempty"`
}

// OutputConfig represents output configuration
type OutputConfig struct {
	Type   string                 `yaml:"type"`
	Config map[string]interface{} `yaml:"config,omitempty"`
}

// SettingsConfig represents pipeline settings
type SettingsConfig struct {
	BatchSize int    `yaml:"batch_size,omitempty"`
	Mode      string `yaml:"mode,omitempty"` // "sync" or "async"
}

// Converter converts YAML configuration to pipeline
type Converter struct {
	registry *plugin.Registry
}

// NewConverter creates a new configuration converter
func NewConverter(registry *plugin.Registry) *Converter {
	return &Converter{
		registry: registry,
	}
}

// ParseYAML parses YAML configuration string
func (c *Converter) ParseYAML(yamlConfig string) (*PipelineConfig, error) {
	var config PipelineConfig
	if err := yaml.Unmarshal([]byte(yamlConfig), &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Validate configuration
	if err := c.ValidateConfig(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// ValidateConfig validates pipeline configuration
func (c *Converter) ValidateConfig(config *PipelineConfig) error {
	// Validate input
	if config.Input.Type == "" {
		return fmt.Errorf("input type is required")
	}

	// Validate output
	if config.Output.Type == "" {
		return fmt.Errorf("output type is required")
	}

	return nil
}

// BuildPipeline builds a pipeline from configuration
func (c *Converter) BuildPipeline(config *PipelineConfig) (*pipeline.Pipeline, error) {
	// Get input plugin
	input, err := c.registry.GetInput(config.Input.Type)
	if err != nil {
		return nil, fmt.Errorf("failed to get input plugin '%s': %w", config.Input.Type, err)
	}

	// Initialize input plugin
	if err := input.Initialize(config.Input.Config); err != nil {
		return nil, fmt.Errorf("failed to initialize input plugin: %w", err)
	}

	// Get processor plugins
	processors := make([]types.ProcessorPlugin, 0, len(config.Processors))
	for i, procConfig := range config.Processors {
		processor, err := c.registry.GetProcessor(procConfig.Type)
		if err != nil {
			return nil, fmt.Errorf("failed to get processor plugin '%s': %w", procConfig.Type, err)
		}

		if err := processor.Initialize(procConfig.Config); err != nil {
			return nil, fmt.Errorf("failed to initialize processor %d: %w", i, err)
		}

		processors = append(processors, processor)
	}

	// Get output plugin
	output, err := c.registry.GetOutput(config.Output.Type)
	if err != nil {
		return nil, fmt.Errorf("failed to get output plugin '%s': %w", config.Output.Type, err)
	}

	// Initialize output plugin
	if err := output.Initialize(config.Output.Config); err != nil {
		return nil, fmt.Errorf("failed to initialize output plugin: %w", err)
	}

	// Create pipeline
	p := pipeline.NewPipeline(input, processors, output)

	// Apply settings
	if config.Settings.BatchSize > 0 {
		p.SetBatchSize(config.Settings.BatchSize)
	}

	return p, nil
}

// BuildConcurrentPipeline builds a concurrent pipeline from configuration
func (c *Converter) BuildConcurrentPipeline(config *PipelineConfig) (*pipeline.ConcurrentPipeline, error) {
	// Get input plugin
	input, err := c.registry.GetInput(config.Input.Type)
	if err != nil {
		return nil, fmt.Errorf("failed to get input plugin '%s': %w", config.Input.Type, err)
	}

	if err := input.Initialize(config.Input.Config); err != nil {
		return nil, fmt.Errorf("failed to initialize input plugin: %w", err)
	}

	// Get processor plugins
	processors := make([]types.ProcessorPlugin, 0, len(config.Processors))
	for i, procConfig := range config.Processors {
		processor, err := c.registry.GetProcessor(procConfig.Type)
		if err != nil {
			return nil, fmt.Errorf("failed to get processor plugin '%s': %w", procConfig.Type, err)
		}

		if err := processor.Initialize(procConfig.Config); err != nil {
			return nil, fmt.Errorf("failed to initialize processor %d: %w", i, err)
		}

		processors = append(processors, processor)
	}

	// Get output plugin
	output, err := c.registry.GetOutput(config.Output.Type)
	if err != nil {
		return nil, fmt.Errorf("failed to get output plugin '%s': %w", config.Output.Type, err)
	}

	if err := output.Initialize(config.Output.Config); err != nil {
		return nil, fmt.Errorf("failed to initialize output plugin: %w", err)
	}

	// Create concurrent pipeline configuration
	pipelineConfig := pipeline.DefaultConcurrentConfig()
	if config.Settings.BatchSize > 0 {
		pipelineConfig.BatchSize = config.Settings.BatchSize
	}

	// Create pipeline
	p := pipeline.NewConcurrentPipeline(input, processors, output, pipelineConfig)

	return p, nil
}

// ConfigToYAML converts pipeline config back to YAML
func (c *Converter) ConfigToYAML(config *PipelineConfig) (string, error) {
	data, err := yaml.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("failed to marshal config to YAML: %w", err)
	}
	return string(data), nil
}

// ExampleConfig returns an example YAML configuration
func ExampleConfig() string {
	return `input:
  type: csv
  config:
    path: /data/input.csv
    delimiter: ","
    has_header: true

processors:
  - type: filter
    config:
      condition: "age > 18"
      mode: include
  
  - type: mapping
    config:
      mappings:
        old_name: new_name
        user_id: id

output:
  type: csv
  config:
    path: /data/output.csv
    delimiter: ","
    write_header: true

settings:
  batch_size: 1000
  mode: async
`
}
