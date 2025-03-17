package sink

import (
	"fmt"
	"github.com/atlanssia/fustgo/plugin" // 添加对 plugin 包的导入
)

// BaseSink 基础实现
type BaseSink struct{}

func (b *BaseSink) Connect() error {
	fmt.Println("Sink Connected")
	return nil
}

func (b *BaseSink) Disconnect() error {
	fmt.Println("Sink Disconnected")
	return nil
}

func (b *BaseSink) Write(data []byte) error {
	fmt.Println("Writing data to sink:", string(data))
	return nil
}

// NewSink 根据配置从插件系统中获取对应的 Sink 实现
func NewSink(config map[string]interface{}) plugin.Sink {
	sinkType := config["type"].(string)
	if plugin, exists := plugin.Get(sinkType); exists {
		if snk, ok := plugin.(plugin.Sink); ok {
			return snk
		}
	}
	return &BaseSink{}
}