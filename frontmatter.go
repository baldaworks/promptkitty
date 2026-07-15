package promptkitty

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

func splitMarkdownDocument(raw string) (map[string]any, string, error) {
	content := strings.ReplaceAll(raw, "\r\n", "\n")
	content = strings.TrimSpace(content)

	for strings.HasPrefix(content, "<!--") {
		end := strings.Index(content, "-->")
		if end < 0 {
			return nil, "", fmt.Errorf("unterminated leading HTML comment")
		}
		content = strings.TrimSpace(content[end+3:])
	}

	metadata := make(map[string]any)
	if !strings.HasPrefix(content, "---\n") {
		return metadata, content, nil
	}

	remainder := content[len("---\n"):]
	end := strings.Index(remainder, "\n---\n")
	if end < 0 {
		return nil, "", fmt.Errorf("unterminated YAML frontmatter")
	}

	if err := yaml.Unmarshal([]byte(remainder[:end]), &metadata); err != nil {
		return nil, "", fmt.Errorf("parse YAML frontmatter: %w", err)
	}

	return metadata, strings.TrimSpace(remainder[end+len("\n---\n"):]), nil
}
