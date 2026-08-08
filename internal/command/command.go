package command

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type Spec struct {
	Name string
	Args []string
}

func C(name string, args ...string) Spec {
	return Spec{Name: name, Args: args}
}

func (s Spec) String() string {
	parts := append([]string{s.Name}, s.Args...)
	for i, part := range parts {
		parts[i] = quote(part)
	}
	return strings.Join(parts, " ")
}

type Runner interface {
	Output(ctx context.Context, spec Spec) ([]byte, error)
}

type ExecRunner struct{}

func (ExecRunner) Output(ctx context.Context, spec Spec) ([]byte, error) {
	cmd := exec.CommandContext(ctx, spec.Name, spec.Args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			return output, fmt.Errorf("%s: %w", spec.String(), err)
		}
		return output, fmt.Errorf("%s: %w: %s", spec.String(), err, message)
	}
	return output, nil
}

func quote(value string) string {
	if value != "" && !strings.ContainsAny(value, " \t\r\n\"'") {
		return value
	}
	return `"` + strings.ReplaceAll(value, `"`, `\"`) + `"`
}
