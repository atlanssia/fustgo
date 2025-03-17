package source

import (
	"fmt"
	"github.com/atlanssia/fustgo/plugin"
)

// Source 接口定义
type Source interface {
	Connect() error
	Disconnect() error
	Read() ([]byte, error)
}

// BaseSource 基础实现
type BaseSource struct{}

func (b *BaseSource) Connect() error {
	fmt.Println("Source Connected")
	return nil
}

func (b *BaseSource) Disconnect() error {
	fmt.Println("Source Disconnected")
	return nil
}

func (b *BaseSource) Read() ([]byte, error) {
	fmt.Println("Reading data from source")
	return []byte("data"), nil
}

// NewSource 根据配置从插件系统中获取对应的 Source 实现
func NewSource(config map[string]interface{}) Source {
	sourceType := config["type"].(string)
	if plugin, exists := plugin.Get(sourceType); exists {
		if src, ok := plugin.(Source); ok {
			return src
		}
	}
	return &BaseSource{}
}