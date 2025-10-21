package types

// PluginType represents the type of plugin
type PluginType string

const (
	PluginTypeInput     PluginType = "input"
	PluginTypeProcessor PluginType = "processor"
	PluginTypeOutput    PluginType = "output"
)

// PluginMetadata contains metadata about a plugin
type PluginMetadata struct {
	Name             string                 `json:"name"`
	Type             PluginType             `json:"type"`
	Version          string                 `json:"version"`
	Description      string                 `json:"description"`
	DataSourceType   string                 `json:"data_source_type,omitempty"`
	ConfigSchema     map[string]interface{} `json:"config_schema"`
	SupportedFormats []string               `json:"supported_formats,omitempty"`
}

// Plugin is the base interface that all plugins must implement
type Plugin interface {
	// Name returns the plugin's unique name
	Name() string

	// Type returns the plugin type (Input, Processor, Output)
	Type() PluginType

	// Initialize initializes the plugin with configuration
	Initialize(config map[string]interface{}) error

	// Validate validates the plugin's configuration
	Validate() error

	// Close closes the plugin and releases resources
	Close() error

	// GetMetadata returns plugin metadata
	GetMetadata() PluginMetadata
}

// InputPlugin defines the interface for input/source plugins
type InputPlugin interface {
	Plugin

	// Connect establishes connection to the data source
	Connect() error

	// ReadBatch reads a batch of data
	ReadBatch(batchSize int) (*DataBatch, error)

	// HasNext checks if there is more data to read
	HasNext() bool

	// GetProgress returns the current reading progress
	GetProgress() *Progress
}

// ProcessorPlugin defines the interface for data processing plugins
type ProcessorPlugin interface {
	Plugin

	// Process processes a batch of data
	Process(input *DataBatch) (*DataBatch, error)

	// GetStatistics returns processing statistics
	GetStatistics() *ProcessStatistics
}

// OutputPlugin defines the interface for output/sink plugins
type OutputPlugin interface {
	Plugin

	// Connect establishes connection to the target system
	Connect() error

	// WriteBatch writes a batch of data
	WriteBatch(data *DataBatch) error

	// Flush flushes any buffered data
	Flush() error

	// GetWriteStatistics returns write statistics
	GetWriteStatistics() *WriteStatistics
}
