package filter

import (
	"fmt"
	"strings"
	"time"

	"github.com/atlanssia/fustgo/pkg/types"
)

// FilterProcessor filters records based on conditions
type FilterProcessor struct {
	config    map[string]interface{}
	condition string
	mode      string // "include" or "exclude"
	stats     *types.ProcessStatistics
	startTime time.Time
}

// Name returns the plugin name
func (p *FilterProcessor) Name() string {
	return "filter"
}

// Type returns the plugin type
func (p *FilterProcessor) Type() types.PluginType {
	return types.PluginTypeProcessor
}

// Initialize initializes the filter processor
func (p *FilterProcessor) Initialize(config map[string]interface{}) error {
	p.config = config
	p.mode = "include"
	
	// Parse configuration
	if condition, ok := config["condition"].(string); ok {
		p.condition = condition
	}
	
	if mode, ok := config["mode"].(string); ok {
		if mode == "include" || mode == "exclude" {
			p.mode = mode
		} else {
			return fmt.Errorf("filter: invalid mode '%s', must be 'include' or 'exclude'", mode)
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
func (p *FilterProcessor) Validate() error {
	if p.condition == "" {
		return fmt.Errorf("filter: condition is required")
	}
	return nil
}

// Process processes a batch of data
func (p *FilterProcessor) Process(input *types.DataBatch) (*types.DataBatch, error) {
	if input == nil || input.IsEmpty() {
		return input, nil
	}
	
	var filteredRecords []types.Record
	
	for _, record := range input.Records {
		p.stats.RecordsIn++
		
		match, err := p.evaluateCondition(record, input.Schema)
		if err != nil {
			p.stats.Errors++
			continue
		}
		
		// Include or exclude based on mode
		include := (p.mode == "include" && match) || (p.mode == "exclude" && !match)
		
		if include {
			filteredRecords = append(filteredRecords, record)
			p.stats.RecordsOut++
		} else {
			p.stats.Filtered++
		}
	}
	
	output := &types.DataBatch{
		Schema:     input.Schema,
		Records:    filteredRecords,
		Metadata:   input.Metadata,
		Checkpoint: input.Checkpoint,
	}
	
	return output, nil
}

// GetStatistics returns processing statistics
func (p *FilterProcessor) GetStatistics() *types.ProcessStatistics {
	p.stats.Duration = time.Since(p.startTime)
	return p.stats
}

// Close closes the processor
func (p *FilterProcessor) Close() error {
	p.stats.Duration = time.Since(p.startTime)
	return nil
}

// GetMetadata returns plugin metadata
func (p *FilterProcessor) GetMetadata() types.PluginMetadata {
	return types.PluginMetadata{
		Name:        "filter",
		Type:        types.PluginTypeProcessor,
		Version:     "1.0.0",
		Description: "Filter records based on conditions",
		ConfigSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"condition": map[string]interface{}{
					"type":        "string",
					"description": "Filter condition expression",
				},
				"mode": map[string]interface{}{
					"type":        "string",
					"description": "Filter mode: 'include' or 'exclude'",
					"enum":        []string{"include", "exclude"},
					"default":     "include",
				},
			},
			"required": []string{"condition"},
		},
	}
}

// evaluateCondition evaluates the filter condition for a record
// This is a simple implementation - production version should use a proper expression evaluator
func (p *FilterProcessor) evaluateCondition(record types.Record, schema types.Schema) (bool, error) {
	// Simple field existence check: "field_name"
	if !strings.Contains(p.condition, " ") {
		fieldName := strings.TrimSpace(p.condition)
		fieldIdx := p.findFieldIndex(fieldName, schema)
		if fieldIdx >= 0 && fieldIdx < len(record.Values) {
			return record.Values[fieldIdx] != nil, nil
		}
		return false, nil
	}
	
	// Simple comparisons: "field_name > value", "field_name = value", etc.
	parts := strings.Fields(p.condition)
	if len(parts) >= 3 {
		fieldName := parts[0]
		operator := parts[1]
		valueStr := strings.Join(parts[2:], " ")
		
		fieldIdx := p.findFieldIndex(fieldName, schema)
		if fieldIdx < 0 || fieldIdx >= len(record.Values) {
			return false, nil
		}
		
		fieldValue := record.Values[fieldIdx]
		if fieldValue == nil {
			return operator == "=" && valueStr == "null", nil
		}
		
		return p.compareValues(fieldValue, operator, valueStr)
	}
	
	// Default: include all records if condition format is unknown
	return true, nil
}

// findFieldIndex finds the index of a field by name
func (p *FilterProcessor) findFieldIndex(fieldName string, schema types.Schema) int {
	for i, col := range schema.Columns {
		if col.Name == fieldName {
			return i
		}
	}
	return -1
}

// compareValues compares a field value with a target value using an operator
func (p *FilterProcessor) compareValues(fieldValue interface{}, operator, targetValue string) (bool, error) {
	switch operator {
	case "=", "==":
		return fmt.Sprintf("%v", fieldValue) == targetValue, nil
	case "!=":
		return fmt.Sprintf("%v", fieldValue) != targetValue, nil
	case ">":
		// Simple numeric comparison (would need proper type handling in production)
		return fmt.Sprintf("%v", fieldValue) > targetValue, nil
	case "<":
		return fmt.Sprintf("%v", fieldValue) < targetValue, nil
	case ">=":
		return fmt.Sprintf("%v", fieldValue) >= targetValue, nil
	case "<=":
		return fmt.Sprintf("%v", fieldValue) <= targetValue, nil
	case "contains":
		return strings.Contains(fmt.Sprintf("%v", fieldValue), targetValue), nil
	default:
		return false, fmt.Errorf("unknown operator: %s", operator)
	}
}
