package cli

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const (
	setupDirMode  = 0o755
	setupFileMode = 0o644
)

type setupTarget struct {
	prepare          func(context.Context, io.Writer) error
	commands         [][]string
	install          func(bool) (setupInstallResult, error)
	installedMessage string
	unchangedMessage string
}

type setupInstallResult struct {
	created   []string
	unchanged []string
}

var runSetupCommand = runSetupCommandDefault

//go:embed assets/opencode assets/cursor
var setupAssets embed.FS

type setupAssetFile struct {
	source      string
	destination string
}

var openCodeAssetFiles = []setupAssetFile{
	{
		source:      "assets/opencode/skills/promptkitty-assemble/SKILL.md",
		destination: ".opencode/skills/promptkitty-assemble/SKILL.md",
	},
	{
		source:      "assets/opencode/skills/promptkitty-author-agent-instructions/SKILL.md",
		destination: ".opencode/skills/promptkitty-author-agent-instructions/SKILL.md",
	},
	{
		source:      "assets/opencode/skills/promptkitty-author-agent-instructions/references/provider-targets.md",
		destination: ".opencode/skills/promptkitty-author-agent-instructions/references/provider-targets.md",
	},
	{
		source:      "assets/opencode/commands/promptkitty.md",
		destination: ".opencode/commands/promptkitty.md",
	},
	{
		source:      "assets/opencode/commands/promptkitty-author-agent-instructions.md",
		destination: ".opencode/commands/promptkitty-author-agent-instructions.md",
	},
}

var cursorAssetFiles = []setupAssetFile{
	{
		source:      "assets/cursor/skills/promptkitty-assemble/SKILL.md",
		destination: ".cursor/skills/promptkitty-assemble/SKILL.md",
	},
	{
		source:      "assets/cursor/skills/promptkitty-author-agent-instructions/SKILL.md",
		destination: ".cursor/skills/promptkitty-author-agent-instructions/SKILL.md",
	},
	{
		source:      "assets/cursor/skills/promptkitty-author-agent-instructions/references/provider-targets.md",
		destination: ".cursor/skills/promptkitty-author-agent-instructions/references/provider-targets.md",
	},
}

func setupCommand() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "setup <codex|claude|grok|copilot|opencode|cursor>",
		Short: "Install the PromptKitty integration for an agent host",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target, err := setupTargetFor(args[0])
			if err != nil {
				return err
			}

			return installSetupTarget(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr(), target, force)
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing local setup files")

	return cmd
}

func installSetupTarget(ctx context.Context, stdout, stderr io.Writer, target setupTarget, force bool) error {
	if target.prepare != nil {
		if err := target.prepare(ctx, stderr); err != nil {
			return err
		}
	}
	for _, command := range target.commands {
		if err := runSetupCommand(ctx, stdout, stderr, command[0], command[1:]...); err != nil {
			return err
		}
	}
	if target.install == nil {
		return nil
	}

	result, err := target.install(force)
	if err != nil {
		return err
	}
	if len(result.created) > 0 {
		if _, err := fmt.Fprintln(stdout, target.installedMessage); err != nil {
			return err
		}
	}
	if len(result.unchanged) > 0 {
		if _, err := fmt.Fprintln(stdout, target.unchangedMessage); err != nil {
			return err
		}
	}

	return nil
}

