package processor

import (
	"fmt"
	"github.com/atlanssia/fustgo/plugin" // 添加对 plugin 包的导入
)

// Processor 接口定义
type Processor interface {
	Process(data []byte) ([]byte, error)
}

// BaseProcessor 基础实现
type BaseProcessor struct{}

func (b *BaseProcessor) Process(data []byte) ([]byte, error) {
	fmt.Println("Processing data:", string(data))
	return data, nil
}

// NewProcessor 根据配置从插件系统中获取对应的 Processor 实现
func NewProcessor(config map[string]interface{}) Processor {
	processorType := config["type"].(string)
	if plugin, exists := plugin.Get(processorType); exists {
		if proc, ok := plugin.(Processor); ok {
			return proc
		}
	}

	// 如果配置中包含 clean_processor 的配置，则创建 CleanProcessor 实例
	if cleanConfig, exists := config["clean_processor"]; exists {
		// 添加类型检查，避免 panic
		if cleanConfigMap, ok := cleanConfig.(map[string]interface{}); ok {
			if cleanEnabled, ok := cleanConfigMap["enabled"].(bool); ok {
				return &CleanProcessor{Enabled: cleanEnabled}
			}
		}
	}

	return &BaseProcessor{}
}