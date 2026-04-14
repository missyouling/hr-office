package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// WebDAVConfig holds configuration for WebDAV storage
type WebDAVConfig struct {
	URL      string `json:"webdav_url"`
	Username string `json:"webdav_username"`
	Password string `json:"webdav_password"`
}

// WebDAVDriver implements Driver for WebDAV storage
type WebDAVDriver struct {
	config WebDAVConfig
	client *http.Client
}

func (d *WebDAVDriver) Type() string { return "webdav" }

func (d *WebDAVDriver) Init(config []byte) error {
	if err := json.Unmarshal(config, &d.config); err != nil {
		return fmt.Errorf("invalid webdav config: %w", err)
	}
	if d.config.URL == "" {
		return fmt.Errorf("webdav url is required")
	}
	d.client = &http.Client{Timeout: 30 * time.Second}
	return nil
}

func (d *WebDAVDriver) Test(ctx context.Context) (*HealthStatus, error) {
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, "PROPFIND", d.config.URL, nil)
	if err != nil {
		return &HealthStatus{Healthy: false, Message: fmt.Sprintf("invalid url: %v", err), LatencyMs: 0, CheckedAt: time.Now()}, nil
	}
	if d.config.Username != "" {
		req.SetBasicAuth(d.config.Username, d.config.Password)
	}
	req.Header.Set("Depth", "0")
	resp, err := d.client.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return &HealthStatus{Healthy: false, Message: fmt.Sprintf("webdav not accessible: %v", err), LatencyMs: latency, CheckedAt: time.Now()}, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return &HealthStatus{Healthy: true, Message: "webdav accessible", LatencyMs: latency, CheckedAt: time.Now()}, nil
	}
	return &HealthStatus{Healthy: false, Message: fmt.Sprintf("webdav returned status %d", resp.StatusCode), LatencyMs: latency, CheckedAt: time.Now()}, nil
}

func (d *WebDAVDriver) Upload(ctx context.Context, path string, reader io.Reader, size int64) error {
	url := d.config.URL + "/" + path
	req, err := http.NewRequestWithContext(ctx, "PUT", url, reader)
	if err != nil {
		return err
	}
	if d.config.Username != "" {
		req.SetBasicAuth(d.config.Username, d.config.Password)
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	return fmt.Errorf("webdav upload failed with status %d", resp.StatusCode)
}

func (d *WebDAVDriver) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	url := d.config.URL + "/" + path
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	if d.config.Username != "" {
		req.SetBasicAuth(d.config.Username, d.config.Password)
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("webdav download failed with status %d", resp.StatusCode)
	}
	return resp.Body, nil
}

func (d *WebDAVDriver) Delete(ctx context.Context, path string) error {
	url := d.config.URL + "/" + path
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}
	if d.config.Username != "" {
		req.SetBasicAuth(d.config.Username, d.config.Password)
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	return fmt.Errorf("webdav delete failed with status %d", resp.StatusCode)
}

func (d *WebDAVDriver) List(ctx context.Context, prefix string) ([]FileInfo, error) {
	return nil, fmt.Errorf("webdav list: PROPFIND parsing not yet implemented")
}

func (d *WebDAVDriver) Exists(ctx context.Context, path string) (bool, error) {
	url := d.config.URL + "/" + path
	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return false, err
	}
	if d.config.Username != "" {
		req.SetBasicAuth(d.config.Username, d.config.Password)
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return false, nil
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK, nil
}

func (d *WebDAVDriver) GetCapacity(ctx context.Context) (*CapacityInfo, error) {
	return &CapacityInfo{
		Available: false,
		Message:   "WebDAV capacity query not yet implemented",
	}, nil
}
