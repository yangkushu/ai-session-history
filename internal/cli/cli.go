package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/yangkushu/ai-session-history/internal/core"
)

type Service interface {
	Doctor() []core.SourceDiagnostic
	List(core.ListOptions) core.ListResult
	Show(string, core.ShowOptions) (core.SessionDetail, error)
	Context(string, core.ContextOptions) (string, error)
}

func Run(args []string, stdout io.Writer, stderr io.Writer) int {
	configPath, args := extractConfig(args)
	service, err := NewService(configPath)
	if err != nil {
		writeError(stderr, err)
		return 1
	}
	return RunWithService(args, stdout, stderr, service)
}

func RunWithService(args []string, stdout io.Writer, stderr io.Writer, service Service) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "Usage: ai-history <command> [flags]")
		fmt.Fprintln(stderr, "Commands: doctor, list, show, context")
		return 2
	}
	switch args[0] {
	case "search":
		fmt.Fprintln(stderr, "search is not available in P0")
		return 2
	case "export":
		fmt.Fprintln(stderr, "export is not available in P0; use context for Markdown handoff or show --json for normalized detail")
		return 2
	case "import":
		fmt.Fprintln(stderr, "import is not available in P0")
		return 2
	case "doctor":
		return runDoctor(args[1:], stdout, stderr, service)
	case "list":
		return runList(args[1:], stdout, stderr, service)
	case "show":
		return runShow(args[1:], stdout, stderr, service)
	case "context":
		return runContext(args[1:], stdout, stderr, service)
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n", args[0])
		return 2
	}
}

func runDoctor(args []string, stdout io.Writer, stderr io.Writer, service Service) int {
	flags := flag.NewFlagSet("doctor", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "write JSON output")
	_ = flags.String("config", "", "config path")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if service == nil {
		fmt.Fprintln(stderr, "service is not configured")
		return 1
	}
	diagnostics := service.Doctor()
	if *jsonOut {
		return writeJSON(stdout, diagnostics)
	}
	for _, diagnostic := range diagnostics {
		fmt.Fprintf(stdout, "%s\t%s\n", diagnostic.Source, diagnostic.Status)
	}
	return 0
}

func runList(args []string, stdout io.Writer, stderr io.Writer, service Service) int {
	flags := flag.NewFlagSet("list", flag.ContinueOnError)
	flags.SetOutput(stderr)
	sourceText := flags.String("source", "", "source filter")
	cwd := flags.String("cwd", "", "exact working directory")
	under := flags.String("under", "", "working directory subtree")
	limit := flags.Int("limit", 50, "maximum sessions")
	jsonOut := flags.Bool("json", false, "write JSON output")
	_ = flags.String("config", "", "config path")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if service == nil {
		fmt.Fprintln(stderr, "service is not configured")
		return 1
	}
	source := core.Source(*sourceText)
	if source != "" && !core.IsSource(source) {
		fmt.Fprintf(stderr, "invalid source: %s\n", source)
		return 2
	}
	result := service.List(core.ListOptions{
		Source: source,
		CWD:    *cwd,
		Under:  *under,
		Limit:  *limit,
	})
	if *jsonOut {
		return writeJSON(stdout, result)
	}
	for _, session := range result.Sessions {
		fmt.Fprintf(stdout, "%s\t%s\t%s\t%s\n", session.ID, session.Source, session.Title, session.CWD)
	}
	return 0
}

func runShow(args []string, stdout io.Writer, stderr io.Writer, service Service) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "Usage: ai-history show <session-id> [--mode clean|summary|raw] [--max-chars n] [--json]")
		return 2
	}
	sessionID := args[0]
	flags := flag.NewFlagSet("show", flag.ContinueOnError)
	flags.SetOutput(stderr)
	modeText := flags.String("mode", string(core.ModeClean), "content mode")
	maxChars := flags.Int("max-chars", 0, "maximum output characters")
	jsonOut := flags.Bool("json", false, "write JSON output")
	_ = flags.String("config", "", "config path")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "Usage: ai-history show <session-id> [--mode clean|summary|raw] [--max-chars n] [--json]")
		return 2
	}
	if service == nil {
		fmt.Fprintln(stderr, "service is not configured")
		return 1
	}
	mode := core.ContentMode(*modeText)
	if mode != core.ModeClean && mode != core.ModeSummary && mode != core.ModeRaw {
		fmt.Fprintf(stderr, "invalid mode: %s\n", mode)
		return 2
	}
	detail, err := service.Show(sessionID, core.ShowOptions{Mode: mode, MaxChars: *maxChars})
	if err != nil {
		writeError(stderr, err)
		return 1
	}
	if *jsonOut {
		return writeJSON(stdout, detail)
	}
	for _, turn := range detail.Turns {
		fmt.Fprintf(stdout, "%s: %s\n\n", turn.Role, turn.Text)
	}
	return 0
}

func runContext(args []string, stdout io.Writer, stderr io.Writer, service Service) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "Usage: ai-history context <session-id> [--target-cwd path] [--max-chars n]")
		return 2
	}
	sessionID := args[0]
	flags := flag.NewFlagSet("context", flag.ContinueOnError)
	flags.SetOutput(stderr)
	targetCWD := flags.String("target-cwd", "", "target working directory")
	maxChars := flags.Int("max-chars", 0, "maximum output characters")
	_ = flags.String("config", "", "config path")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "Usage: ai-history context <session-id> [--target-cwd path] [--max-chars n]")
		return 2
	}
	if service == nil {
		fmt.Fprintln(stderr, "service is not configured")
		return 1
	}
	text, err := service.Context(sessionID, core.ContextOptions{
		TargetCWD: *targetCWD,
		MaxChars:  *maxChars,
	})
	if err != nil {
		writeError(stderr, err)
		return 1
	}
	fmt.Fprint(stdout, text)
	return 0
}

func writeJSON(stdout io.Writer, value any) int {
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return 1
	}
	return 0
}

func extractConfig(args []string) (string, []string) {
	cleaned := make([]string, 0, len(args))
	for index := 0; index < len(args); index++ {
		arg := args[index]
		if arg == "--config" && index+1 < len(args) {
			index++
			return args[index], append(cleaned, args[index+1:]...)
		}
		if value, ok := strings.CutPrefix(arg, "--config="); ok {
			return value, append(cleaned, args[index+1:]...)
		}
		cleaned = append(cleaned, arg)
	}
	return "", cleaned
}

func parseLimit(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func writeError(stderr io.Writer, err error) {
	if appErr, ok := err.(*core.AppError); ok {
		fmt.Fprintf(stderr, "%s: %s\n", appErr.Code, appErr.Message)
		return
	}
	fmt.Fprintln(stderr, err)
}
