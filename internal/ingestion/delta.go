package ingestion

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// DeltaResult holds the result of computing a git diff between two commits.
type DeltaResult struct {
	ChangedFiles  []string // modified/added relative paths
	DeletedFiles  []string // deleted relative paths
	PreviousSHA   string
	CurrentSHA    string
	IsIncremental bool
}

// ComputeGitDelta runs git diff --name-status between previousSHA and HEAD,
// returning the changed and deleted files.
func ComputeGitDelta(ctx context.Context, workDir, previousSHA string) (*DeltaResult, error) {
	// Get current HEAD SHA
	headCmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	headCmd.Dir = workDir
	headOut, err := headCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git rev-parse HEAD: %w", err)
	}
	currentSHA := strings.TrimSpace(string(headOut))

	if previousSHA == currentSHA {
		return &DeltaResult{
			PreviousSHA:   previousSHA,
			CurrentSHA:    currentSHA,
			IsIncremental: true,
		}, nil
	}

	// Run git diff --name-status
	diffCmd := exec.CommandContext(ctx, "git", "diff", "--name-status", previousSHA+"..HEAD")
	diffCmd.Dir = workDir
	diffOut, err := diffCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff: %w", err)
	}

	result := &DeltaResult{
		PreviousSHA:   previousSHA,
		CurrentSHA:    currentSHA,
		IsIncremental: true,
	}

	scanner := bufio.NewScanner(strings.NewReader(string(diffOut)))
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) < 2 {
			continue
		}
		status := line[0]
		// Tab-separated: status\tpath (or status\told\tnew for renames)
		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			continue
		}
		path := parts[1]

		switch status {
		case 'A', 'M', 'C':
			result.ChangedFiles = append(result.ChangedFiles, path)
		case 'D':
			result.DeletedFiles = append(result.DeletedFiles, path)
		case 'R':
			// Rename: old path deleted, new path added
			result.DeletedFiles = append(result.DeletedFiles, path)
			if len(parts) >= 3 {
				result.ChangedFiles = append(result.ChangedFiles, parts[2])
			}
		}
	}

	return result, nil
}
