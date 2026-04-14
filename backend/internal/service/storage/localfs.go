package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// LocalConfig holds configuration for local filesystem storage
type LocalConfig struct {
	RootPath string `json:"root_path"`
}

// LocalDriver implements Driver for local filesystem
type LocalDriver struct {
	config LocalConfig
}

func (d *LocalDriver) Type() string { return "local" }

func (d *LocalDriver) Init(config []byte) error {
	if err := json.Unmarshal(config, &d.config); err != nil {
		return fmt.Errorf("invalid local config: %w", err)
	}
	if d.config.RootPath == "" {
		d.config.RootPath = "./data/uploads"
	}
	return os.MkdirAll(d.config.RootPath, 0755)
}

func (d *LocalDriver) Test(ctx context.Context) (*HealthStatus, error) {
	start := time.Now()
	info, err := os.Stat(d.config.RootPath)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return &HealthStatus{Healthy: false, Message: fmt.Sprintf("path not accessible: %v", err), LatencyMs: latency, CheckedAt: time.Now()}, nil
	}
	if !info.IsDir() {
		return &HealthStatus{Healthy: false, Message: "path is not a directory", LatencyMs: latency, CheckedAt: time.Now()}, nil
	}
	return &HealthStatus{Healthy: true, Message: "local storage accessible", LatencyMs: latency, CheckedAt: time.Now()}, nil
}

func (d *LocalDriver) Upload(ctx context.Context, path string, reader io.Reader, size int64) error {
	fullPath := filepath.Join(d.config.RootPath, path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}
	f, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, reader)
	return err
}

func (d *LocalDriver) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	fullPath := filepath.Join(d.config.RootPath, path)
	return os.Open(fullPath)
}

func (d *LocalDriver) Delete(ctx context.Context, path string) error {
	fullPath := filepath.Join(d.config.RootPath, path)
	return os.Remove(fullPath)
}

func (d *LocalDriver) List(ctx context.Context, prefix string) ([]FileInfo, error) {
	fullPath := filepath.Join(d.config.RootPath, prefix)
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, err
	}
	var files []FileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, FileInfo{
			Name:         entry.Name(),
			Path:         filepath.Join(prefix, entry.Name()),
			Size:         info.Size(),
			IsDir:        entry.IsDir(),
			LastModified: info.ModTime(),
		})
	}
	return files, nil
}

func (d *LocalDriver) Exists(ctx context.Context, path string) (bool, error) {
	fullPath := filepath.Join(d.config.RootPath, path)
	_, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

// GetCapacity returns disk capacity for the local storage path
func (d *LocalDriver) GetCapacity(ctx context.Context) (*CapacityInfo, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(d.config.RootPath, &stat); err != nil {
		return &CapacityInfo{Available: false, Message: fmt.Sprintf("failed to get disk stats: %v", err)}, nil
	}
	total := int64(stat.Blocks) * int64(stat.Bsize)
	free := int64(stat.Bavail) * int64(stat.Bsize)
	used := total - free
	var pct float64
	if total > 0 {
		pct = float64(used) / float64(total) * 100
	}
	return &CapacityInfo{
		TotalBytes:   total,
		UsedBytes:    used,
		FreeBytes:    free,
		UsagePercent: pct,
		Available:    true,
	}, nil
}
