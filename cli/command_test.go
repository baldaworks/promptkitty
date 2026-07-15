package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/baldaworks/promptkitty"
	promptkittycli "github.com/baldaworks/promptkitty/cli"
)

func TestVersionMatchesRelease(t *testing.T) {
	if got, want := promptkittycli.Version, "0.2.1"; got != want {
		t.Fatalf("Version = %q, want %q", got, want)
	}
}

func TestCatalogCommands(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    string
		notWant string
	}{
		{
			name:    "list defaults to templates",
			args:    []string{"list"},
			want:    "review-code",
			notWant: "systems-engineer",
		},
		{
			name: "list all component types",
			args: []string{"list", "--all"},
			want: "systems-engineer",
		},
		{
			name: "search with type filter",
			args: []string{"search", "code review", "--type", "template"},
			want: "review-code",
		},
		{
			name: "show component",
			args: []string{"show", "review-code"},
			want: "Type:         template",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stdout, _, err := execute(t, test.args...)
			if err != nil {
				t.Fatalf("Execute(%q) returned unexpected error: %v", test.args, err)
			}
			if !strings.Contains(stdout, test.want) {
				t.Errorf("Execute(%q) output does not contain %q:\n%s", test.args, test.want, stdout)
			}
			if test.notWant != "" && strings.Contains(stdout, test.notWant) {
				t.Errorf("Execute(%q) output unexpectedly contains %q:\n%s", test.args, test.notWant, stdout)
			}
		})
	}
}

func TestCatalogJSON(t *testing.T) {
	stdout, _, err := execute(t, "show", "review-code", "--json")
	if err != nil {
		t.Fatalf("Execute(show review-code --json) returned unexpected error: %v", err)
	}

	var detail promptkitty.ComponentDetail
	if err := json.Unmarshal([]byte(stdout), &detail); err != nil {
		t.Fatalf("json.Unmarshal(show output) returned unexpected error: %v\noutput: %s", err, stdout)
	}
	if got, want := detail.Name, "review-code"; got != want {
		t.Errorf("detail.Name = %q, want %q", got, want)
	}
}

func TestAssemble(t *testing.T) {
	dir := t.TempDir()
	codePath := filepath.Join(dir, "code.txt")
	code := "line one\nline two\n"
	if err := os.WriteFile(codePath, []byte(code), 0o600); err != nil {
		t.Fatalf("os.WriteFile(%q) returned unexpected error: %v", codePath, err)
	}

	args := []string{
		"assemble", "review-code",
		"--param-file", "code=" + codePath,
		"--param", "review_focus=correctness",
		"--param", "language=Go",
		"--param", "additional_protocols=",
		"--param", "context=CLI test",
	}
	stdout, _, err := execute(t, args...)
	if err != nil {
		t.Fatalf("Execute(%q) returned unexpected error: %v", args, err)
	}
	if !strings.Contains(stdout, code) {
		t.Errorf("Execute(%q) output does not preserve file parameter %q", args, code)
	}
	if !strings.HasSuffix(stdout, "\n") {
		t.Errorf("Execute(%q) output does not end in a newline", args)
	}
}

func TestAssembleJSON(t *testing.T) {
	args := []string{
		"assemble", "review-code", "--json",
		"-p", "code=package main",
		"-p", "review_focus=all",
		"-p", "language=Go",
		"-p", "additional_protocols=",
		"-p", "context=example",
	}
	stdout, _, err := execute(t, args...)
	if err != nil {
		t.Fatalf("Execute(%q) returned unexpected error: %v", args, err)
	}

	var result promptkitty.AssembleResult
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("json.Unmarshal(assemble output) returned unexpected error: %v\noutput: %s", err, stdout)
	}
	if got, want := result.Template.Name, "review-code"; got != want {
		t.Errorf("result.Template.Name = %q, want %q", got, want)
	}
	if !strings.Contains(result.Markdown, "package main") {
		t.Errorf("result.Markdown does not contain the supplied code")
	}
}

func TestAssembleOutputPolicy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "role.md")
	args := []string{
		"assemble", "review-code", "--output", path,
		"-p", "code=package main",
		"-p", "review_focus=all",
		"-p", "language=Go",
		"-p", "additional_protocols=",
		"-p", "context=example",
	}
	if _, _, err := execute(t, args...); err != nil {
		t.Fatalf("Execute(%q) returned unexpected error: %v", args, err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("os.Stat(%q) returned unexpected error: %v", path, err)
	}
	if _, _, err := execute(t, args...); err == nil {
		t.Errorf("Execute(%q) returned nil error for an existing output file", args)
	}
	if _, _, err := execute(t, append(args, "--force")...); err != nil {
		t.Errorf("Execute(%q with --force) returned unexpected error: %v", args, err)
	}
}

func TestAssembleRejectsDuplicateParameters(t *testing.T) {
	args := []string{"assemble", "review-code", "-p", "code=one", "-p", "code=two"}
	if _, _, err := execute(t, args...); err == nil {
		t.Errorf("Execute(%q) returned nil error for a duplicate parameter", args)
	}
}

func TestRunJSONError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := promptkittycli.Run(context.Background(), []string{"show", "missing", "--json"}, &stdout, &stderr)
	if got, want := code, 1; got != want {
		t.Errorf("Run(show missing --json) = %d, want %d", got, want)
	}
	if stdout.Len() != 0 {
		t.Errorf("Run(show missing --json) stdout = %q, want empty", stdout.String())
	}
	var diagnostic map[string]string
	if err := json.Unmarshal(stderr.Bytes(), &diagnostic); err != nil {
		t.Fatalf("json.Unmarshal(Run stderr) returned unexpected error: %v\nstderr: %s", err, stderr.String())
	}
	if diagnostic["error"] == "" {
		t.Errorf("Run(show missing --json) diagnostic has an empty error")
	}
}

func ExampleNewCommand() {
	cmd := promptkittycli.NewCommand(promptkittycli.Options{Use: "promptkit"})
	fmt.Println(cmd.Use)
	// Output:
	// promptkit
}

func execute(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	library, err := promptkitty.New()
	if err != nil {
		t.Fatalf("PromptKitty New() returned unexpected error: %v", err)
	}

	var stdoutBuffer, stderrBuffer bytes.Buffer
	cmd := promptkittycli.NewCommand(promptkittycli.Options{Library: library})
	cmd.SetArgs(args)
	cmd.SetOut(&stdoutBuffer)
	cmd.SetErr(&stderrBuffer)
	cmd.SetContext(t.Context())
	err = cmd.Execute()
	return stdoutBuffer.String(), stderrBuffer.String(), err
}
