package connectors

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	minioclient "github.com/maraichr/codegraph/internal/store/minio"
)

// ZipConnector handles ZIP file upload and extraction.
type ZipConnector struct {
	minio *minioclient.Client
}

func NewZipConnector(minio *minioclient.Client) *ZipConnector {
	return &ZipConnector{minio: minio}
}

// Upload streams the ZIP file to MinIO object storage.
func (z *ZipConnector) Upload(ctx context.Context, objectName string, reader io.Reader, size int64) error {
	return z.minio.UploadFile(ctx, objectName, reader, size)
}

// Extract downloads a ZIP from MinIO and extracts it to a local directory.
func (z *ZipConnector) Extract(ctx context.Context, objectName, destDir string) error {
	reader, err := z.minio.DownloadFile(ctx, objectName)
	if err != nil {
		return fmt.Errorf("download zip: %w", err)
	}
	defer reader.Close()

	// Write to temp file for zip.OpenReader
	tmpFile, err := os.CreateTemp("", "codegraph-zip-*.zip")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := io.Copy(tmpFile, reader); err != nil {
		return fmt.Errorf("copy to temp: %w", err)
	}
	tmpFile.Close()

	zr, err := zip.OpenReader(tmpFile.Name())
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer zr.Close()

	for _, f := range zr.File {
		target := filepath.Join(destDir, f.Name)

		// Prevent zip slip
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid zip entry: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(target, 0o755)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("mkdir: %w", err)
		}

		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("open zip entry: %w", err)
		}

		outFile, err := os.Create(target)
		if err != nil {
			rc.Close()
			return fmt.Errorf("create file: %w", err)
		}

		// Limit extraction size to prevent zip bombs (100MB per file)
		if _, err := io.Copy(outFile, io.LimitReader(rc, 100*1024*1024)); err != nil {
			outFile.Close()
			rc.Close()
			return fmt.Errorf("extract file: %w", err)
		}

		outFile.Close()
		rc.Close()
	}

	return nil
}
