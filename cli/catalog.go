package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/baldaworks/promptkitty"
	"github.com/spf13/cobra"
)

const tablePadding = 2

type libraryLoader func() (*promptkitty.Library, error)

func listCommand(load libraryLoader) *cobra.Command {
	var componentType, category, language string
	var all, jsonOutput bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List PromptKit components",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			kind, err := componentTypeValue(componentType)
			if err != nil {
				return err
			}
			if kind == "" && !all {
				kind = promptkitty.ComponentTemplate
			}

			library, err := load()
			if err != nil {
				return fmt.Errorf("load embedded PromptKit catalog: %w", err)
			}

			components := library.List(promptkitty.Filter{Type: kind, Category: category, Language: language})
			return writeComponents(cmd, components, jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "list every component type")
	cmd.Flags().StringVar(&componentType, "type", "", "component type: persona, protocol, format, taxonomy, or template")
	cmd.Flags().StringVar(&category, "category", "", "filter by category")
	cmd.Flags().StringVar(&language, "language", "", "filter by language")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output components as JSON")

	return cmd
}

func searchCommand(load libraryLoader) *cobra.Command {
	var componentType string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search PromptKit components",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			kind, err := componentTypeValue(componentType)
			if err != nil {
				return err
			}

			library, err := load()
			if err != nil {
				return fmt.Errorf("load embedded PromptKit catalog: %w", err)
			}

			return writeComponents(cmd, library.Search(args[0], promptkitty.Filter{Type: kind}), jsonOutput)
		},
	}
	cmd.Flags().StringVar(&componentType, "type", "", "component type: persona, protocol, format, taxonomy, or template")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output components as JSON")

	return cmd
}

func showCommand(load libraryLoader) *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "show <name>",
		Short: "Show a PromptKit component",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			library, err := load()
			if err != nil {
				return fmt.Errorf("load embedded PromptKit catalog: %w", err)
			}

			detail, err := library.Show(args[0])
			if err != nil {
				return err
			}
			if jsonOutput {
				return writeJSON(cmd, detail)
			}

			return writeDetail(cmd, detail)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output the component as JSON")

	return cmd
}

func componentTypeValue(value string) (promptkitty.ComponentType, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}

	kind := promptkitty.ComponentType(value)
	switch kind {
	case promptkitty.ComponentPersona, promptkitty.ComponentProtocol, promptkitty.ComponentFormat,
		promptkitty.ComponentTaxonomy, promptkitty.ComponentTemplate:
		return kind, nil
	default:
		return "", fmt.Errorf("unsupported PromptKit component type %q", value)
	}
}

func writeComponents(cmd *cobra.Command, components []promptkitty.Component, jsonOutput bool) error {
	if jsonOutput {
		return writeJSON(cmd, components)
	}

	out := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, tablePadding, ' ', 0)
	currentGroup := ""
	for _, component := range components {
		group := string(component.Type)
		if component.Category != "" {
			group += " / " + component.Category
		}
		if group != currentGroup {
			if currentGroup != "" {
				if _, err := fmt.Fprintln(out); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprintf(out, "%s\nNAME\tDESCRIPTION\n", strings.ToUpper(group)); err != nil {
				return err
			}
			currentGroup = group
		}
		if _, err := fmt.Fprintf(out, "%s\t%s\n", component.Name, component.Description); err != nil {
			return err
		}
	}

	return out.Flush()
}

func writeDetail(cmd *cobra.Command, detail promptkitty.ComponentDetail) error {
	out := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, tablePadding, ' ', 0)
	fields := [][2]string{
		{"Name", detail.Name}, {"Type", string(detail.Type)}, {"Category", detail.Category},
		{"Language", detail.Language}, {"Path", detail.Path}, {"Description", detail.Description},
	}
	for _, field := range fields {
		if field[1] != "" {
			if _, err := fmt.Fprintf(out, "%s:\t%s\n", field[0], field[1]); err != nil {
				return err
			}
		}
	}

	for _, field := range []string{"persona", "protocols", "taxonomies", "format", "mode", "params"} {
		value, ok := detail.Metadata[field]
		if !ok {
			continue
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("format PromptKit metadata %q: %w", field, err)
		}
		if _, err := fmt.Fprintf(out, "%s:\t%s\n", strings.ToUpper(field[:1])+field[1:], encoded); err != nil {
			return err
		}
	}

	if len(detail.UsedByTemplates) > 0 {
		if _, err := fmt.Fprintf(out, "Used by:\t%s\n", strings.Join(detail.UsedByTemplates, ", ")); err != nil {
			return err
		}
	}

	return out.Flush()
}
