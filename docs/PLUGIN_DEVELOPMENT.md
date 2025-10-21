# Plugin Development Guide

## Overview

FustGo DataX uses a static compilation plugin architecture where all plugins are compiled into the main binary. This guide explains how to develop custom plugins for input, processing, and output operations.

## Plugin Types

### 1. Input Plugins
Read data from external sources (databases, files, APIs, message queues)

### 2. Processor Plugins
Transform, filter, enrich, or aggregate data in transit

### 3. Output Plugins
Write data to target destinations

## Development Steps

### Step 1: Create Plugin Directory

```bash
# For input plugin
mkdir -p plugins/input/myplugin

# For processor plugin
mkdir -p plugins/processor/myplugin

# For output plugin
mkdir -p plugins/output/myplugin
```

### Step 2: Implement the Plugin Interface

All plugins must implement the base `Plugin` interface plus their specific interface.

#### Base Plugin Interface

```go
type Plugin interface {
    Name() string
    Type() PluginType
    Initialize(config map[string]interface{}) error
    Validate() error
    Close() error
    GetMetadata() PluginMetadata
}
```

#### Input Plugin Interface

```go
type InputPlugin interface {
    Plugin
    Connect() error
    ReadBatch(batchSize int) (*DataBatch, error)
    HasNext() bool
    GetProgress() *Progress
}
```

#### Processor Plugin Interface

```go
type ProcessorPlugin interface {
    Plugin
    Process(input *DataBatch) (*DataBatch, error)
    GetStatistics() *ProcessStatistics
}
```

#### Output Plugin Interface

```go
type OutputPlugin interface {
    Plugin
    Connect() error
    WriteBatch(data *DataBatch) error
    Flush() error
    GetWriteStatistics() *WriteStatistics
}
```

## Example: Creating a JSON Input Plugin

### File: `plugins/input/json/json.go`

```go
package json

import (
    "encoding/json"
    "fmt"
    "io"
    "os"

    "github.com/atlanssia/fustgo/pkg/types"
)

type JSONInputPlugin struct {
    config   map[string]interface{}
    file     *os.File
    decoder  *json.Decoder
    progress *types.Progress
}

func (p *JSONInputPlugin) Name() string {
    return "json"
}

func (p *JSONInputPlugin) Type() types.PluginType {
    return types.PluginTypeInput
}

func (p *JSONInputPlugin) Initialize(config map[string]interface{}) error {
    p.config = config
    p.progress = &types.Progress{
        ProcessedRecords: 0,
    }
    return nil
}

func (p *JSONInputPlugin) Validate() error {
    if p.config["path"] == nil {
        return fmt.Errorf("json input: path is required")
    }
    return nil
}

func (p *JSONInputPlugin) Connect() error {
    path := p.config["path"].(string)
    
    file, err := os.Open(path)
    if err != nil {
        return fmt.Errorf("failed to open file: %w", err)
    }
    
    p.file = file
    p.decoder = json.NewDecoder(file)
    
    return nil
}

func (p *JSONInputPlugin) ReadBatch(batchSize int) (*types.DataBatch, error) {
    var records []types.Record
    
    for i := 0; i < batchSize; i++ {
        var data map[string]interface{}
        if err := p.decoder.Decode(&data); err == io.EOF {
            break
        } else if err != nil {
            return nil, err
        }
        
        // Convert map to record
        // Implementation details...
        
        p.progress.ProcessedRecords++
    }
    
    // Build and return DataBatch
    // ...
    
    return batch, nil
}

func (p *JSONInputPlugin) HasNext() bool {
    return true
}

func (p *JSONInputPlugin) GetProgress() *types.Progress {
    return p.progress
}

func (p *JSONInputPlugin) Close() error {
    if p.file != nil {
        return p.file.Close()
    }
    return nil
}

func (p *JSONInputPlugin) GetMetadata() types.PluginMetadata {
    return types.PluginMetadata{
        Name:        "json",
        Type:        types.PluginTypeInput,
        Version:     "1.0.0",
        Description: "JSON file input plugin",
        ConfigSchema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "path": map[string]interface{}{
                    "type": "string",
                    "description": "Path to JSON file",
                },
            },
            "required": []string{"path"},
        },
    }
}
```

### Step 3: Register the Plugin

Create an `init.go` file:

```go
package json

import (
    "github.com/atlanssia/fustgo/internal/plugin"
)

func init() {
    plugin.RegisterInput("json", &JSONInputPlugin{})
}
```

### Step 4: Add to Plugin Loader

Edit `plugins/loader.go`:

```go
import (
    _ "github.com/atlanssia/fustgo/plugins/input/json"
)
```

### Step 5: Write Tests

Create `plugins/input/json/json_test.go`:

