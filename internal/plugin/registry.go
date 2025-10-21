package plugin

import (
	"fmt"
	"sync"

	"github.com/atlanssia/fustgo/pkg/types"
)

// Registry manages all registered plugins
type Registry struct {
	mu         sync.RWMutex
	inputs     map[string]types.InputPlugin
	processors map[string]types.ProcessorPlugin
	outputs    map[string]types.OutputPlugin
}

// Global registry instance
var globalRegistry = &Registry{
	inputs:     make(map[string]types.InputPlugin),
	processors: make(map[string]types.ProcessorPlugin),
	outputs:    make(map[string]types.OutputPlugin),
}

// GetRegistry returns the global plugin registry
func GetRegistry() *Registry {
	return globalRegistry
}

// RegisterInput registers an input plugin
func (r *Registry) RegisterInput(name string, plugin types.InputPlugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.inputs[name]; exists {
		return fmt.Errorf("input plugin already registered: %s", name)
	}

	r.inputs[name] = plugin
	return nil
}

// RegisterProcessor registers a processor plugin
func (r *Registry) RegisterProcessor(name string, plugin types.ProcessorPlugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.processors[name]; exists {
		return fmt.Errorf("processor plugin already registered: %s", name)
	}

	r.processors[name] = plugin
	return nil
}

// RegisterOutput registers an output plugin
func (r *Registry) RegisterOutput(name string, plugin types.OutputPlugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.outputs[name]; exists {
		return fmt.Errorf("output plugin already registered: %s", name)
	}

	r.outputs[name] = plugin
	return nil
}

// GetInput retrieves an input plugin by name
func (r *Registry) GetInput(name string) (types.InputPlugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, exists := r.inputs[name]
	if !exists {
		return nil, fmt.Errorf("input plugin not found: %s", name)
	}

	return plugin, nil
}

// GetProcessor retrieves a processor plugin by name
func (r *Registry) GetProcessor(name string) (types.ProcessorPlugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, exists := r.processors[name]
	if !exists {
		return nil, fmt.Errorf("processor plugin not found: %s", name)
	}

	return plugin, nil
}

// GetOutput retrieves an output plugin by name
func (r *Registry) GetOutput(name string) (types.OutputPlugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, exists := r.outputs[name]
	if !exists {
		return nil, fmt.Errorf("output plugin not found: %s", name)
	}

	return plugin, nil
}

// ListInputs returns all registered input plugins
func (r *Registry) ListInputs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.inputs))
	for name := range r.inputs {
		names = append(names, name)
	}
	return names
}

// ListProcessors returns all registered processor plugins
func (r *Registry) ListProcessors() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.processors))
	for name := range r.processors {
		names = append(names, name)
	}
	return names
}

// ListOutputs returns all registered output plugins
func (r *Registry) ListOutputs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.outputs))
	for name := range r.outputs {
		names = append(names, name)
	}
	return names
}

// GetPluginMetadata returns metadata for all plugins
func (r *Registry) GetPluginMetadata() []types.PluginMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var metadata []types.PluginMetadata

	for _, plugin := range r.inputs {
		metadata = append(metadata, plugin.GetMetadata())
	}

	for _, plugin := range r.processors {
		metadata = append(metadata, plugin.GetMetadata())
	}

	for _, plugin := range r.outputs {
		metadata = append(metadata, plugin.GetMetadata())
	}

	return metadata
}

// Convenience functions for global registry
func RegisterInput(name string, plugin types.InputPlugin) error {
	return globalRegistry.RegisterInput(name, plugin)
}

func RegisterProcessor(name string, plugin types.ProcessorPlugin) error {
	return globalRegistry.RegisterProcessor(name, plugin)
}

func RegisterOutput(name string, plugin types.OutputPlugin) error {
	return globalRegistry.RegisterOutput(name, plugin)
}

func GetInput(name string) (types.InputPlugin, error) {
	return globalRegistry.GetInput(name)
}

func GetProcessor(name string) (types.ProcessorPlugin, error) {
	return globalRegistry.GetProcessor(name)
}

func GetOutput(name string) (types.OutputPlugin, error) {
	return globalRegistry.GetOutput(name)
}
