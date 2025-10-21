# Contributing to FustGo DataX

Thank you for your interest in contributing to FustGo DataX! This document provides guidelines for contributing to the project.

## 🎯 Ways to Contribute

- **Report bugs**: File detailed bug reports with reproduction steps
- **Suggest features**: Propose new features or improvements
- **Write code**: Submit pull requests for bug fixes or new features
- **Improve documentation**: Enhance docs, tutorials, or examples
- **Review code**: Review pull requests from other contributors
- **Write plugins**: Create new input/processor/output plugins

## 🚀 Getting Started

### Prerequisites

- Go 1.23 or higher
- Docker and Docker Compose
- Git
- Basic understanding of ETL/ELT concepts

### Setting Up Development Environment

1. **Fork and clone the repository**:
   ```bash
   git clone https://github.com/YOUR_USERNAME/fustgo.git
   cd fustgo
   ```

2. **Install dependencies**:
   ```bash
   go mod download
   ```

3. **Build the project**:
   ```bash
   go build -o fustgo ./cmd/fustgo
   ```

4. **Run tests**:
   ```bash
   go test ./...
   ```

5. **Run the application**:
   ```bash
   ./fustgo --config configs/default.yaml
   ```

## 📝 Development Workflow

### 1. Create a Feature Branch

```bash
git checkout -b feature/your-feature-name
```

Branch naming conventions:
- `feature/` - New features
- `bugfix/` - Bug fixes
- `docs/` - Documentation changes
- `refactor/` - Code refactoring
- `test/` - Test additions/modifications

### 2. Make Your Changes

- Write clean, readable code
- Follow Go conventions and best practices
- Add tests for new functionality
- Update documentation as needed
- Keep commits atomic and well-described

### 3. Run Tests and Checks

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Check code formatting
go fmt ./...

# Run linter (if golangci-lint is installed)
golangci-lint run
```

### 4. Commit Your Changes

Follow conventional commit messages:

```
<type>(<scope>): <subject>

<body>

<footer>
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting)
- `refactor`: Code refactoring
- `test`: Test additions/changes
- `chore`: Build process or auxiliary tool changes

Example:
```
feat(plugin): add CSV input plugin

Implement CSV file reader plugin with support for:
- Custom delimiters
- Header detection
- Type inference
- Large file streaming

Closes #123
```

### 5. Push and Create Pull Request

```bash
git push origin feature/your-feature-name
```

Then create a pull request on GitHub.

## 🔌 Plugin Development Guide

### Creating a New Plugin

1. **Choose the plugin type** (Input, Processor, or Output)

2. **Create plugin directory**:
   ```bash
   mkdir -p plugins/input/myplugin
   ```

3. **Implement the plugin interface**:

```go
package myplugin

import "github.com/atlanssia/fustgo/pkg/types"

type MyPlugin struct {
    config map[string]interface{}
    // ... other fields
}

func (p *MyPlugin) Name() string {
    return "my-plugin"
}

func (p *MyPlugin) Type() types.PluginType {
    return types.PluginTypeInput
}

func (p *MyPlugin) Initialize(config map[string]interface{}) error {
    p.config = config
    // Initialization logic
    return nil
}

func (p *MyPlugin) Validate() error {
    // Validation logic
    return nil
}

func (p *MyPlugin) Close() error {
    // Cleanup logic
    return nil
}

func (p *MyPlugin) GetMetadata() types.PluginMetadata {
    return types.PluginMetadata{
        Name:        "my-plugin",
        Type:        types.PluginTypeInput,
        Version:     "1.0.0",
        Description: "My custom plugin",
    }
}

// Implement interface-specific methods
// For InputPlugin: Connect, ReadBatch, HasNext, GetProgress
```

4. **Register the plugin**:

```go
func init() {
    plugin.RegisterInput("my-plugin", &MyPlugin{})
}
```

5. **Write tests**:

```go
package myplugin_test

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestMyPlugin(t *testing.T) {
    // Your tests here
}
```

6. **Update documentation**:
   - Add plugin to README
   - Create plugin-specific docs in `docs/plugins/`

## 🧪 Testing Guidelines

### Unit Tests

- Write unit tests for all new code
- Aim for 80%+ code coverage
- Use table-driven tests where appropriate
- Mock external dependencies

Example:
```go
func TestFilterProcessor(t *testing.T) {
    tests := []struct {
        name      string
        condition string
        input     *types.DataBatch
        expected  int
    }{
        {
            name:      "filter by age",
            condition: "age > 18",
            input:     createTestBatch(),
            expected:  5,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### Integration Tests

- Test end-to-end workflows
- Use Docker containers for external services
- Tag with `//go:build integration`

