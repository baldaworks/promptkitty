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

func TestLocalSetupInstallsPreservesAndForcesAssets(t *testing.T) {
	tests := []struct {
		name      string
		assets    []setupAssetFile
		install   func(bool) (setupInstallResult, error)
		preserved string
	}{
		{
			name:      "opencode",
			assets:    openCodeAssetFiles,
			install:   writeOpenCodeIntegration,
			preserved: ".opencode/commands/promptkitty-assemble.md",
		},
		{
			name:      "cursor",
			assets:    cursorAssetFiles,
			install:   writeCursorIntegration,
			preserved: ".cursor/skills/promptkitty-assemble/SKILL.md",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			path := filepath.FromSlash(test.preserved)
			if err := os.MkdirAll(filepath.Dir(path), setupDirMode); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte("custom"), setupFileMode); err != nil {
				t.Fatal(err)
			}

			result, err := test.install(false)
			if err != nil {
				t.Fatal(err)
			}
			if len(result.created) != len(test.assets)-1 || len(result.unchanged) != 1 {
				t.Fatalf("install result = %#v", result)
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != "custom" {
				t.Fatalf("preserved file = %q", got)
			}

			result, err = test.install(true)
			if err != nil {
				t.Fatal(err)
			}
			if len(result.created) != len(test.assets) || len(result.unchanged) != 0 {
				t.Fatalf("forced install result = %#v", result)
			}
			asset := findSetupAsset(t, test.assets, test.preserved)
			want, err := setupAssets.ReadFile(asset.source)
			if err != nil {
				t.Fatal(err)
			}
			got, err = os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("forced file differs from %s", asset.source)
			}
		})
	}
}

func TestLocalSetupAssetsMatchPrefixedPluginSkills(t *testing.T) {
	pluginRoot := filepath.Join("..", "plugins", "promptkitty", "prefixed-skills")
	tests := []struct {
		asset  string
		plugin string
	}{
		{
			asset:  "skills/promptkitty-assemble/SKILL.md",
			plugin: "promptkitty-assemble/SKILL.md",
		},
		{
			asset:  "skills/promptkitty-author-agent-instructions/SKILL.md",
			plugin: "promptkitty-author-agent-instructions/SKILL.md",
		},
		{
			asset:  "skills/promptkitty-author-agent-instructions/references/provider-targets.md",
			plugin: "promptkitty-author-agent-instructions/references/provider-targets.md",
		},
	}

	for _, host := range []string{"opencode", "cursor"} {
		for _, test := range tests {
			assetPath := filepath.ToSlash(filepath.Join("assets", host, test.asset))
			embedded, err := setupAssets.ReadFile(assetPath)
			if err != nil {
				t.Fatal(err)
			}
			plugin, err := os.ReadFile(filepath.Join(pluginRoot, filepath.FromSlash(test.plugin)))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(embedded, plugin) {
				t.Errorf("%s differs from prefixed plugin %s", assetPath, test.plugin)
			}
		}
	}
}

func TestOpenCodeCommandsLoadMatchingSkills(t *testing.T) {
	tests := map[string][]string{
		"assets/opencode/commands/promptkitty-assemble.md": {
			"Load the `promptkitty-assemble` skill",
			"$ARGUMENTS",
		},
		"assets/opencode/commands/promptkitty-author-agent-instructions.md": {
			"Load the `promptkitty-author-agent-instructions` skill",
			"$ARGUMENTS",
		},
	}
	for path, fragments := range tests {
		command, err := setupAssets.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range fragments {
			if !strings.Contains(string(command), want) {
				t.Errorf("%s is missing %q", path, want)
			}
		}
	}
}

func TestSetupRejectsUnknownTarget(t *testing.T) {
	cmd := NewCommand(Options{})
	cmd.SetArgs([]string{"setup", "other"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "cursor") {
		t.Fatalf("setup error = %v", err)
	}
}

func findSetupAsset(t *testing.T, assets []setupAssetFile, destination string) setupAssetFile {
	t.Helper()
	for _, asset := range assets {
		if asset.destination == destination {
			return asset
		}
	}
	t.Fatalf("missing setup asset for %s", destination)
	return setupAssetFile{}
}
