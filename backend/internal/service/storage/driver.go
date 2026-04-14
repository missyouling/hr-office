package storage

import (
	"context"
	"io"
	"time"
)

// FileInfo represents metadata about a stored file
type FileInfo struct {
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	Size         int64     `json:"size"`
	IsDir        bool      `json:"is_dir"`
	LastModified time.Time `json:"last_modified"`
	ContentType  string    `json:"content_type,omitempty"`
}

// HealthStatus represents the health state of a storage driver
type HealthStatus struct {
	Healthy   bool      `json:"healthy"`
	Message   string    `json:"message"`
	LatencyMs int64     `json:"latency_ms"`
	CheckedAt time.Time `json:"checked_at"`
}

// CapacityInfo represents storage capacity information
type CapacityInfo struct {
	TotalBytes   int64   `json:"total_bytes"`
	UsedBytes    int64   `json:"used_bytes"`
	FreeBytes    int64   `json:"free_bytes"`
	UsagePercent float64 `json:"usage_percent"`
	Available    bool    `json:"available"` // whether capacity info is available
	Message      string  `json:"message,omitempty"`
}

// CapacityProvider is an optional interface for drivers that support capacity reporting
type CapacityProvider interface {
	GetCapacity(ctx context.Context) (*CapacityInfo, error)
}

// Driver is the interface that all storage backends must implement
type Driver interface {
	// Type returns the storage type identifier (e.g., "local", "s3", "webdav")
	Type() string

	// Init initializes the driver with the given JSON configuration
	Init(config []byte) error

	// Test tests the connection to the storage backend
	Test(ctx context.Context) (*HealthStatus, error)

	// Upload stores data at the given path
	Upload(ctx context.Context, path string, reader io.Reader, size int64) error

	// Download retrieves data from the given path
	Download(ctx context.Context, path string) (io.ReadCloser, error)

	// Delete removes the file at the given path
	Delete(ctx context.Context, path string) error

	// List returns file info for items under the given prefix
	List(ctx context.Context, prefix string) ([]FileInfo, error)

	// Exists checks if a file exists at the given path
	Exists(ctx context.Context, path string) (bool, error)
}
