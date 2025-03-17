package plugin

import (
	"github.com/atlanssia/fustgo/source"
)

// Sink 接口定义
type Sink interface {
	Connect() error
	Disconnect() error
	Write(data []byte) error
}

// plugins 存储已注册的插件
var plugins = make(map[string]interface{})

// Get 根据插件类型获取已注册的插件
func Get(pluginType string) (interface{}, bool) {
	if plugin, exists := plugins[pluginType]; exists {
		return plugin, true
	}
	return nil, false
}

func init() {
	// 注册 HTTP Source 插件
	RegisterSource("http", &source.HTTPSource{})

	// 注册 Database Sink 插件
	RegisterSink("database", &sink.DatabaseSink{})
}