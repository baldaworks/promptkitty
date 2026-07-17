package promptkitty

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const releaseVersion = "0.4.0"

func TestAssembleSkillUsesPromptKittyCLIAndHandoff(t *testing.T) {
	data := readTestFile(t, filepath.Join("skills", "assemble", "SKILL.md"))
	text := string(data)
	for _, want := range []string{
		"name: assemble",
		"# PromptKitty Assemble",
		"npx --yes @baldaworks/promptkitty@" + releaseVersion,
		"promptkitty search \"<intent>\" --type template --json",
		"promptkitty show \"<template>\" --json",
		"promptkitty assemble \"<template>\"",
		"metadata.mode",
		"provisional assembly",
		"first confirmation gate",
		"Raw prompt",
		"Project instructions",
		"Subagent profile",
		"source-only",
		"promptkitty-author-agent-instructions",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("assemble skill is missing %q", want)
		}
	}
	assertNoCalleeRoleSyntax(t, text)
}

func TestAuthorAgentInstructionsSkillUsesPinnedTemplateAndProviderReference(t *testing.T) {
	data := readTestFile(t, filepath.Join("skills", "author-agent-instructions", "SKILL.md"))
	text := string(data)
	for _, want := range []string{
		"name: author-agent-instructions",
		"# PromptKitty Author Agent Instructions",
		"npx --yes @baldaworks/promptkitty@" + releaseVersion,
		"references/provider-targets.md",
		"promptkitty show \"author-agent-instructions\" --json",
		"promptkitty assemble \"author-agent-instructions\"",
		"--param-file \"behaviors=<source-prompt-path>\"",
		"explicit confirmation before any write",
		"Do not implement application code",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("author skill is missing %q", want)
		}
	}
	assertNoCalleeRoleSyntax(t, text)

	reference := string(readTestFile(t, filepath.Join("skills", "author-agent-instructions", "references", "provider-targets.md")))
	for _, want := range []string{
		"AGENTS.md",
		".codex/agents/<slug>.toml",
		".claude/agents/<slug>.md",
		".grok/agents/<slug>.md",
		".github/agents/<slug>.agent.md",
		".opencode/agents/<slug>.md",
		".cursor/agents/<slug>.md",
		"<!-- promptkitty:<slug>:begin -->",
	} {
		if !strings.Contains(reference, want) {
			t.Errorf("provider reference is missing %q", want)
		}
	}
	if strings.Contains(reference, ".cursorrules") {
		t.Error("provider reference contains deprecated .cursorrules path")
	}
}

func TestSkillVariantsHaveMatchingBodiesAndReferences(t *testing.T) {
	tests := []struct {
		canonical string
		prefixed  string
	}{
		{
			canonical: filepath.Join("skills", "assemble", "SKILL.md"),
			prefixed:  filepath.Join("prefixed-skills", "promptkitty-assemble", "SKILL.md"),
		},
		{
			canonical: filepath.Join("skills", "author-agent-instructions", "SKILL.md"),
			prefixed:  filepath.Join("prefixed-skills", "promptkitty-author-agent-instructions", "SKILL.md"),
		},
	}
	for _, test := range tests {
		canonical := readTestFile(t, test.canonical)
		prefixed := readTestFile(t, test.prefixed)
		if skillBody(t, canonical) != skillBody(t, prefixed) {
			t.Errorf("%s and %s bodies differ", test.canonical, test.prefixed)
		}
	}

	canonicalReference := readTestFile(t, filepath.Join("skills", "author-agent-instructions", "references", "provider-targets.md"))
	prefixedReference := readTestFile(t, filepath.Join("prefixed-skills", "promptkitty-author-agent-instructions", "references", "provider-targets.md"))
	if !bytes.Equal(canonicalReference, prefixedReference) {
		t.Fatal("canonical and prefixed provider references differ")
	}
}

func TestPluginManifests(t *testing.T) {
	for path, skills := range map[string]string{
		filepath.Join(".codex-plugin", "plugin.json"):  "./skills/",
		filepath.Join(".claude-plugin", "plugin.json"): "./skills/",
		filepath.Join(".grok-plugin", "plugin.json"):   "./prefixed-skills/",
		filepath.Join(".plugin", "plugin.json"):        "./prefixed-skills/",
	} {
		data := readTestFile(t, path)
		var manifest struct {
			Name        string `json:"name"`
			Version     string `json:"version"`
			Description string `json:"description"`
			Skills      string `json:"skills"`
		}
		if err := json.Unmarshal(data, &manifest); err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		if manifest.Name != "promptkitty" || manifest.Version != releaseVersion || manifest.Skills != skills {
			t.Errorf("%s = %#v", path, manifest)
		}
		if !strings.Contains(manifest.Description, "agent instructions") {
			t.Errorf("%s description does not mention agent instructions", path)
		}
	}
}

func TestCodexSkillMetadata(t *testing.T) {
	tests := map[string][]string{
		filepath.Join("skills", "assemble", "agents", "openai.yaml"): {
			`display_name: "PromptKitty Assemble"`,
			`short_description: "Assemble raw and interactive PromptKit prompts"`,
			`default_prompt: "Use $promptkitty:assemble`,
		},
		filepath.Join("skills", "author-agent-instructions", "agents", "openai.yaml"): {
			`display_name: "PromptKitty Author Agent Instructions"`,
			`short_description: "Author provider-native agent instructions"`,
			`default_prompt: "Use $promptkitty:author-agent-instructions`,
		},
	}
	for path, fragments := range tests {
		data := string(readTestFile(t, path))
		for _, want := range fragments {
			if !strings.Contains(data, want) {
				t.Errorf("%s is missing %q", path, want)
			}
		}
	}
}

func TestNPMDistributionIsStaticAndScoped(t *testing.T) {
	data := readTestFile(t, filepath.Join("..", "..", ".omnidist", "omnidist.yaml"))
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

func TestREADMEDocumentsNpxSkillsAndEverySetupTarget(t *testing.T) {
	data := readTestFile(t, filepath.Join("..", "..", "README.md"))
	text := string(data)
	for _, want := range []string{
		"## PromptKitty Assemble and reusable agent instructions",
		"npx --yes @baldaworks/promptkitty@latest setup codex",
		"| Codex | `promptkitty setup codex` |",
		"| Claude Code | `promptkitty setup claude` |",
		"| Grok Build | `promptkitty setup grok` |",
		"| Copilot CLI | `promptkitty setup copilot` |",
		"| OpenCode | `promptkitty setup opencode` |",
		"| Cursor | `promptkitty setup cursor` |",
		"PromptKitty Author Agent Instructions",
		"provisional assembly",
		"CGO_ENABLED=0",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("README is missing %q", want)
		}
	}
}

func readTestFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func assertNoCalleeRoleSyntax(t *testing.T, text string) {
	t.Helper()
	for _, forbidden := range []string{"callee", ".callee", "kind: role", "role create"} {
		if strings.Contains(strings.ToLower(text), forbidden) {
			t.Errorf("skill contains forbidden role syntax %q", forbidden)
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
