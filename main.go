package main

import (
	"fmt"
	"github.com/atlanssia/fustgo/source"
	"github.com/atlanssia/fustgo/sink"
	"github.com/atlanssia/fustgo/processor"
	"github.com/BurntSushi/toml"
	"io/ioutil"
	"os"
)

// Config 定义配置文件结构
type Config struct {
	Source map[string]interface{} `toml:"source"`
	Sink   map[string]interface{} `toml:"sink"`
}

// loadConfig 根据文件类型加载配置文件
func loadConfig(filename string) (*Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Config
	if _, err := toml.Decode(string(data), &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func main() {
	fmt.Println("Fustgo Data Sync System")

	// 加载配置文件
	config, err := loadConfig("config/config.toml") // 配置文件路径更新为 config/config.toml
	if err != nil {
		fmt.Println("Failed to load config:", err)
		os.Exit(1)
	}

	// 根据配置创建源和目标
	src := source.NewSource(config.Source)
	snk := sink.NewSink(config.Sink)
	proc := processor.NewProcessor()

	src.Connect()
	snk.Connect()

	// 读取、处理、写入数据
	data, _ := src.Read()
	processedData, _ := proc.Process(data)
	snk.Write(processedData)
}