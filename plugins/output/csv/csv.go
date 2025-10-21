package csv

import (
	"encoding/csv"
	"fmt"
	"os"
	"time"

	"github.com/atlanssia/fustgo/pkg/types"
)

// CSVOutputPlugin writes data to CSV files
type CSVOutputPlugin struct {
	config     map[string]interface{}
	file       *os.File
	writer     *csv.Writer
	delimiter  rune
	writeHeader bool
	headerWritten bool
	stats      *types.WriteStatistics
	startTime  time.Time
}

// Name returns the plugin name
func (p *CSVOutputPlugin) Name() string {
	return "csv"
}

// Type returns the plugin type
func (p *CSVOutputPlugin) Type() types.PluginType {
	return types.PluginTypeOutput
}

// Initialize initializes the CSV output plugin
func (p *CSVOutputPlugin) Initialize(config map[string]interface{}) error {
	p.config = config
	p.delimiter = ','
	p.writeHeader = true
	p.headerWritten = false
	
	// Parse configuration
	if delimiter, ok := config["delimiter"].(string); ok && len(delimiter) > 0 {
		p.delimiter = rune(delimiter[0])
	}
	
	if writeHeader, ok := config["write_header"].(bool); ok {
		p.writeHeader = writeHeader
	}
	
	// Initialize statistics
	p.stats = &types.WriteStatistics{
		RecordsWritten: 0,
		RecordsFailed:  0,
		BytesWritten:   0,
	}
	
	return nil
}

// Validate validates the configuration
func (p *CSVOutputPlugin) Validate() error {
	if p.config["path"] == nil {
		return fmt.Errorf("csv output: path is required")
	}
	return nil
}

// Connect creates/opens the CSV file
func (p *CSVOutputPlugin) Connect() error {
	path, ok := p.config["path"].(string)
	if !ok {
		return fmt.Errorf("csv output: invalid path configuration")
	}
	
	// Determine file mode
	mode := os.O_CREATE | os.O_WRONLY
	if append, ok := p.config["append"].(bool); ok && append {
		mode |= os.O_APPEND
		p.headerWritten = true // Don't write header in append mode
	} else {
		mode |= os.O_TRUNC
	}
	
	file, err := os.OpenFile(path, mode, 0644)
	if err != nil {
		return fmt.Errorf("csv output: failed to open file: %w", err)
	}
	p.file = file
	
	p.writer = csv.NewWriter(file)
	p.writer.Comma = p.delimiter
	p.startTime = time.Now()
	
	return nil
}

// WriteBatch writes a batch of records to the CSV file
func (p *CSVOutputPlugin) WriteBatch(data *types.DataBatch) error {
	if p.writer == nil {
		return fmt.Errorf("csv output: not connected")
	}
	
	if data == nil || data.IsEmpty() {
		return nil
	}
	
	// Write header if needed
	if p.writeHeader && !p.headerWritten {
		header := make([]string, len(data.Schema.Columns))
		for i, col := range data.Schema.Columns {
			header[i] = col.Name
		}
		if err := p.writer.Write(header); err != nil {
			return fmt.Errorf("csv output: failed to write header: %w", err)
		}
		p.headerWritten = true
	}
	
	// Write records
	for _, record := range data.Records {
		row := make([]string, len(record.Values))
		for i, val := range record.Values {
			row[i] = p.formatValue(val)
		}
		
		if err := p.writer.Write(row); err != nil {
			p.stats.RecordsFailed++
			return fmt.Errorf("csv output: failed to write record: %w", err)
		}
		
		p.stats.RecordsWritten++
	}
	
	return nil
}

// Flush flushes any buffered data to the file
func (p *CSVOutputPlugin) Flush() error {
	if p.writer != nil {
		p.writer.Flush()
		if err := p.writer.Error(); err != nil {
			return fmt.Errorf("csv output: flush error: %w", err)
		}
		
		if p.file != nil {
			if err := p.file.Sync(); err != nil {
				return fmt.Errorf("csv output: sync error: %w", err)
			}
		}
	}
	
	p.stats.Duration = time.Since(p.startTime)
	return nil
}

// GetWriteStatistics returns write statistics
func (p *CSVOutputPlugin) GetWriteStatistics() *types.WriteStatistics {
	p.stats.Duration = time.Since(p.startTime)
	return p.stats
}

// Close closes the CSV file
func (p *CSVOutputPlugin) Close() error {
	if err := p.Flush(); err != nil {
		return err
	}
	
	if p.file != nil {
		return p.file.Close()
	}
	return nil
}

// GetMetadata returns plugin metadata
func (p *CSVOutputPlugin) GetMetadata() types.PluginMetadata {
	return types.PluginMetadata{
		Name:        "csv",
		Type:        types.PluginTypeOutput,
		Version:     "1.0.0",
		Description: "CSV file output plugin",
		DataSourceType: "file",
		ConfigSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Path to output CSV file",
				},
				"write_header": map[string]interface{}{
					"type":        "boolean",
					"description": "Whether to write header row",
					"default":     true,
				},
				"delimiter": map[string]interface{}{
					"type":        "string",
					"description": "Field delimiter character",
					"default":     ",",
				},
				"append": map[string]interface{}{
					"type":        "boolean",
					"description": "Append to existing file",
					"default":     false,
				},
			},
			"required": []string{"path"},
		},
	}
}

// formatValue converts a value to string for CSV output
func (p *CSVOutputPlugin) formatValue(val interface{}) string {
	if val == nil {
		return ""
	}
	return fmt.Sprintf("%v", val)
}
