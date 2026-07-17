package promptkitty

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const releaseVersion = "0.3.0"

func TestAssembleSkillUsesOnlyPromptKittyCLI(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("skills", "assemble", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}

	text := string(data)
	for _, want := range []string{
		"name: assemble",
		"# PromptKitty Assemble",
		"npx --yes @baldaworks/promptkitty@" + releaseVersion,
		"promptkitty search \"<intent>\" --type template --json",
		"promptkitty show \"<template>\" --json",
		"promptkitty assemble \"<template>\"",
		"--param-file \"<name>=<path>\"",
		"Return the assembled stdout unchanged.",
		"Interactive PromptKit templates are also assembled and returned, not executed.",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("assemble skill is missing %q", want)
		}
	}

	for _, forbidden := range []string{"callee", ".callee", "kind: role", "role create"} {
		if strings.Contains(strings.ToLower(text), forbidden) {
			t.Errorf("assemble skill contains forbidden role syntax %q", forbidden)
		}
	}
}

func TestSkillVariantsHaveMatchingBodies(t *testing.T) {
	canonical, err := os.ReadFile(filepath.Join("skills", "assemble", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	prefixed, err := os.ReadFile(filepath.Join("prefixed-skills", "promptkitty-assemble", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if skillBody(t, canonical) != skillBody(t, prefixed) {
		t.Fatal("canonical and prefixed skill bodies differ")
	}
}

func TestPluginManifests(t *testing.T) {
	for path, skills := range map[string]string{
		filepath.Join(".codex-plugin", "plugin.json"):  "./skills/",
		filepath.Join(".claude-plugin", "plugin.json"): "./skills/",
		filepath.Join(".grok-plugin", "plugin.json"):   "./prefixed-skills/",
		filepath.Join(".plugin", "plugin.json"):        "./prefixed-skills/",
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var manifest struct {
			Name    string `json:"name"`
			Version string `json:"version"`
			Skills  string `json:"skills"`
		}
		if err := json.Unmarshal(data, &manifest); err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		if manifest.Name != "promptkitty" || manifest.Version != releaseVersion || manifest.Skills != skills {
			t.Errorf("%s = %#v", path, manifest)
		}
	}
}

func TestCodexSkillMetadata(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("skills", "assemble", "agents", "openai.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`display_name: "PromptKitty Assemble"`,
		`short_description: "Assemble task-specific PromptKit prompts"`,
		`default_prompt: "Use $promptkitty:assemble to assemble a prompt for my task."`,
	} {
		if !strings.Contains(string(data), want) {
			t.Errorf("openai.yaml is missing %q", want)
		}
	}
}

func TestNPMDistributionIsStaticAndScoped(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", ".omnidist", "omnidist.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"name: promptkitty",
		"main: ./cmd/promptkitty",
		"cgo: false",
		"package: '@baldaworks/promptkitty'",
	} {
		if !strings.Contains(string(data), want) {
			t.Errorf("omnidist config is missing %q", want)
		}
	}
	if strings.Contains(string(data), "uv:") {
		t.Error("omnidist config unexpectedly enables uv publishing")
	}
}

func TestREADMEDocumentsNpxAndEverySetupTarget(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{
		"## PromptKitty Assemble for task-specific engineering prompts",
		"npx --yes @baldaworks/promptkitty@latest setup codex",
		"| Codex | `promptkitty setup codex` |",
		"| Claude Code | `promptkitty setup claude` |",
		"| Grok Build | `promptkitty setup grok` |",
		"| Copilot CLI | `promptkitty setup copilot` |",
		"| OpenCode | `promptkitty setup opencode` |",
		"CGO_ENABLED=0",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("README is missing %q", want)
		}
	}
}

func skillBody(t *testing.T, data []byte) string {
	t.Helper()
	parts := strings.SplitN(string(data), "---", 3)
	if len(parts) != 3 || strings.TrimSpace(parts[0]) != "" {
		t.Fatal("skill has invalid YAML frontmatter delimiters")
	}

	return strings.TrimSpace(parts[2])
}
