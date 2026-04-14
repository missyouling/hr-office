package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OAuthConfig holds configuration for OAuth-based cloud drives
type OAuthConfig struct {
	Provider     string `json:"provider"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	RootFolderID string `json:"root_folder_id"`
	APIURL       string `json:"api_url"`
}

// OAuthDriver implements Driver for OAuth-based cloud storage
type OAuthDriver struct {
	config   OAuthConfig
	client   *http.Client
	provider string
}

// NewOAuthDriver creates a driver for a specific provider
func NewOAuthDriver(provider string) *OAuthDriver {
	return &OAuthDriver{provider: provider}
}

func (d *OAuthDriver) Type() string { return d.provider }

func (d *OAuthDriver) Init(config []byte) error {
	if err := json.Unmarshal(config, &d.config); err != nil {
		return fmt.Errorf("invalid oauth config: %w", err)
	}
	d.config.Provider = d.provider
	d.client = &http.Client{Timeout: 30 * time.Second}
	return nil
}

func (d *OAuthDriver) Test(ctx context.Context) (*HealthStatus, error) {
	start := time.Now()
	if d.config.AccessToken == "" {
		return &HealthStatus{
			Healthy:   false,
			Message:   fmt.Sprintf("%s: no access token configured", d.provider),
			LatencyMs: 0,
			CheckedAt: time.Now(),
		}, nil
	}
	apiURL := d.getAPIURL()
	if apiURL == "" {
		return &HealthStatus{
			Healthy:   true,
			Message:   fmt.Sprintf("%s: credentials configured (API URL not set for validation)", d.provider),
			LatencyMs: time.Since(start).Milliseconds(),
			CheckedAt: time.Now(),
		}, nil
	}
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return &HealthStatus{Healthy: false, Message: fmt.Sprintf("invalid api url: %v", err), LatencyMs: 0, CheckedAt: time.Now()}, nil
	}
	req.Header.Set("Authorization", "Bearer "+d.config.AccessToken)
	resp, err := d.client.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return &HealthStatus{Healthy: false, Message: fmt.Sprintf("%s not accessible: %v", d.provider, err), LatencyMs: latency, CheckedAt: time.Now()}, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return &HealthStatus{Healthy: true, Message: fmt.Sprintf("%s connected", d.provider), LatencyMs: latency, CheckedAt: time.Now()}, nil
	}
	return &HealthStatus{Healthy: false, Message: fmt.Sprintf("%s returned status %d", d.provider, resp.StatusCode), LatencyMs: latency, CheckedAt: time.Now()}, nil
}

func (d *OAuthDriver) getAPIURL() string {
	if d.config.APIURL != "" {
		return d.config.APIURL
	}
	switch d.provider {
	case "google_drive":
		return "https://www.googleapis.com/drive/v3/about?fields=user"
	case "onedrive":
		return "https://graph.microsoft.com/v1.0/me/drive"
	default:
		return ""
	}
}

func (d *OAuthDriver) Upload(ctx context.Context, path string, reader io.Reader, size int64) error {
	return fmt.Errorf("%s upload: not yet implemented (requires provider-specific API)", d.provider)
}

func (d *OAuthDriver) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("%s download: not yet implemented", d.provider)
}

func (d *OAuthDriver) Delete(ctx context.Context, path string) error {
	return fmt.Errorf("%s delete: not yet implemented", d.provider)
}

func (d *OAuthDriver) List(ctx context.Context, prefix string) ([]FileInfo, error) {
	return nil, fmt.Errorf("%s list: not yet implemented", d.provider)
}

func (d *OAuthDriver) Exists(ctx context.Context, path string) (bool, error) {
	return false, fmt.Errorf("%s exists: not yet implemented", d.provider)
}
