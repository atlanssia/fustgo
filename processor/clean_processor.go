package processor

import (
	"fmt"
	"strings"
)

// CleanProcessor 实现数据清洗的处理器
type CleanProcessor struct {
	Enabled bool // 新增字段，用于控制处理器是否启用
}

// NewCleanProcessor 创建一个新的 CleanProcessor 实例
func NewCleanProcessor(enabled bool) *CleanProcessor {
	return &CleanProcessor{Enabled: enabled}
}

// Process 实现数据清洗逻辑
func (c *CleanProcessor) Process(data []byte) ([]byte, error) {
	if !c.Enabled {
		return data, nil // 如果处理器未启用，直接返回原始数据
	}

	// 示例：去除数据中的空格和换行符
	cleanedData := strings.TrimSpace(string(data))
	cleanedData = strings.ReplaceAll(cleanedData, "\n", "")
	fmt.Println("Data cleaned:", cleanedData)
	return []byte(cleanedData), nil
}