package connectors

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// GitLabConnector handles cloning GitLab repositories.
type GitLabConnector struct{}

func NewGitLabConnector() *GitLabConnector {
	return &GitLabConnector{}
}

// Clone clones a GitLab repository to destDir (shallow, --depth=1).
// PAT is read from GITLAB_TOKEN env var per the security model.
func (g *GitLabConnector) Clone(ctx context.Context, repoURL, destDir string) error {
	cloneURL := injectToken(repoURL)

	cmd := exec.CommandContext(ctx, "git", "clone", "--depth=1", cloneURL, destDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone: %w", err)
	}

	return nil
}

// CloneFull clones a GitLab repository without --depth=1 (needed for git diff in incremental indexing).
func (g *GitLabConnector) CloneFull(ctx context.Context, repoURL, destDir string) error {
	cloneURL := injectToken(repoURL)

	cmd := exec.CommandContext(ctx, "git", "clone", cloneURL, destDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone (full): %w", err)
	}

	return nil
}

// ParseSourceConfig extracts useful config from a source's connection_uri.
func (g *GitLabConnector) ParseSourceConfig(connectionURI string) (repoURL, branch string) {
	// Format: https://gitlab.com/group/repo or https://gitlab.com/group/repo@branch
	parts := strings.SplitN(connectionURI, "@", 2)
	repoURL = parts[0]
	if len(parts) > 1 {
		branch = parts[1]
	} else {
		branch = "main"
	}
	return
}

// injectToken adds the GitLab PAT to the clone URL for authentication.
func injectToken(repoURL string) string {
	token := os.Getenv("GITLAB_TOKEN")
	if token == "" {
		return repoURL
	}

	// Transform https://gitlab.com/... to https://oauth2:TOKEN@gitlab.com/...
	if strings.HasPrefix(repoURL, "https://") {
		return "https://oauth2:" + token + "@" + strings.TrimPrefix(repoURL, "https://")
	}
	return repoURL
}
