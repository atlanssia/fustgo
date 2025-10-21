package mapping

import (
	"fmt"
	"time"

	"github.com/atlanssia/fustgo/pkg/types"
)

// MappingProcessor maps and renames fields
type MappingProcessor struct {
	config        map[string]interface{}
	fieldMappings map[string]string // old_name -> new_name
	stats         *types.ProcessStatistics
	startTime     time.Time
}

// Name returns the plugin name
func (p *MappingProcessor) Name() string {
	return "mapping"
}

// Type returns the plugin type
func (p *MappingProcessor) Type() types.PluginType {
	return types.PluginTypeProcessor
}

// Initialize initializes the mapping processor
func (p *MappingProcessor) Initialize(config map[string]interface{}) error {
	p.config = config
	p.fieldMappings = make(map[string]string)
	
	// Parse field mappings
	if mappings, ok := config["field_mappings"].(map[string]interface{}); ok {
		for oldName, newName := range mappings {
			if newNameStr, ok := newName.(string); ok {
				p.fieldMappings[oldName] = newNameStr
			}
		}
	}
	
	// Initialize statistics
	p.stats = &types.ProcessStatistics{
		RecordsIn:  0,
		RecordsOut: 0,
		Filtered:   0,
		Errors:     0,
	}
	
	p.startTime = time.Now()
	
	return nil
}

// Validate validates the configuration
func (p *MappingProcessor) Validate() error {
	if len(p.fieldMappings) == 0 {
		return fmt.Errorf("mapping: at least one field mapping is required")
	}
	return nil
}

// Process processes a batch of data
func (p *MappingProcessor) Process(input *types.DataBatch) (*types.DataBatch, error) {
	if input == nil || input.IsEmpty() {
		return input, nil
	}
	
	// Create new schema with mapped column names
	newColumns := make([]types.Column, len(input.Schema.Columns))
	for i, col := range input.Schema.Columns {
		newCol := col
		if newName, exists := p.fieldMappings[col.Name]; exists {
			newCol.Name = newName
		}
		newColumns[i] = newCol
	}
	
	newSchema := types.Schema{
		Columns:     newColumns,
		PrimaryKeys: p.mapPrimaryKeys(input.Schema.PrimaryKeys),
	}
	
	// Records data remains the same, only schema changes
	output := &types.DataBatch{
		Schema:     newSchema,
		Records:    input.Records,
		Metadata:   input.Metadata,
		Checkpoint: input.Checkpoint,
	}
	
	p.stats.RecordsIn += int64(len(input.Records))
	p.stats.RecordsOut += int64(len(input.Records))
	
	return output, nil
}

// GetStatistics returns processing statistics
func (p *MappingProcessor) GetStatistics() *types.ProcessStatistics {
	p.stats.Duration = time.Since(p.startTime)
	return p.stats
}

// Close closes the processor
func (p *MappingProcessor) Close() error {
	p.stats.Duration = time.Since(p.startTime)
	return nil
}

// GetMetadata returns plugin metadata
func (p *MappingProcessor) GetMetadata() types.PluginMetadata {
	return types.PluginMetadata{
		Name:        "mapping",
		Type:        types.PluginTypeProcessor,
		Version:     "1.0.0",
		Description: "Map and rename fields",
		ConfigSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"field_mappings": map[string]interface{}{
					"type":        "object",
					"description": "Map of old field names to new field names",
					"additionalProperties": map[string]interface{}{
						"type": "string",
					},
				},
			},
			"required": []string{"field_mappings"},
		},
	}
}

// mapPrimaryKeys renames primary key columns based on mappings
func (p *MappingProcessor) mapPrimaryKeys(oldKeys []string) []string {
	newKeys := make([]string, len(oldKeys))
	for i, key := range oldKeys {
		if newName, exists := p.fieldMappings[key]; exists {
			newKeys[i] = newName
		} else {
			newKeys[i] = key
		}
	}
	return newKeys
}