```go
package json

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestJSONInputPlugin_Initialize(t *testing.T) {
    plugin := &JSONInputPlugin{}
    
    config := map[string]interface{}{
        "path": "/tmp/test.json",
    }
    
    err := plugin.Initialize(config)
    assert.NoError(t, err)
    assert.Equal(t, "/tmp/test.json", plugin.config["path"])
}

func TestJSONInputPlugin_Validate(t *testing.T) {
    tests := []struct {
        name    string
        config  map[string]interface{}
        wantErr bool
    }{
        {
            name:    "valid config",
            config:  map[string]interface{}{"path": "/tmp/test.json"},
            wantErr: false,
        },
        {
            name:    "missing path",
            config:  map[string]interface{}{},
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            plugin := &JSONInputPlugin{config: tt.config}
            err := plugin.Validate()
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

## Configuration Schema

Define a JSON Schema for your plugin configuration:

```go
func (p *MyPlugin) GetMetadata() types.PluginMetadata {
    return types.PluginMetadata{
        Name:    "my-plugin",
        Type:    types.PluginTypeInput,
        Version: "1.0.0",
        ConfigSchema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "host": map[string]interface{}{
                    "type":        "string",
                    "description": "Database host",
                },
                "port": map[string]interface{}{
                    "type":        "integer",
                    "description": "Database port",
                    "default":     3306,
                },
            },
            "required": []string{"host"},
        },
    }
}
```

## Best Practices

### 1. Error Handling

Always wrap errors with context:

```go
if err != nil {
    return fmt.Errorf("failed to connect to database: %w", err)
}
```

### 2. Resource Management

Always clean up resources:

```go
func (p *MyPlugin) Close() error {
    if p.connection != nil {
        return p.connection.Close()
    }
    return nil
}
```

### 3. Configuration Validation

Validate configuration early:

```go
func (p *MyPlugin) Validate() error {
    if p.config["required_field"] == nil {
        return fmt.Errorf("required_field is missing")
    }
    return nil
}
```

### 4. Progress Tracking

Update progress for long-running operations:

```go
p.progress.ProcessedRecords += int64(len(records))
p.progress.LastUpdateTime = time.Now()
```

### 5. Statistics Collection

Collect meaningful statistics:

```go
p.stats.RecordsIn++
p.stats.RecordsOut++
p.stats.Duration = time.Since(p.startTime)
```

## Common Patterns

### Database Connection Pooling

```go
type MySQLPlugin struct {
    db *sql.DB
}

func (p *MySQLPlugin) Connect() error {
    db, err := sql.Open("mysql", p.buildDSN())
    if err != nil {
        return err
    }
    
    db.SetMaxOpenConns(10)
    db.SetMaxIdleConns(5)
    db.SetConnMaxLifetime(time.Hour)
    
    p.db = db
    return db.Ping()
}
```

### Batch Processing

```go
func (p *MyPlugin) ReadBatch(batchSize int) (*types.DataBatch, error) {
    rows, err := p.db.Query("SELECT * FROM table LIMIT ?", batchSize)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var records []types.Record
    for rows.Next() {
        // Scan row into record
        records = append(records, record)
    }
    
    return &types.DataBatch{
        Schema:  schema,
        Records: records,
    }, nil
}
```

### Retry Logic

```go
func (p *MyPlugin) WriteBatch(data *types.DataBatch) error {
    maxRetries := 3
    backoff := time.Second
    
    for i := 0; i < maxRetries; i++ {
        err := p.writeWithRetry(data)
        if err == nil {
            return nil
        }
        
        if i < maxRetries-1 {
            time.Sleep(backoff)
            backoff *= 2
        }
    }
    
    return fmt.Errorf("failed after %d retries", maxRetries)
}
```

## Testing Guidelines

### Unit Tests

Test each method independently:

```go
func TestMyPlugin_Initialize(t *testing.T) {
    // Test initialization logic
}

func TestMyPlugin_Validate(t *testing.T) {
    // Test validation logic
}

func TestMyPlugin_ReadBatch(t *testing.T) {
    // Test batch reading
}
```

### Integration Tests

Test with real data sources:

```go
// +build integration

func TestMyPlugin_Integration(t *testing.T) {
    // Start test database/service
    // Test actual data reading/writing
    // Clean up
}
```

### Table-Driven Tests

```go
func TestMyPlugin_Process(t *testing.T) {
    tests := []struct {
        name     string
        input    *types.DataBatch
        expected *types.DataBatch
        wantErr  bool
    }{
        {
            name:     "valid input",
            input:    createTestBatch(),
            expected: createExpectedBatch(),
            wantErr:  false,
        },
        // More test cases...
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test logic
        })
    }
}
```

## Plugin Examples

### Input Plugin Examples
- CSV File Reader (✅ Implemented)
- JSON File Reader
- MySQL Database Reader
- PostgreSQL Database Reader
- Kafka Consumer
- HTTP API Client
- MongoDB Reader
- Redis Reader

### Processor Plugin Examples
- Filter (✅ Implemented)
- Mapping (✅ Implemented)
- Type Converter
- Aggregator
- Joiner
- Splitter
- Deduplicator
- Validator

### Output Plugin Examples
- CSV File Writer (✅ Implemented)
- JSON File Writer
- PostgreSQL Writer
- MySQL Writer
- Elasticsearch Indexer
- Kafka Producer
- S3 Uploader

## Troubleshooting

### Plugin Not Found

Ensure the plugin is:
1. Registered in `init()` function
2. Imported in `plugins/loader.go`
3. Listed in enabled plugins in config

### Build Errors

Check that:
1. All imports are correct
2. Interface methods are fully implemented
3. Type assertions are valid

### Runtime Errors

Enable debug logging:
```yaml
observability:
  logs:
    local:
      level: debug
```

## Performance Optimization

### 1. Batch Size Tuning

```go
// Adjust based on record size and memory
optimalBatchSize := calculateOptimalBatch(avgRecordSize)
```

### 2. Connection Pooling

```go
db.SetMaxOpenConns(runtime.NumCPU() * 2)
db.SetMaxIdleConns(runtime.NumCPU())
```

### 3. Memory Management

```go
// Reuse buffers
var recordBuffer []types.Record
recordBuffer = recordBuffer[:0]
```

### 4. Parallel Processing

```go
// Use goroutines for independent operations
var wg sync.WaitGroup
for _, batch := range batches {
    wg.Add(1)
    go func(b *types.DataBatch) {
        defer wg.Done()
        process(b)
    }(batch)
}
wg.Wait()
```

## Contributing Your Plugin

1. Fork the repository
2. Create your plugin following this guide
3. Write comprehensive tests
4. Update documentation
5. Submit a pull request

For questions, open an issue on GitHub.

---

**Happy Plugin Development!** 🔌