### Running Tests

```bash
# Unit tests only
go test ./...

# Integration tests
go test -tags=integration ./test/...

# With coverage
go test -cover -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## 📚 Documentation Standards

### Code Documentation

- Add package documentation at the top of each package
- Document all exported types and functions
- Use godoc format

Example:
```go
// Package myplugin provides a custom data source plugin.
//
// This plugin reads data from X and supports Y features.
package myplugin

// MyPlugin implements the InputPlugin interface for reading from X.
type MyPlugin struct {
    // config holds the plugin configuration
    config map[string]interface{}
}

// ReadBatch reads a batch of records from the data source.
//
// The batchSize parameter controls the maximum number of records to read.
// Returns a DataBatch containing the records, or an error if reading fails.
func (p *MyPlugin) ReadBatch(batchSize int) (*types.DataBatch, error) {
    // Implementation
}
```

### User Documentation

- Update README.md for user-facing changes
- Add examples in `docs/examples/`
- Update configuration documentation

## 🎨 Code Style

### Go Conventions

- Follow [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)
- Use `gofmt` for formatting
- Use meaningful variable names
- Keep functions small and focused
- Avoid global variables

### Project-Specific Conventions

- Use structured logging (not fmt.Println)
- Handle all errors explicitly
- Use contexts for cancellation
- Prefer composition over inheritance
- Use interfaces for testability

### Example

```go
// Good
func (p *MyPlugin) ReadBatch(ctx context.Context, batchSize int) (*types.DataBatch, error) {
    logger.Info("reading batch of size %d", batchSize)
    
    records, err := p.source.Read(ctx, batchSize)
    if err != nil {
        return nil, fmt.Errorf("failed to read records: %w", err)
    }
    
    return &types.DataBatch{
        Records: records,
        // ... other fields
    }, nil
}

// Bad
func (p *MyPlugin) ReadBatch(batchSize int) *types.DataBatch {
    fmt.Println("reading")
    records, _ := p.source.Read(batchSize) // Ignoring error
    return &types.DataBatch{Records: records}
}
```

## 🔍 Pull Request Process

### Before Submitting

- [ ] Code compiles without errors
- [ ] All tests pass
- [ ] Code is formatted (`go fmt`)
- [ ] No linter warnings
- [ ] Documentation updated
- [ ] Commits are squashed and well-described

### PR Checklist

Your pull request should include:

1. **Clear description** of what the PR does
2. **Issue reference** if fixing a bug or implementing a feature request
3. **Test coverage** for new functionality
4. **Documentation updates** for user-facing changes
5. **Breaking change notes** if applicable

### Review Process

1. Automated checks must pass (tests, linting)
2. At least one maintainer approval required
3. Address review comments
4. Maintain a clean commit history

### After Approval

- Maintainers will merge your PR
- Your changes will be included in the next release

## 🐛 Reporting Bugs

### Before Reporting

1. Check existing issues to avoid duplicates
2. Try to reproduce with the latest version
3. Gather relevant information

### Bug Report Template

```markdown
**Describe the bug**
A clear description of what the bug is.

**To Reproduce**
Steps to reproduce the behavior:
1. Configure job with '...'
2. Run '....'
3. See error

**Expected behavior**
What you expected to happen.

**Actual behavior**
What actually happened.

**Environment:**
- FustGo version: [e.g., 0.1.0]
- OS: [e.g., Ubuntu 22.04]
- Go version: [e.g., 1.23.0]
- Deployment mode: [standalone/lightweight/distributed]

**Logs**
```
Paste relevant logs here
```

**Additional context**
Any other relevant information.
```

## 💡 Feature Requests

### Suggesting Features

1. Check if feature already requested
2. Explain the use case
3. Describe the proposed solution
4. Consider alternative solutions

### Feature Request Template

```markdown
**Is your feature request related to a problem?**
A clear description of the problem.

**Describe the solution you'd like**
What you want to happen.

**Describe alternatives you've considered**
Other approaches you've thought about.

**Additional context**
Any other relevant information.
```

## 📞 Getting Help

- **GitHub Issues**: For bug reports and feature requests
- **Discussions**: For questions and general discussion
- **Email**: fustgo@example.com for private inquiries

## 📄 License

By contributing to FustGo DataX, you agree that your contributions will be licensed under the Apache License 2.0.

---

Thank you for contributing to FustGo DataX! 🎉
