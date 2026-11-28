package gitboard

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// RunGit executes git -C repoDir with non-interactive, C-locale environment settings.
func RunGit(ctx context.Context, repoDir string, args ...string) ([]byte, string, error) {
	return runGit(ctx, repoDir, nil, args...)
}

// RunGitWithStdin executes git -C repoDir with stdin supplied.
func RunGitWithStdin(ctx context.Context, repoDir string, stdin []byte, args ...string) ([]byte, string, error) {
	return runGit(ctx, repoDir, stdin, args...)
}

func runGit(ctx context.Context, repoDir string, stdin []byte, args ...string) ([]byte, string, error) {
	cmdArgs := make([]string, 0, len(args)+2)
	if repoDir != "" {
		cmdArgs = append(cmdArgs, "-C", repoDir)
	}
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.CommandContext(ctx, "git", cmdArgs...)
	cmd.Env = sanitizedGitEnv()
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	errText := strings.TrimSpace(stderr.String())
	if err != nil {
		if errText != "" {
			return stdout.Bytes(), errText, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, errText)
		}
		return stdout.Bytes(), errText, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return stdout.Bytes(), errText, nil
}

func sanitizedGitEnv() []string {
	env := os.Environ()
	out := make([]string, 0, len(env)+2)
	for _, entry := range env {
		if strings.HasPrefix(entry, "LC_ALL=") || strings.HasPrefix(entry, "GIT_TERMINAL_PROMPT=") {
			continue
		}
		out = append(out, entry)
	}
	out = append(out, "LC_ALL=C", "GIT_TERMINAL_PROMPT=0")
	return out
}
