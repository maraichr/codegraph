package connectors

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	appconfig "github.com/codegraph-labs/codegraph/internal/config"
)

// S3Connector downloads files from an S3-compatible bucket.
type S3Connector struct {
	client *s3.Client
	bucket string
}

// NewS3Connector creates a new S3 connector. Works with both AWS S3 and MinIO.
func NewS3Connector(cfg appconfig.S3Config) (*S3Connector, error) {
	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(cfg.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = &cfg.Endpoint
			o.UsePathStyle = true
		}
	})

	return &S3Connector{client: client, bucket: cfg.Bucket}, nil
}

// Sync downloads all objects under the given prefix to destDir.
func (c *S3Connector) Sync(ctx context.Context, prefix, destDir string) error {
	paginator := s3.NewListObjectsV2Paginator(c.client, &s3.ListObjectsV2Input{
		Bucket: &c.bucket,
		Prefix: &prefix,
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("list objects: %w", err)
		}

		for _, obj := range page.Contents {
			if obj.Key == nil {
				continue
			}
			key := *obj.Key

			// Skip "directory" markers
			if key[len(key)-1] == '/' {
				continue
			}

			localPath := filepath.Join(destDir, key)
			if err := c.downloadObject(ctx, key, localPath); err != nil {
				return fmt.Errorf("download %s: %w", key, err)
			}
		}
	}

	return nil
}

func (c *S3Connector) downloadObject(ctx context.Context, key, localPath string) error {
	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		return err
	}

	resp, err := c.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &c.bucket,
		Key:    &key,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	f, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return err
	}
	return nil
}
