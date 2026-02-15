package ingestion

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/maraichr/codegraph/internal/ingestion/connectors"
	"github.com/maraichr/codegraph/internal/store"
)

// CloneStage fetches source files (ZIP extract, git clone, or S3 sync) into a local work directory.
type CloneStage struct {
	store   *store.Store
	zipConn *connectors.ZipConnector
	gitConn *connectors.GitLabConnector
	s3Conn  *connectors.S3Connector
}

func NewCloneStage(s *store.Store, zipConn *connectors.ZipConnector, gitConn *connectors.GitLabConnector, s3Conn *connectors.S3Connector) *CloneStage {
	return &CloneStage{store: s, zipConn: zipConn, gitConn: gitConn, s3Conn: s3Conn}
}

func (s *CloneStage) Name() string { return "clone" }

func (s *CloneStage) Execute(ctx context.Context, rc *IndexRunContext) error {
	source, err := s.store.GetSource(ctx, rc.SourceID)
	if err != nil {
		return fmt.Errorf("get source: %w", err)
	}

	workDir := filepath.Join(os.TempDir(), "codegraph-ingest", rc.IndexRunID.String())
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return fmt.Errorf("create work dir: %w", err)
	}

	switch rc.SourceType {
	case "upload":
		var cfg map[string]string
		if err := json.Unmarshal(source.Config, &cfg); err != nil {
			return fmt.Errorf("parse source config: %w", err)
		}
		objectName := cfg["object_name"]
		if objectName == "" {
			return fmt.Errorf("source config missing object_name")
		}
		if err := s.zipConn.Extract(ctx, objectName, workDir); err != nil {
			return fmt.Errorf("extract zip: %w", err)
		}

	case "git":
		if source.ConnectionUri == nil || *source.ConnectionUri == "" {
			return fmt.Errorf("git source missing connection_uri")
		}

		// Check for incremental indexing
		previousSHA := ""
		if source.LastCommitSha != nil {
			previousSHA = *source.LastCommitSha
		}

		if previousSHA != "" {
			// Full clone needed for git diff
			if err := s.gitConn.CloneFull(ctx, *source.ConnectionUri, workDir); err != nil {
				return fmt.Errorf("git clone (full): %w", err)
			}

			delta, err := ComputeGitDelta(ctx, workDir, previousSHA)
			if err != nil {
				// Fall back to full re-index
				rc.Incremental = false
			} else {
				rc.Incremental = delta.IsIncremental
				rc.PreviousSHA = delta.PreviousSHA
				rc.CurrentSHA = delta.CurrentSHA
				rc.ChangedFiles = delta.ChangedFiles
				rc.DeletedFiles = delta.DeletedFiles
			}
		} else {
			// First index — shallow clone
			if err := s.gitConn.Clone(ctx, *source.ConnectionUri, workDir); err != nil {
				return fmt.Errorf("git clone: %w", err)
			}
			// Capture HEAD SHA for next incremental run
			rc.CurrentSHA = gitHeadSHA(ctx, workDir)
		}

	case "s3":
		if s.s3Conn == nil {
			return fmt.Errorf("S3 connector not configured")
		}
		var cfg map[string]string
		if err := json.Unmarshal(source.Config, &cfg); err != nil {
			return fmt.Errorf("parse source config: %w", err)
		}
		prefix := cfg["prefix"]
		if err := s.s3Conn.Sync(ctx, prefix, workDir); err != nil {
			return fmt.Errorf("s3 sync: %w", err)
		}

	default:
		return fmt.Errorf("unsupported source type: %s", rc.SourceType)
	}

	rc.WorkDir = workDir
	return nil
}

// gitHeadSHA reads the current HEAD SHA from a git repo.
func gitHeadSHA(ctx context.Context, workDir string) string {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	cmd.Dir = workDir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
