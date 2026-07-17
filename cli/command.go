package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/baldaworks/promptkitty"
	"github.com/spf13/cobra"
)

// Version is the PromptKitty application version.
const Version = "0.3.0"

// Options configures a reusable PromptKitty command tree.
type Options struct {
	// Use is the command name shown in help. The default is "promptkitty".
	Use string
	// Version is the version shown by Cobra's version flag.
	Version string
	// Library replaces the embedded catalog, primarily for hosts with a
	// compatible private catalog. A nil Library loads the embedded catalog.
	Library *promptkitty.Library
}

// NewCommand returns the complete PromptKitty command tree. Callers may mount
// the returned command below another Cobra root and add host-specific commands.
func NewCommand(options Options) *cobra.Command {
	use := strings.TrimSpace(options.Use)
	if use == "" {
		use = "promptkitty"
	}

	load := func() (*promptkitty.Library, error) {
		if options.Library != nil {
			return options.Library, nil
		}

		return promptkitty.New()
	}

	cmd := &cobra.Command{
		Use:           use,
		Short:         "Browse and assemble embedded PromptKit templates",
		Version:       options.Version,
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.NoArgs,
	}
	cmd.CompletionOptions.DisableDefaultCmd = true
	cmd.AddCommand(listCommand(load), searchCommand(load), showCommand(load), assembleCommand(load), setupCommand())

	return cmd
}

// Run executes the standalone PromptKitty command and returns its process exit
// code. Successful machine output is written only to stdout; diagnostics use
// stderr.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	cmd := NewCommand(Options{Version: Version})
	cmd.SetArgs(args)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetContext(ctx)

	if err := cmd.Execute(); err != nil {
		if jsonRequested(args) {
			_ = json.NewEncoder(stderr).Encode(map[string]string{"error": err.Error()})
		} else {
			_, _ = fmt.Fprintf(stderr, "Error: %v\n", err)
		}

		return 1
	}

	return 0
}

func jsonRequested(args []string) bool {
	for _, arg := range args {
		if arg == "--json" || arg == "--json=true" {
			return true
		}
	}

	return false
}

func writeJSON(cmd *cobra.Command, value any) error {
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")

	return encoder.Encode(value)
}