func setupTargetFor(name string) (setupTarget, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "codex":
		return setupTarget{
			prepare: prepareCodexMarketplace,
			commands: [][]string{
				{"codex", "plugin", "marketplace", "add", "baldaworks/promptkitty"},
				{"codex", "plugin", "add", "promptkitty@promptkitty"},
			},
		}, nil
	case "claude":
		return setupTarget{
			commands: [][]string{
				{"claude", "plugin", "marketplace", "add", "baldaworks/promptkitty"},
				{"claude", "plugin", "install", "promptkitty@promptkitty", "--scope", "project"},
			},
		}, nil
	case "grok":
		return setupTarget{
			commands: [][]string{
				{"grok", "plugin", "marketplace", "add", "baldaworks/promptkitty"},
				{"grok", "plugin", "install", "promptkitty@promptkitty", "--trust"},
			},
		}, nil
	case "copilot":
		return setupTarget{
			commands: [][]string{
				{"copilot", "plugin", "marketplace", "add", "baldaworks/promptkitty"},
				{"copilot", "plugin", "install", "promptkitty@promptkitty"},
			},
		}, nil
	case "opencode":
		return setupTarget{
			install:          writeOpenCodeIntegration,
			installedMessage: "Installed the PromptKitty skills and commands for OpenCode.",
			unchangedMessage: "Existing OpenCode PromptKitty files were left unchanged.",
		}, nil
	case "cursor":
		return setupTarget{
			install:          writeCursorIntegration,
			installedMessage: "Installed the PromptKitty skills for Cursor.",
			unchangedMessage: "Existing Cursor PromptKitty files were left unchanged.",
		}, nil
	default:
		return setupTarget{}, fmt.Errorf(
			"unsupported setup target %q (want codex, claude, grok, copilot, opencode, or cursor)",
			name,
		)
	}
}

func prepareCodexMarketplace(ctx context.Context, stderr io.Writer) error {
	var diagnostics bytes.Buffer
	err := runSetupCommand(
		ctx,
		io.Discard,
		&diagnostics,
		"codex",
		"plugin",
		"marketplace",
		"remove",
		"promptkitty",
	)
	if err != nil && strings.Contains(diagnostics.String(), "not configured or installed") {
		return nil
	}
	if _, writeErr := io.Copy(stderr, &diagnostics); writeErr != nil {
		return fmt.Errorf("write Codex marketplace diagnostics: %w", writeErr)
	}
	if err != nil {
		return fmt.Errorf("remove existing Codex marketplace: %w", err)
	}

	return nil
}

func writeOpenCodeIntegration(force bool) (setupInstallResult, error) {
	return writeLocalIntegration("OpenCode", openCodeAssetFiles, force)
}

func writeCursorIntegration(force bool) (setupInstallResult, error) {
	return writeLocalIntegration("Cursor", cursorAssetFiles, force)
}

func writeLocalIntegration(host string, assets []setupAssetFile, force bool) (setupInstallResult, error) {
	result := setupInstallResult{}
	for _, asset := range assets {
		content, err := setupAssets.ReadFile(asset.source)
		if err != nil {
			return setupInstallResult{}, fmt.Errorf("read embedded %s asset %q: %w", host, asset.source, err)
		}
		created, err := writeSetupFile(filepath.FromSlash(asset.destination), content, force)
		if err != nil {
			return setupInstallResult{}, fmt.Errorf("write %s asset %q: %w", host, asset.destination, err)
		}
		if created {
			result.created = append(result.created, asset.destination)
		} else {
			result.unchanged = append(result.unchanged, asset.destination)
		}
	}

	return result, nil
}

func writeSetupFile(path string, content []byte, force bool) (bool, error) {
	if !force {
		if _, err := os.Stat(path); err == nil {
			return false, nil
		} else if !os.IsNotExist(err) {
			return false, fmt.Errorf("check existing file: %w", err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), setupDirMode); err != nil {
		return false, fmt.Errorf("create parent directory: %w", err)
	}
	if err := os.WriteFile(path, content, setupFileMode); err != nil {
		return false, fmt.Errorf("write file: %w", err)
	}

	return true, nil
}

func runSetupCommandDefault(ctx context.Context, stdout, stderr io.Writer, name string, args ...string) error {
	command := exec.CommandContext(ctx, name, args...)
	command.Stdout = stdout
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		invocation := strings.Join(append([]string{name}, args...), " ")
		return fmt.Errorf("run %s: %w", invocation, err)
	}

	return nil
}
