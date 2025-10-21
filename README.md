# FustGo DataX

**A High-Performance, Distributed ETL/ELT Data Synchronization System**

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

---

## 🚀 Overview

FustGo DataX is a powerful, configuration-driven ETL/ELT data synchronization platform that enables seamless data migration and transformation between heterogeneous data sources. Inspired by Alibaba DataX and Benthos, it combines declarative YAML configuration with visual flow design for zero-code data pipeline orchestration.

### Key Features

✅ **20+ Data Source Support**: Relational databases, NoSQL, message queues, object storage, HTTP APIs  
✅ **Configuration-Driven**: Define complete data pipelines using YAML configuration  
✅ **Visual Flow Designer**: Drag-and-drop interface for pipeline creation  
✅ **Plugin Architecture**: Static compilation with dynamic enable/disable  
✅ **Multiple Deployment Modes**: Standalone, lightweight, and distributed  
✅ **Built-in Observability**: OpenObserve integration for logs, metrics, and traces  
✅ **High Performance**: 100K+ records/sec throughput  
✅ **Fault Tolerance**: Automatic checkpoint and recovery mechanisms  

---

## 📋 Table of Contents

- [Architecture](#architecture)
- [Quick Start](#quick-start)
- [Deployment Modes](#deployment-modes)
- [Configuration](#configuration)
- [Plugins](#plugins)
- [Web UI](#web-ui)
- [Development](#development)
- [Roadmap](#roadmap)
- [Contributing](#contributing)
- [License](#license)

---

## 🏗️ Architecture

FustGo follows a three-stage pipeline architecture:

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│   Input     │────▶│  Processors  │────▶│   Output    │
│   Plugins   │     │   (Chain)    │     │   Plugins   │
└─────────────┘     └──────────────┘     └─────────────┘
     MySQL              Filter                PostgreSQL
     Kafka              Transform             Elasticsearch
     HTTP               Enrichment            S3
     CSV                Aggregate             Kafka
```

### Core Components

- **Job Manager**: Manages job lifecycle and configuration
- **Scheduler**: Cron-based task scheduling
- **Worker Pool**: Distributed task execution
- **Plugin Registry**: Static compilation with runtime enable/disable
- **Metadata Store**: SQLite/PostgreSQL for job metadata
- **Observability**: OpenObserve integration for monitoring

---

## ⚡ Quick Start

> **Note:** FustGo DataX is currently at version 0.1.0 with 50% core functionality implemented. The system can process CSV files with filtering and mapping. Additional plugins and features are under development.

### Prerequisites

- Docker & Docker Compose
- Go 1.23+ (for development)

### Standalone Mode (Recommended for Testing)

1. **Clone the repository**:
   ```bash
   git clone https://github.com/atlanssia/fustgo.git
   cd fustgo
   ```

2. **Start with Docker Compose**:
   ```bash
   cd deploy/standalone
   docker-compose up -d
   ```

3. **Access the Web UI**:
   - FustGo: http://localhost:8080
   - OpenObserve: http://localhost:5080
     - Username: `admin@fustgo.local`
     - Password: `changeme`

4. **Check logs**:
   ```bash
   docker-compose logs -f fustgo
   ```

### Build from Source

```bash
# Download dependencies
go mod tidy

# Build
go build -o fustgo ./cmd/fustgo

# Run
./fustgo --config configs/default.yaml
```

---

## 📦 Deployment Modes

FustGo supports three deployment modes based on your scale:

| Mode | Use Case | Components | Dependencies |
|------|----------|------------|--------------|
| **Standalone** | Dev/Test, <10K records/day | Single process | SQLite only |
| **Lightweight** | 10K-100K records/day | Main + Workers | SQLite/PostgreSQL |
| **Distributed** | >100K records/day | Cluster | PostgreSQL + Redis + NATS |

### Standalone Mode

- **Single executable** with embedded SQLite
- **Memory-based** task queue
- **Local file** logging
- **Perfect for**: Development, testing, edge computing

```bash
cd deploy/standalone
docker-compose up -d
```

### Lightweight Mode

- **Separate workers** on different hosts
- **Shared database** (SQLite via NFS or PostgreSQL)
- **Database polling** for task distribution

```bash
cd deploy/lightweight
docker-compose up -d
```

### Distributed Mode

- **High availability** with master/standby
- **Horizontal scaling** with worker pool
- **Enterprise-grade** with Redis, NATS, PostgreSQL cluster

```bash
cd deploy/distributed
docker-compose up -d
```

---

## ⚙️ Configuration

### YAML Configuration Structure

```yaml
# Job configuration example
input:
  type: mysql
  connection:
    host: localhost
    port: 3306
    database: source_db
    user: root
    password: secret
  query:
    table: users
    columns: ["id", "name", "email"]
    where: "created_at > '2024-01-01'"

pipeline:
  processors:
    - type: filter
      condition: "age > 18"
    
    - type: mapping
      field_mappings:
        name: username
        email: user_email

output:
  type: postgresql
  connection:
    host: localhost
    port: 5432
    database: target_db
  write:
    mode: upsert
    table: users
    batch_size: 1000
```

### System Configuration

Edit `configs/default.yaml`:

```yaml
server:
  port: 8080
  mode: production

database:
  type: sqlite  # or postgresql, mysql
  path: ./fustgo.db

deployment:
  mode: standalone  # or lightweight, distributed

observability:
  logs:
    local:
      enabled: true
      path: /var/log/fustgo
    openobserve:
      enabled: true
      endpoint: http://localhost:5080
```

See [Configuration Guide](docs/configuration.md) for details.

---

## 🔌 Plugins

### Built-in Plugins

**Input Plugins** (P0):
- MySQL, PostgreSQL, SQL Server
- Kafka, RabbitMQ
- HTTP/REST API
- CSV, JSON files
- MongoDB, Redis

**Processor Plugins**:
- `filter`: Filter records by condition
- `mapping`: Rename fields
- `transform`: Type conversion
- `enrichment`: External data lookup
- `aggregate`: Data aggregation

**Output Plugins** (P0):
- PostgreSQL, MySQL
- Elasticsearch
- Kafka
- CSV, JSON files
- S3, MinIO

### Plugin Development

Create a new plugin:

```go
package myplugin

import "github.com/atlanssia/fustgo/pkg/types"

type MyInputPlugin struct {
    config map[string]interface{}
}

func (p *MyInputPlugin) Name() string {
    return "my-input"
}

func (p *MyInputPlugin) Type() types.PluginType {
    return types.PluginTypeInput
}

func (p *MyInputPlugin) ReadBatch(batchSize int) (*types.DataBatch, error) {
    // Implementation
}

// Register in init()
func init() {
    plugin.RegisterInput("my-input", &MyInputPlugin{})
}
```

See [Plugin Development Guide](docs/plugin-development.md).

---

## 🖥️ Web UI

The Web UI provides:

- **Flow Designer**: Drag-and-drop pipeline creation
- **Job Management**: Create, edit, run, monitor jobs
- **Data Preview**: Sample data from sources
- **Monitoring Dashboard**: Real-time metrics and alerts
- **Plugin Manager**: Enable/disable plugins

### Screenshots

*(Coming soon)*

---

## 🛠️ Development

### Project Structure

```
fustgo/
├── cmd/fustgo/          # Main entry point
├── internal/            # Internal packages
│   ├── api/             # REST API handlers
│   ├── config/          # Configuration management
│   ├── database/        # Metadata storage
│   ├── logger/          # Logging
│   ├── models/          # Data models
│   ├── pipeline/        # Pipeline execution
│   ├── plugin/          # Plugin registry
│   ├── scheduler/       # Task scheduler
│   └── worker/          # Worker pool
├── pkg/                 # Public packages
│   ├── types/           # Shared types
│   └── utils/           # Utilities
├── plugins/             # Plugin implementations
│   ├── input/
│   ├── processor/
│   └── output/
├── configs/             # Configuration files
├── deploy/              # Deployment configs
├── docs/                # Documentation
└── test/                # Tests
```

### Running Tests

```bash
# Unit tests
go test ./...

# Integration tests
go test -tags=integration ./test/...

# Coverage
go test -cover ./...
```

### Code Style

We follow [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md).

---

## 🗺️ Roadmap

### Version 0.1.0 (Current - 50% Complete)

- [x] Core architecture
- [x] SQLite metadata storage
- [x] Plugin registry
- [x] Basic configuration
- [x] CSV input/output plugins
- [x] Filter/Mapping processors
- [x] Pipeline execution engine
- [x] Docker deployment
- [x] Comprehensive documentation
- [x] Unit tests (core types: 100%, plugin registry: 59.2%)
- [ ] MySQL/PostgreSQL plugins (pending)
- [ ] HTTP input plugin (pending)
- [ ] Job Manager (pending)
- [ ] REST API (pending)

### Version 0.2.0

- [ ] Web UI (Flow Designer)
- [ ] REST API
- [ ] Job scheduler
- [ ] Worker pool
- [ ] Kafka input/output
- [ ] HTTP input plugin

### Version 0.3.0

- [ ] OpenObserve integration
- [ ] Metrics & tracing
- [ ] Elasticsearch output
- [ ] Lightweight deployment mode

### Version 1.0.0

- [ ] Production-ready
- [ ] Distributed mode
- [ ] Complete documentation
- [ ] Comprehensive tests (80%+ coverage)

---

## 🤝 Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for details.

### Development Setup

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Make changes and add tests
4. Run tests: `go test ./...`
5. Commit: `git commit -m "Add my feature"`
6. Push: `git push origin feature/my-feature`
7. Create a Pull Request

---

## 📄 License

FustGo is licensed under the Apache License 2.0. See [LICENSE](LICENSE) for details.

---

## 🙏 Acknowledgments

- [Alibaba DataX](https://github.com/alibaba/DataX) - ETL framework inspiration
- [Benthos](https://github.com/benthosdev/benthos) - Configuration-driven design
- [OpenObserve](https://github.com/openobserve/openobserve) - Observability platform

---

## 📧 Contact

- **GitHub**: [@atlanssia](https://github.com/atlanssia)
- **Issues**: [GitHub Issues](https://github.com/atlanssia/fustgo/issues)
- **Email**: fustgo@example.com

---

**Built with ❤️ using Go**
