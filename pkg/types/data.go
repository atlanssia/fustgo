package types

import "time"

// DataType represents the type of a column value
type DataType int

const (
	DataTypeUnknown DataType = iota
	DataTypeString
	DataTypeInt
	DataTypeBigInt
	DataTypeFloat
	DataTypeDouble
	DataTypeBool
	DataTypeDate
	DataTypeTimestamp
	DataTypeBytes
	DataTypeJSON
)

// String returns the string representation of DataType
func (dt DataType) String() string {
	switch dt {
	case DataTypeString:
		return "STRING"
	case DataTypeInt:
		return "INT"
	case DataTypeBigInt:
		return "BIGINT"
	case DataTypeFloat:
		return "FLOAT"
	case DataTypeDouble:
		return "DOUBLE"
	case DataTypeBool:
		return "BOOL"
	case DataTypeDate:
		return "DATE"
	case DataTypeTimestamp:
		return "TIMESTAMP"
	case DataTypeBytes:
		return "BYTES"
	case DataTypeJSON:
		return "JSON"
	default:
		return "UNKNOWN"
	}
}

// Column represents a column definition in the schema
type Column struct {
	Name         string      `json:"name"`
	DataType     DataType    `json:"data_type"`
	Nullable     bool        `json:"nullable"`
	DefaultValue interface{} `json:"default_value,omitempty"`
}

// Schema represents the structure of data
type Schema struct {
	Columns     []Column `json:"columns"`
	PrimaryKeys []string `json:"primary_keys,omitempty"`
}

// Record represents a single row of data
type Record struct {
	Values   []interface{}     `json:"values"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Checkpoint represents a position in the data stream for recovery
type Checkpoint struct {
	Position  interface{}       `json:"position"`
	Timestamp time.Time         `json:"timestamp"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// DataBatch represents a batch of records with schema
type DataBatch struct {
	Schema     Schema            `json:"schema"`
	Records    []Record          `json:"records"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	Checkpoint *Checkpoint       `json:"checkpoint,omitempty"`
}

// Size returns the number of records in the batch
func (db *DataBatch) Size() int {
	return len(db.Records)
}

// IsEmpty checks if the batch is empty
func (db *DataBatch) IsEmpty() bool {
	return len(db.Records) == 0
}

// Progress represents the progress of a data operation
type Progress struct {
	TotalRecords     int64     `json:"total_records"`
	ProcessedRecords int64     `json:"processed_records"`
	FailedRecords    int64     `json:"failed_records"`
	BytesTransferred int64     `json:"bytes_transferred"`
	StartTime        time.Time `json:"start_time"`
	LastUpdateTime   time.Time `json:"last_update_time"`
}

// Percentage calculates the completion percentage
func (p *Progress) Percentage() float64 {
	if p.TotalRecords == 0 {
		return 0
	}
	return float64(p.ProcessedRecords) / float64(p.TotalRecords) * 100
}

// ProcessStatistics represents statistics from a processor
type ProcessStatistics struct {
	RecordsIn  int64         `json:"records_in"`
	RecordsOut int64         `json:"records_out"`
	Filtered   int64         `json:"filtered"`
	Errors     int64         `json:"errors"`
	Duration   time.Duration `json:"duration"`
}

// WriteStatistics represents statistics from a writer
type WriteStatistics struct {
	RecordsWritten int64         `json:"records_written"`
	RecordsFailed  int64         `json:"records_failed"`
	BytesWritten   int64         `json:"bytes_written"`
	Duration       time.Duration `json:"duration"`
}
