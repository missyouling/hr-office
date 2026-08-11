package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
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
	client *s3.Client
}

func (d *S3Driver) Type() string { return "s3" }

func (d *S3Driver) Init(configBytes []byte) error {
	if err := json.Unmarshal(configBytes, &d.config); err != nil {
		return fmt.Errorf("invalid s3 config: %w", err)
	}
	if d.config.Endpoint == "" || d.config.Bucket == "" {
		return fmt.Errorf("s3 endpoint and bucket are required")
	}

	// SSRF 防护：校验 endpoint 不指向内网地址
	if err := ValidateEndpoint(d.config.Endpoint); err != nil {
		log.Printf("[S3Driver] SSRF check failed for endpoint %s: %v", d.config.Endpoint, err)
		return fmt.Errorf("endpoint rejected by SSRF policy: %w", err)
	}

	if d.config.Region == "" {
		d.config.Region = "us-east-1"
	}

	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if service == s3.ServiceID {
			return aws.Endpoint{
				URL:           d.config.Endpoint,
				SigningRegion: d.config.Region,
			}, nil
		}
		return aws.Endpoint{}, fmt.Errorf("unknown service")
	})

	cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(d.config.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			d.config.AccessKey,
			d.config.SecretKey,
			"",
		)),
		awsconfig.WithEndpointResolverWithOptions(customResolver),
	)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	d.client = s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	return nil
}

func (d *S3Driver) Test(ctx context.Context) (*HealthStatus, error) {
	start := time.Now()
	_, err := d.client.ListBuckets(ctx, &s3.ListBucketsInput{})
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return &HealthStatus{
			Healthy:   false,
			Message:   fmt.Sprintf("s3 endpoint not accessible: %v", err),
			LatencyMs: latency,
			CheckedAt: time.Now(),
		}, nil
	}

	return &HealthStatus{
		Healthy:   true,
		Message:   "s3 endpoint accessible",
		LatencyMs: latency,
		CheckedAt: time.Now(),
	}, nil
}

func (d *S3Driver) Upload(ctx context.Context, path string, reader io.Reader, size int64) error {
	// 路径遍历防护
	if err := ValidateStoragePath(path); err != nil {
		return err
	}
	_, err := d.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(d.config.Bucket),
		Key:           aws.String(path),
		Body:          reader,
		ContentLength: aws.Int64(size),
	})
	if err != nil {
		return fmt.Errorf("s3 upload failed: %w", err)
	}
	return nil
}

func (d *S3Driver) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	// 路径遍历防护
	if err := ValidateStoragePath(path); err != nil {
		return nil, err
	}
	result, err := d.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(d.config.Bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		return nil, fmt.Errorf("s3 download failed: %w", err)
	}
	return result.Body, nil
}

func (d *S3Driver) Delete(ctx context.Context, path string) error {
	// 路径遍历防护
	if err := ValidateStoragePath(path); err != nil {
		return err
	}
	_, err := d.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(d.config.Bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		return fmt.Errorf("s3 delete failed: %w", err)
	}
	return nil
}

func (d *S3Driver) List(ctx context.Context, prefix string) ([]FileInfo, error) {
	paginator := s3.NewListObjectsV2Paginator(d.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(d.config.Bucket),
		Prefix: aws.String(prefix),
	})

	var files []FileInfo

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("s3 list failed: %w", err)
		}

		for _, obj := range page.Contents {
			files = append(files, FileInfo{
				Name:         *obj.Key,
				Path:         *obj.Key,
				Size:         *obj.Size,
				IsDir:        false,
				LastModified: *obj.LastModified,
			})
		}
	}

	return files, nil
}

func (d *S3Driver) Exists(ctx context.Context, path string) (bool, error) {
	_, err := d.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(d.config.Bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		if _, ok := err.(*types.NotFound); ok {
			return false, nil
		}
		return false, fmt.Errorf("s3 exists check failed: %w", err)
	}
	return true, nil
}

func (d *S3Driver) GetCapacity(ctx context.Context) (*CapacityInfo, error) {
	return &CapacityInfo{
		Available: false,
		Message:   "S3 storage does not provide capacity information",
	}, nil
}
