package main

import (
	"os"

	"github.com/yangkushu/ai-session-history/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
