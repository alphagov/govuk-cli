package integration_tests

import (
	"bytes"
	"context"
	"os/exec"
	"strconv"
	"strings"
)

type CompletionResult struct {
	CobraCompletionDirectiveName string
	CompletionDirectiveCode      int
	Suggestions                  []string
	StdErrLines                  []string
}

func completionCliCmd(ctx context.Context, args ...string) (*exec.Cmd, error) {
	allArgs := append([]string{"--kubeconfig", kubeconfigPath, "__complete"}, args...)

	return cliCmd(ctx, allArgs...)
}

func getCompletionResult(cliCmd *exec.Cmd) (*CompletionResult, error) {
	var stdoutBuffer, stderrBuffer bytes.Buffer
	cliCmd.Stdout = &stdoutBuffer
	cliCmd.Stderr = &stderrBuffer

	err := cliCmd.Run()
	if err != nil {
		return nil, err
	}

	stdoutLines := strings.Split(stdoutBuffer.String(), "\n")
	stderrLines := strings.Split(stderrBuffer.String(), "\n")

	// There's always a blank line on the end of the stdout and stderr buffers, remove it
	stdoutLines = stdoutLines[0 : len(stdoutLines)-1]
	stderrLines = stderrLines[0 : len(stderrLines)-1]

	completionCode, err := parseCompletionDirectiveCode(stdoutLines)
	if err != nil {
		return nil, err
	}

	return &CompletionResult{
		CompletionDirectiveCode:      completionCode,
		CobraCompletionDirectiveName: parseCobraCompletionDirective(stderrLines),
		Suggestions:                  stdoutLines[0 : len(stdoutLines)-1],
		StdErrLines:                  stderrLines[0 : len(stderrLines)-1],
	}, nil
}

func parseCompletionDirectiveCode(stdoutLines []string) (int, error) {
	directiveLine := stdoutLines[len(stdoutLines)-1]

	trimmedLine := strings.TrimPrefix(directiveLine, ":")

	return strconv.Atoi(trimmedLine)
}

func parseCobraCompletionDirective(stderrLines []string) string {
	directiveLine := stderrLines[len(stderrLines)-1]

	words := strings.Split(directiveLine, " ")

	return words[len(words)-1]
}
