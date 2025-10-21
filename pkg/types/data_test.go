package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDataType_String(t *testing.T) {
	tests := []struct {
		name     string
		dataType DataType
		expected string
	}{
		{"String type", DataTypeString, "STRING"},
		{"Int type", DataTypeInt, "INT"},
		{"BigInt type", DataTypeBigInt, "BIGINT"},
		{"Float type", DataTypeFloat, "FLOAT"},
		{"Double type", DataTypeDouble, "DOUBLE"},
		{"Bool type", DataTypeBool, "BOOL"},
		{"Date type", DataTypeDate, "DATE"},
		{"Timestamp type", DataTypeTimestamp, "TIMESTAMP"},
		{"Bytes type", DataTypeBytes, "BYTES"},
		{"JSON type", DataTypeJSON, "JSON"},
		{"Unknown type", DataTypeUnknown, "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.dataType.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDataBatch_Size(t *testing.T) {
	batch := &DataBatch{
		Records: []Record{
			{Values: []interface{}{"test1"}},
			{Values: []interface{}{"test2"}},
			{Values: []interface{}{"test3"}},
		},
	}

	assert.Equal(t, 3, batch.Size())
}

func TestDataBatch_IsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		batch    *DataBatch
		expected bool
	}{
		{
			name: "Empty batch",
			batch: &DataBatch{
				Records: []Record{},
			},
			expected: true,
		},
		{
			name: "Non-empty batch",
			batch: &DataBatch{
				Records: []Record{
					{Values: []interface{}{"test"}},
				},
			},
			expected: false,
		},
		{
			name: "Nil records",
			batch: &DataBatch{
				Records: nil,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.batch.IsEmpty()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProgress_Percentage(t *testing.T) {
	tests := []struct {
		name     string
		progress *Progress
		expected float64
	}{
		{
			name: "50% complete",
			progress: &Progress{
				TotalRecords:     100,
				ProcessedRecords: 50,
			},
			expected: 50.0,
		},
		{
			name: "100% complete",
			progress: &Progress{
				TotalRecords:     100,
				ProcessedRecords: 100,
			},
			expected: 100.0,
		},
		{
			name: "0% complete",
			progress: &Progress{
				TotalRecords:     100,
				ProcessedRecords: 0,
			},
			expected: 0.0,
		},
		{
			name: "Zero total",
			progress: &Progress{
				TotalRecords:     0,
				ProcessedRecords: 0,
			},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.progress.Percentage()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestColumn(t *testing.T) {
	col := Column{
		Name:         "user_id",
		DataType:     DataTypeInt,
		Nullable:     false,
		DefaultValue: 0,
	}

	assert.Equal(t, "user_id", col.Name)
	assert.Equal(t, DataTypeInt, col.DataType)
	assert.False(t, col.Nullable)
	assert.Equal(t, 0, col.DefaultValue)
}

func TestSchema(t *testing.T) {
	schema := Schema{
		Columns: []Column{
			{Name: "id", DataType: DataTypeInt},
			{Name: "name", DataType: DataTypeString},
		},
		PrimaryKeys: []string{"id"},
	}

	assert.Len(t, schema.Columns, 2)
	assert.Len(t, schema.PrimaryKeys, 1)
	assert.Equal(t, "id", schema.PrimaryKeys[0])
}

func TestRecord(t *testing.T) {
	record := Record{
		Values: []interface{}{1, "John", "john@example.com"},
		Metadata: map[string]string{
			"source": "mysql",
			"table":  "users",
		},
	}

	assert.Len(t, record.Values, 3)
	assert.Equal(t, 1, record.Values[0])
	assert.Equal(t, "John", record.Values[1])
	assert.Equal(t, "mysql", record.Metadata["source"])
}

func TestCheckpoint(t *testing.T) {
	now := time.Now()
	checkpoint := Checkpoint{
		Position:  int64(12345),
		Timestamp: now,
		Metadata: map[string]string{
			"file":   "data.csv",
			"offset": "12345",
		},
	}

	assert.Equal(t, int64(12345), checkpoint.Position)
	assert.Equal(t, now, checkpoint.Timestamp)
	assert.Equal(t, "data.csv", checkpoint.Metadata["file"])
}
