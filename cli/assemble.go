package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/baldaworks/promptkitty"
	"github.com/spf13/cobra"
)

const (
	outputDirectoryMode = 0o750
	outputFileMode      = 0o600
)

func assembleCommand(load libraryLoader) *cobra.Command {
	var persona, format, output string
	var params, paramFiles, protocols, taxonomies []string
	var noFormat, jsonOutput, force bool

	cmd := &cobra.Command{
		Use:   "assemble <template>",
		Short: "Assemble a fully rendered PromptKit template",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if force && output == "" {
				return fmt.Errorf("--force requires --output")
			}

			values, err := parameterValues(params, paramFiles)
			if err != nil {
				return err
			}

			library, err := load()
			if err != nil {
				return fmt.Errorf("load embedded PromptKit catalog: %w", err)
			}

			result, err := library.Assemble(promptkitty.AssembleRequest{
				Template: args[0], Params: values, Persona: persona,
				AdditionalProtocols: protocols, AdditionalTaxonomies: taxonomies,
				Format: formatOverride(cmd, format, noFormat),
			})
			if err != nil {
				return err
			}

			if jsonOutput {
				return writeJSON(cmd, result)
			}
			if output != "" {
				if err := writeMarkdown(output, result.Markdown, force); err != nil {
					return err
				}

				_, err = fmt.Fprintf(cmd.OutOrStdout(), "created %s\n", output)
				return err
			}

			_, err = cmd.OutOrStdout().Write(markdownBytes(result.Markdown))
			return err
		},
	}
	cmd.Flags().StringArrayVarP(&params, "param", "p", nil, "template parameter as key=value; repeatable")
	cmd.Flags().StringArrayVar(&paramFiles, "param-file", nil, "template parameter as key=path; repeatable")
	cmd.Flags().StringVar(&persona, "persona", "", "replace the PromptKit persona")
	cmd.Flags().StringArrayVar(&protocols, "protocol", nil, "add a PromptKit protocol; repeatable")
	cmd.Flags().StringArrayVar(&taxonomies, "taxonomy", nil, "add a PromptKit taxonomy; repeatable")
	cmd.Flags().StringVar(&format, "format", "", "replace the PromptKit output format")
	cmd.Flags().BoolVar(&noFormat, "no-format", false, "omit the PromptKit output format")
	cmd.Flags().StringVarP(&output, "output", "o", "", "write Markdown to this file instead of stdout")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite an existing output file")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output the complete assembly result as JSON")
	cmd.MarkFlagsMutuallyExclusive("format", "no-format")
	cmd.MarkFlagsMutuallyExclusive("json", "output")

	return cmd
}

func parameterValues(raw, files []string) (map[string]string, error) {
	values := make(map[string]string, len(raw)+len(files))
	for _, item := range raw {
		name, value, err := splitAssignment(item)
		if err != nil {
			return nil, err
		}
		if _, exists := values[name]; exists {
			return nil, fmt.Errorf("PromptKit parameter %q is specified more than once", name)
		}
		values[name] = value
	}
	for _, item := range files {
		name, path, err := splitAssignment(item)
		if err != nil {
			return nil, err
		}
		if _, exists := values[name]; exists {
			return nil, fmt.Errorf("PromptKit parameter %q is specified more than once", name)
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read PromptKit parameter %q from %q: %w", name, path, err)
		}
		values[name] = string(content)
	}

	return values, nil
}

func splitAssignment(value string) (string, string, error) {
	name, assigned, ok := strings.Cut(value, "=")
	name = strings.TrimSpace(name)
	if !ok || name == "" {
		return "", "", fmt.Errorf("PromptKit parameter %q must use key=value", value)
	}

	return name, assigned, nil
}

func formatOverride(cmd *cobra.Command, format string, noFormat bool) *string {
	if noFormat {
		empty := ""
		return &empty
	}
	if cmd.Flags().Changed("format") {
		return &format
	}

	return nil
}

func markdownBytes(markdown string) []byte {
	return []byte(strings.TrimRight(markdown, "\n") + "\n")
}

func writeMarkdown(path, markdown string, force bool) (err error) {
	if err := os.MkdirAll(filepath.Dir(path), outputDirectoryMode); err != nil {
		return fmt.Errorf("create PromptKitty output directory: %w", err)
	}

	flags := os.O_WRONLY | os.O_CREATE | os.O_EXCL
	if force {
		flags = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	}
	file, err := os.OpenFile(path, flags, outputFileMode)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("output file %q already exists; use --force to overwrite it", path)
		}

		return fmt.Errorf("create PromptKitty output: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); err == nil && closeErr != nil {
			err = fmt.Errorf("close PromptKitty output: %w", closeErr)
		}
	}()

	if _, err := file.Write(markdownBytes(markdown)); err != nil {
		return fmt.Errorf("write PromptKitty output: %w", err)
	}

	return nil
}
