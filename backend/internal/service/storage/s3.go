package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// S3Config holds configuration for S3-compatible storage
type S3Config struct {
	Endpoint  string `json:"s3_endpoint"`
	Bucket    string `json:"s3_bucket"`
	Region    string `json:"s3_region"`
	AccessKey string `json:"s3_access_key"`
	SecretKey string `json:"s3_secret_key"`
}

// S3Driver implements Driver for S3-compatible storage
type S3Driver struct {
	config S3Config
	client *http.Client
}

func (d *S3Driver) Type() string { return "s3" }

func (d *S3Driver) Init(config []byte) error {
	if err := json.Unmarshal(config, &d.config); err != nil {
		return fmt.Errorf("invalid s3 config: %w", err)
	}
	if d.config.Endpoint == "" || d.config.Bucket == "" {
		return fmt.Errorf("s3 endpoint and bucket are required")
	}
	d.client = &http.Client{Timeout: 30 * time.Second}
	return nil
}

func (d *S3Driver) Test(ctx context.Context) (*HealthStatus, error) {
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, "HEAD", d.config.Endpoint, nil)
	if err != nil {
		return &HealthStatus{Healthy: false, Message: fmt.Sprintf("invalid endpoint: %v", err), LatencyMs: 0, CheckedAt: time.Now()}, nil
	}
	resp, err := d.client.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return &HealthStatus{Healthy: false, Message: fmt.Sprintf("endpoint not accessible: %v", err), LatencyMs: latency, CheckedAt: time.Now()}, nil
	}
	defer resp.Body.Close()
	return &HealthStatus{Healthy: true, Message: "s3 endpoint accessible", LatencyMs: latency, CheckedAt: time.Now()}, nil
}

func (d *S3Driver) Upload(ctx context.Context, path string, reader io.Reader, size int64) error {
	return fmt.Errorf("s3 upload: full S3 signing not yet implemented (requires AWS SDK or manual V4 signing)")
}

func (d *S3Driver) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("s3 download: full S3 signing not yet implemented")
}

func (d *S3Driver) Delete(ctx context.Context, path string) error {
	return fmt.Errorf("s3 delete: full S3 signing not yet implemented")
}

func (d *S3Driver) List(ctx context.Context, prefix string) ([]FileInfo, error) {
	return nil, fmt.Errorf("s3 list: full S3 signing not yet implemented")
}

func (d *S3Driver) Exists(ctx context.Context, path string) (bool, error) {
	return false, fmt.Errorf("s3 exists: full S3 signing not yet implemented")
}

func (d *S3Driver) GetCapacity(ctx context.Context) (*CapacityInfo, error) {
	return &CapacityInfo{
		Available: false,
		Message:   "S3 storage does not provide capacity information",
	}, nil
}
