package csv

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/atlanssia/fustgo/pkg/types"
)

// CSVInputPlugin reads data from CSV files
type CSVInputPlugin struct {
	config       map[string]interface{}
	file         *os.File
	reader       *csv.Reader
	header       []string
	schema       *types.Schema
	hasHeader    bool
	delimiter    rune
	currentRow   int
	totalRows    int
	progress     *types.Progress
}

// Name returns the plugin name
func (p *CSVInputPlugin) Name() string {
	return "csv"
}

// Type returns the plugin type
func (p *CSVInputPlugin) Type() types.PluginType {
	return types.PluginTypeInput
}

// Initialize initializes the CSV input plugin
func (p *CSVInputPlugin) Initialize(config map[string]interface{}) error {
	p.config = config
	p.hasHeader = true
	p.delimiter = ','
	p.currentRow = 0
	
	// Parse configuration
	if hasHeader, ok := config["has_header"].(bool); ok {
		p.hasHeader = hasHeader
	}
	
	if delimiter, ok := config["delimiter"].(string); ok && len(delimiter) > 0 {
		p.delimiter = rune(delimiter[0])
	}
	
	// Initialize progress
	p.progress = &types.Progress{
		TotalRecords:   0,
		ProcessedRecords: 0,
	}
	
	return nil
}

// Validate validates the configuration
func (p *CSVInputPlugin) Validate() error {
	if p.config["path"] == nil {
		return fmt.Errorf("csv input: path is required")
	}
	return nil
}

// Connect opens the CSV file and reads the header
func (p *CSVInputPlugin) Connect() error {
	path, ok := p.config["path"].(string)
	if !ok {
		return fmt.Errorf("csv input: invalid path configuration")
	}
	
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("csv input: failed to open file: %w", err)
	}
	p.file = file
	
	p.reader = csv.NewReader(file)
	p.reader.Comma = p.delimiter
	p.reader.TrimLeadingSpace = true
	
	// Read header if configured
	if p.hasHeader {
		header, err := p.reader.Read()
		if err != nil {
			return fmt.Errorf("csv input: failed to read header: %w", err)
		}
		p.header = header
		
		// Build schema from header
		columns := make([]types.Column, len(header))
		for i, name := range header {
			columns[i] = types.Column{
				Name:     strings.TrimSpace(name),
				DataType: types.DataTypeString, // Default to string, can be inferred
				Nullable: true,
			}
		}
		p.schema = &types.Schema{
			Columns: columns,
		}
	}
	
	return nil
}

// ReadBatch reads a batch of records from the CSV file
func (p *CSVInputPlugin) ReadBatch(batchSize int) (*types.DataBatch, error) {
	if p.reader == nil {
		return nil, fmt.Errorf("csv input: not connected")
	}
	
	var records []types.Record
	
	for i := 0; i < batchSize; i++ {
		row, err := p.reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("csv input: failed to read row %d: %w", p.currentRow, err)
		}
		
		// Convert row to record
		values := make([]interface{}, len(row))
		for j, val := range row {
			values[j] = p.inferValue(val)
		}
		
		record := types.Record{
			Values: values,
			Metadata: map[string]string{
				"row_number": strconv.Itoa(p.currentRow),
			},
		}
		
		records = append(records, record)
		p.currentRow++
		p.progress.ProcessedRecords++
	}
	
	if len(records) == 0 {
		return nil, io.EOF
	}
	
	batch := &types.DataBatch{
		Schema:  *p.schema,
		Records: records,
		Metadata: map[string]string{
			"source": "csv",
			"file":   p.config["path"].(string),
		},
	}
	
	return batch, nil
}

// HasNext checks if there are more records to read
func (p *CSVInputPlugin) HasNext() bool {
	// We can't know without reading, so we return true until EOF
	return true
}

// GetProgress returns the current reading progress
func (p *CSVInputPlugin) GetProgress() *types.Progress {
	return p.progress
}

// Close closes the CSV file
func (p *CSVInputPlugin) Close() error {
	if p.file != nil {
		return p.file.Close()
	}
	return nil
}

// GetMetadata returns plugin metadata
func (p *CSVInputPlugin) GetMetadata() types.PluginMetadata {
	return types.PluginMetadata{
		Name:        "csv",
		Type:        types.PluginTypeInput,
		Version:     "1.0.0",
		Description: "CSV file input plugin",
		DataSourceType: "file",
		ConfigSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Path to CSV file",
				},
				"has_header": map[string]interface{}{
					"type":        "boolean",
					"description": "Whether the file has a header row",
					"default":     true,
				},
				"delimiter": map[string]interface{}{
					"type":        "string",
					"description": "Field delimiter character",
					"default":     ",",
				},
			},
			"required": []string{"path"},
		},
	}
}

// inferValue attempts to infer the type of a string value
func (p *CSVInputPlugin) inferValue(val string) interface{} {
	val = strings.TrimSpace(val)
	
	// Empty string
	if val == "" {
		return nil
	}
	
	// Try integer
	if intVal, err := strconv.ParseInt(val, 10, 64); err == nil {
		return intVal
	}
	
	// Try float
	if floatVal, err := strconv.ParseFloat(val, 64); err == nil {
		return floatVal
	}
	
	// Try boolean
	if boolVal, err := strconv.ParseBool(val); err == nil {
		return boolVal
	}
	
	// Default to string
	return val
}
