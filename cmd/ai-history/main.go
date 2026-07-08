package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/yangzeqi/ai-session-history/internal/history"
)

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr, defaultService()))
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer, service history.Service) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: ai-history <doctor|list|show|context>")
		return 2
	}
	switch args[0] {
	case "doctor":
		return runDoctor(ctx, args[1:], stdout, stderr, service)
	case "list":
		return runList(ctx, args[1:], stdout, stderr, service)
	case "show":
		return runShow(ctx, args[1:], stdout, stderr, service)
	case "context":
		return runContext(ctx, args[1:], stdout, stderr, service)
	case "search":
		fmt.Fprintln(stderr, "search is unavailable in P0")
		return 2
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n", args[0])
		return 2
	}
}

func runDoctor(ctx context.Context, args []string, stdout, stderr io.Writer, service history.Service) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOut := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	diagnostics := service.Doctor(ctx)
	if *jsonOut {
		return writeJSON(stdout, stderr, diagnostics)
	}
	for _, diagnostic := range diagnostics {
		status := diagnostic.Status
		if status == "" && diagnostic.Available {
			status = "available"
		}
		if status == "" {
			status = "unavailable"
		}
		fmt.Fprintf(stdout, "%s: %s", diagnostic.Source, status)
		if diagnostic.Reason != "" {
			fmt.Fprintf(stdout, " (%s)", diagnostic.Reason)
		}
		fmt.Fprintln(stdout)
	}
	return 0
}

func runList(ctx context.Context, args []string, stdout, stderr io.Writer, service history.Service) int {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	source := fs.String("source", "", "source filter")
	cwd := fs.String("cwd", "", "exact cwd filter")
	under := fs.String("under", "", "directory subtree filter")
	limit := fs.Int("limit", 50, "maximum sessions")
	jsonOut := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	result := service.List(ctx, history.ListOptions{
		Source: history.Source(*source),
		CWD:    *cwd,
		Under:  *under,
		Limit:  *limit,
	})
	if *jsonOut {
		return writeJSON(stdout, stderr, result.Sessions)
	}
	for _, session := range result.Sessions {
		fmt.Fprintf(stdout, "%s\t%s\t%s\n", session.ID, session.Title, session.CWD)
	}
	return 0
}

func runShow(ctx context.Context, args []string, stdout, stderr io.Writer, service history.Service) int {
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOut := fs.Bool("json", false, "emit JSON")
	_ = fs.String("mode", "clean", "content mode")
	_ = fs.Int("max-chars", 50000, "maximum detail characters")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "usage: ai-history show <source:id>")
		return 2
	}
	detail, err := service.Show(ctx, fs.Arg(0))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *jsonOut {
		return writeJSON(stdout, stderr, detail)
	}
	for _, turn := range detail.Turns {
		fmt.Fprintf(stdout, "%s: %s\n\n", turn.Role, turn.Text)
	}
	return 0
}

func runContext(ctx context.Context, args []string, stdout, stderr io.Writer, service history.Service) int {
	fs := flag.NewFlagSet("context", flag.ContinueOnError)
	fs.SetOutput(stderr)
	targetCWD := fs.String("target-cwd", "", "target cwd")
	maxChars := fs.Int("max-chars", 20000, "maximum context characters")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "usage: ai-history context <source:id>")
		return 2
	}
	text, err := service.Context(ctx, fs.Arg(0), history.ContextOptions{TargetCWD: *targetCWD, MaxChars: *maxChars})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stdout, text)
	return 0
}

func writeJSON(stdout, stderr io.Writer, value any) int {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stdout, string(data))
	return 0
}

func defaultService() history.Service {
	cursorRoot := os.ExpandEnv("$HOME/Library/Application Support/Cursor/User")
	return history.NewService(history.Config{}, map[history.Source]history.Reader{
		history.SourceCodex:  unavailableReader{source: history.SourceCodex, reason: "Codex reader is not implemented yet"},
		history.SourceClaude: unavailableReader{source: history.SourceClaude, reason: "Claude Code reader is not implemented yet"},
		history.SourceCursor: history.NewCursorStorageReader([]string{cursorRoot}),
	})
}

type unavailableReader struct {
	source history.Source
	reason string
}

func (r unavailableReader) ListSessions(context.Context) ([]history.SessionSummary, error) {
	return nil, fmt.Errorf("source_unavailable: %s", r.reason)
}

func (r unavailableReader) GetSession(context.Context, string) (history.SessionDetail, error) {
	return history.SessionDetail{}, fmt.Errorf("source_unavailable: %s", r.reason)
}

func (r unavailableReader) Doctor(context.Context) history.SourceDiagnostic {
	return history.SourceDiagnostic{
		Source: r.source,
		Status: "unavailable",
		Code:   "source_unavailable",
		Reason: r.reason,
	}
}

func parsePositiveInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
