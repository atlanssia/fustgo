package source

import (
	"fmt"
	"net/http"
	"io/ioutil"
)

// HTTPSource 实现了 Source 接口，用于从 HTTP 服务读取数据
type HTTPSource struct {
	url string
}

// NewHTTPSource 创建一个新的 HTTPSource 实例
func NewHTTPSource(url string) *HTTPSource {
	return &HTTPSource{url: url}
}

// Connect 连接到 HTTP 服务
func (h *HTTPSource) Connect() error {
	fmt.Println("HTTP Source Connected")
	return nil
}

// Disconnect 断开与 HTTP 服务的连接
func (h *HTTPSource) Disconnect() error {
	fmt.Println("HTTP Source Disconnected")
	return nil
}

// Read 从 HTTP 服务读取数据
func (h *HTTPSource) Read() ([]byte, error) {
	resp, err := http.Get(h.url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return data, nil
}