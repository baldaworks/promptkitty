// The promptkitty command browses and assembles embedded PromptKit components.
package main

import (
	"context"
	"os"

	promptkittycli "github.com/baldaworks/promptkitty/cli"
)

func main() {
	os.Exit(promptkittycli.Run(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}
