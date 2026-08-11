package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

// WebDAVConfig holds configuration for WebDAV storage
type WebDAVConfig struct {
	URL       string `json:"webdav_url"`
	Username  string `json:"webdav_username"`
	Password  string `json:"webdav_password"`
	Directory string `json:"directory"`
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
	// SSRF 防护：校验 endpoint 不指向内网地址
	if err := ValidateEndpoint(d.config.URL); err != nil {
		log.Printf("[WebDAVDriver] SSRF check failed for URL %s: %v", d.config.URL, err)
		return fmt.Errorf("endpoint rejected by SSRF policy: %w", err)
	}
	d.client = &http.Client{Timeout: 30 * time.Second}
	return nil
}

func (d *WebDAVDriver) Test(ctx context.Context) (*HealthStatus, error) {
	start := time.Now()
	baseURL := d.config.URL
	if !strings.HasSuffix(baseURL, "/dav") && !strings.HasSuffix(baseURL, "/dav/") {
		if strings.HasSuffix(baseURL, "/") {
			baseURL = baseURL + "dav"
		} else {
			baseURL = baseURL + "/dav"
		}
	}
	req, err := http.NewRequestWithContext(ctx, "PROPFIND", baseURL, nil)
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

func (d *WebDAVDriver) buildURL(filePath string) string {
	baseURL := d.config.URL

	if !strings.HasSuffix(baseURL, "/dav") && !strings.HasSuffix(baseURL, "/dav/") {
		if strings.HasSuffix(baseURL, "/") {
			baseURL = baseURL + "dav"
		} else {
			baseURL = baseURL + "/dav"
		}
	}

	dir := strings.Trim(d.config.Directory, "/")
	if dir != "" {
		baseURL = baseURL + "/" + dir
	}

	if filePath != "" {
		filePath = strings.Trim(filePath, "/")
		baseURL = baseURL + "/" + filePath
	}
	return baseURL
}

func (d *WebDAVDriver) Upload(ctx context.Context, filePath string, reader io.Reader, size int64) error {
	// 路径遍历防护
	if err := ValidateStoragePath(filePath); err != nil {
		return err
	}
	url := d.buildURL(filePath)
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

func (d *WebDAVDriver) Download(ctx context.Context, filePath string) (io.ReadCloser, error) {
	// 路径遍历防护
	if err := ValidateStoragePath(filePath); err != nil {
		return nil, err
	}
	url := d.buildURL(filePath)
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

func (d *WebDAVDriver) Delete(ctx context.Context, filePath string) error {
	// 路径遍历防护
	if err := ValidateStoragePath(filePath); err != nil {
		return err
	}
	url := d.buildURL(filePath)
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

// XML structures for parsing WebDAV PROPFIND response
type multiStatus struct {
	XMLName   xml.Name   `xml:"multistatus"`
	Responses []response `xml:"response"`
}

type response struct {
	Href     string   `xml:"href"`
	PropStat propStat `xml:"propstat"`
}

type propStat struct {
	Prop   prop   `xml:"prop"`
	Status string `xml:"status"`
}

type prop struct {
	DisplayName   string       `xml:"displayname"`
	ContentLength int64        `xml:"getcontentlength"`
	LastModified  string       `xml:"getlastmodified"`
	ContentType   string       `xml:"getcontenttype"`
	ResourceType  resourceType `xml:"resourcetype"`
}

type resourceType struct {
	Collection *struct{} `xml:"collection"`
}

func (d *WebDAVDriver) List(ctx context.Context, prefix string) ([]FileInfo, error) {
	listURL := d.buildURL(prefix)

	// Create PROPFIND request body
	propfindBody := `<?xml version="1.0" encoding="utf-8"?>
<D:propfind xmlns:D="DAV:">
  <D:prop>
    <D:displayname/>
    <D:getcontentlength/>
    <D:getlastmodified/>
    <D:getcontenttype/>
    <D:resourcetype/>
  </D:prop>
</D:propfind>`

	req, err := http.NewRequestWithContext(ctx, "PROPFIND", listURL, bytes.NewBufferString(propfindBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create PROPFIND request: %w", err)
	}

	// Set required headers
	req.Header.Set("Depth", "1")
	req.Header.Set("Content-Type", "application/xml; charset=utf-8")

	// Set basic auth if configured
	if d.config.Username != "" {
		req.SetBasicAuth(d.config.Username, d.config.Password)
	}

	// Execute the request
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("PROPFIND request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		log.Printf("[WebDAV List] Rate limited (429) at URL: %s\n", listURL)
		return []FileInfo{}, nil
	}

	if resp.StatusCode != http.StatusMultiStatus && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("[WebDAV List] Failed PROPFIND - Status: %d, URL: %s, Auth: %v, Response: %s\n",
			resp.StatusCode, listURL, d.config.Username != "", truncateString(string(body), 300))
		return nil, fmt.Errorf("PROPFIND returned status %d: %s", resp.StatusCode, string(body))
	}

	var ms multiStatus
	bodyBytes, _ := io.ReadAll(resp.Body)
	if err := xml.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&ms); err != nil {
		log.Printf("[WebDAV List] XML parse error at URL: %s, Error: %v, Response: %s\n",
			listURL, err, truncateString(string(bodyBytes), 300))
		return []FileInfo{}, nil
	}

	// Convert responses to FileInfo
	var files []FileInfo
	for i, r := range ms.Responses {
		// Skip the first response (it's the directory itself)
		if i == 0 {
			continue
		}

		// Parse the href to get the filename
		decodedHref, err := url.QueryUnescape(r.Href)
		if err != nil {
			decodedHref = r.Href
		}

		// Extract filename from path
		filename := path.Base(decodedHref)
		if filename == "" || filename == "/" {
			continue
		}

		// Parse last modified date (RFC1123 format)
		var lastModified time.Time
		if r.PropStat.Prop.LastModified != "" {
			// Try RFC1123 format first
			if t, err := time.Parse(time.RFC1123, r.PropStat.Prop.LastModified); err == nil {
				lastModified = t
			} else {
				// Fallback to current time if parsing fails
				lastModified = time.Now()
			}
		}

		// Determine if it's a directory
		isDir := r.PropStat.Prop.ResourceType.Collection != nil

		cleanPath := decodedHref
		if idx := strings.Index(cleanPath, "/dav/"); idx >= 0 {
			cleanPath = cleanPath[idx+len("/dav"):]
		}
		cleanPath = strings.TrimPrefix(cleanPath, "/")

		fileInfo := FileInfo{
			Name:         filename,
			Path:         cleanPath,
			Size:         r.PropStat.Prop.ContentLength,
			IsDir:        isDir,
			LastModified: lastModified,
			ContentType:  r.PropStat.Prop.ContentType,
		}

		files = append(files, fileInfo)
	}

	return files, nil
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

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
