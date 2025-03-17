package sink

import (
	"fmt"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
)

// DatabaseSink 实现了 Sink 接口，用于将数据写入数据库
type DatabaseSink struct {
	db *sql.DB
}

// NewDatabaseSink 创建一个新的 DatabaseSink 实例
func NewDatabaseSink(host string, port int, username string, password string, database string) (*DatabaseSink, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", username, password, host, port, database)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	return &DatabaseSink{db: db}, nil
}

// Connect 连接到数据库
func (d *DatabaseSink) Connect() error {
	fmt.Println("Database Sink Connected")
	return d.db.Ping()
}

// Disconnect 断开与数据库的连接
func (d *DatabaseSink) Disconnect() error {
	fmt.Println("Database Sink Disconnected")
	return d.db.Close()
}

// Write 将数据写入数据库
func (d *DatabaseSink) Write(data []byte) error {
	_, err := d.db.Exec("INSERT INTO data_table (data) VALUES (?)", string(data))
	if err != nil {
		return err
	}
	fmt.Println("Data written to database:", string(data))
	return nil
}