package storage

import (
	"encoding/json"
	"fmt"
	"sync"
)

// DriverFactory is a function that creates a new Driver instance
type DriverFactory func() Driver

// Registry manages available storage driver types
type Registry struct {
	mu        sync.RWMutex
	factories map[string]DriverFactory
}

// NewRegistry creates a new driver registry
func NewRegistry() *Registry {
	return &Registry{
		factories: make(map[string]DriverFactory),
	}
}

// Register adds a driver factory for the given type
func (r *Registry) Register(storageType string, factory DriverFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[storageType] = factory
}

// Create creates a new driver instance for the given type and initializes it with config
func (r *Registry) Create(storageType string, config json.RawMessage) (Driver, error) {
	r.mu.RLock()
	factory, ok := r.factories[storageType]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unsupported storage type: %s", storageType)
	}

	driver := factory()
	if err := driver.Init(config); err != nil {
		return nil, fmt.Errorf("failed to initialize %s driver: %w", storageType, err)
	}

	return driver, nil
}

// SupportedTypes returns a list of registered storage types
func (r *Registry) SupportedTypes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]string, 0, len(r.factories))
	for t := range r.factories {
		types = append(types, t)
	}
	return types
}

// DefaultRegistry is the global driver registry
var DefaultRegistry = NewRegistry()
