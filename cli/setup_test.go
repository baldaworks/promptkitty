package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSetupTargetsInstallPlugins(t *testing.T) {
	tests := []struct {
		target string
		want   [][]string
	}{
		{
			target: "codex",
			want: [][]string{
				{"codex", "plugin", "marketplace", "remove", "promptkitty"},
				{"codex", "plugin", "marketplace", "add", "baldaworks/promptkitty"},
				{"codex", "plugin", "add", "promptkitty@promptkitty"},
			},
		},
		{
			target: "claude",
			want: [][]string{
				{"claude", "plugin", "marketplace", "add", "baldaworks/promptkitty"},
				{"claude", "plugin", "install", "promptkitty@promptkitty", "--scope", "project"},
			},
		},
		{
			target: "grok",
			want: [][]string{
				{"grok", "plugin", "marketplace", "add", "baldaworks/promptkitty"},
				{"grok", "plugin", "install", "promptkitty@promptkitty", "--trust"},
			},
		},
		{
			target: "copilot",
			want: [][]string{
				{"copilot", "plugin", "marketplace", "add", "baldaworks/promptkitty"},
				{"copilot", "plugin", "install", "promptkitty@promptkitty"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.target, func(t *testing.T) {
			original := runSetupCommand
			t.Cleanup(func() { runSetupCommand = original })
			var got [][]string
			runSetupCommand = func(_ context.Context, _, _ io.Writer, name string, args ...string) error {
				got = append(got, append([]string{name}, args...))
				return nil
			}

			cmd := NewCommand(Options{})
			cmd.SetArgs([]string{"setup", test.target})
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("commands = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestPrepareCodexMarketplaceIgnoresMissingRegistration(t *testing.T) {
	original := runSetupCommand
	t.Cleanup(func() { runSetupCommand = original })
	runSetupCommand = func(_ context.Context, _, stderr io.Writer, _ string, _ ...string) error {
		_, _ = io.WriteString(stderr, "marketplace is not configured or installed")
		return errors.New("exit status 1")
	}

	var stderr bytes.Buffer
	if err := prepareCodexMarketplace(context.Background(), &stderr); err != nil {
		t.Fatal(err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestOpenCodeSetupInstallsPreservesAndForcesAssets(t *testing.T) {
	t.Chdir(t.TempDir())
	path := filepath.FromSlash(".opencode/commands/promptkitty.md")
	if err := os.MkdirAll(filepath.Dir(path), setupDirMode); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("custom"), setupFileMode); err != nil {
		t.Fatal(err)
	}

	result, err := writeOpenCodeIntegration(false)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.created) != 1 || len(result.unchanged) != 1 {
		t.Fatalf("install result = %#v", result)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "custom" {
		t.Fatalf("preserved command = %q", got)
	}

	result, err = writeOpenCodeIntegration(true)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.created) != len(openCodeAssetFiles) || len(result.unchanged) != 0 {
		t.Fatalf("forced install result = %#v", result)
	}
	want, err := openCodeAssets.ReadFile("assets/opencode/commands/promptkitty.md")
	if err != nil {
		t.Fatal(err)
	}
	got, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("forced command = %q, want %q", got, want)
	}
}

func TestOpenCodeSkillMatchesPluginAndCommand(t *testing.T) {
	embedded, err := openCodeAssets.ReadFile("assets/opencode/skills/promptkitty-assemble/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	plugin, err := os.ReadFile(filepath.Join("..", "plugins", "promptkitty", "prefixed-skills", "promptkitty-assemble", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(embedded, plugin) {
		t.Fatal("embedded OpenCode skill differs from the prefixed plugin skill")
	}
	command, err := openCodeAssets.ReadFile("assets/opencode/commands/promptkitty.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Load the `promptkitty-assemble` skill", "$ARGUMENTS"} {
		if !strings.Contains(string(command), want) {
			t.Errorf("OpenCode command is missing %q", want)
		}
	}
}

func TestSetupRejectsUnknownTarget(t *testing.T) {
	cmd := NewCommand(Options{})
	cmd.SetArgs([]string{"setup", "other"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "unsupported setup target") {
		t.Fatalf("setup error = %v", err)
	}
}
