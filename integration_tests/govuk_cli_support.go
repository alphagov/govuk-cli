package integration_tests

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
)

func cliCmd(ctx context.Context, args ...string) (*exec.Cmd, error) {
	cmd := exec.CommandContext(ctx, "../govuk-debug", args...)

	coverageDir, err := getCoverageDir()
	if err != nil {
		return nil, err
	}

	cmd.Env = append(cmd.Env, fmt.Sprintf("GOCOVERDIR=%s", coverageDir))

	return cmd, nil
}

func getCoverageDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	return filepath.Abs(filepath.Clean(path.Join(cwd, "..", "coverage", "integration")))
}
