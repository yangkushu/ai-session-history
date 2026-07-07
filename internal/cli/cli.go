package cli

import (
	"fmt"
	"io"
)

func Run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "Usage: ai-history <command> [flags]")
		fmt.Fprintln(stderr, "Commands: doctor, list, show, context")
		return 2
	}
	fmt.Fprintf(stderr, "unknown command: %s\n", args[0])
	return 2
}
