package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/yangkushu/ai-session-history/internal/core"
	"github.com/yangkushu/ai-session-history/internal/render"
)

type Service interface {
	Doctor() []core.SourceDiagnostic
	List(core.ListOptions) core.ListResult
	Show(string, core.ShowOptions) (core.SessionDetail, error)
	Context(string, core.ContextOptions) (string, error)
	ContextHandoff(string, core.ContextOptions) (render.HandoffContext, error)
}

var (
	version   = "dev"
	commit    = ""
	buildDate = ""
)

func Run(args []string, stdout io.Writer, stderr io.Writer) int {
	if handled, code := handleHelp(args, stdout, stderr); handled {
		return code
	}
	if isTopLevelVersion(args) {
		writeVersion(stdout)
		return 0
	}
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
		writeTopLevelUsage(stderr)
		return 2
	}
	if handled, code := handleHelp(args, stdout, stderr); handled {
		return code
	}
	if isTopLevelVersion(args) {
		writeVersion(stdout)
		return 0
	}
	switch args[0] {
	case "help":
		if len(args) == 1 {
			writeTopLevelUsage(stdout)
			return 0
		}
		if len(args) == 2 && writeCommandUsage(args[1], stdout) {
			return 0
		}
		fmt.Fprintf(stderr, "unknown help topic: %s\n", strings.Join(args[1:], " "))
		return 2
	case "version":
		if len(args) == 1 {
			writeVersion(stdout)
			return 0
		}
		fmt.Fprintln(stderr, "Usage: ai-history version")
		return 2
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
	if hasHelpFlag(args) {
		writeDoctorUsage(stdout)
		return 0
	}
	flags := flag.NewFlagSet("doctor", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var jsonOut bool
	flags.BoolVar(&jsonOut, "json", false, "write JSON output")
	flags.BoolVar(&jsonOut, "j", false, "write JSON output")
	_ = flags.String("config", "", "config path")
	_ = flags.String("c", "", "config path")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if service == nil {
		fmt.Fprintln(stderr, "service is not configured")
		return 1
	}
	diagnostics := service.Doctor()
	if jsonOut {
		return writeJSON(stdout, diagnostics)
	}
	for _, diagnostic := range diagnostics {
		fmt.Fprintf(stdout, "%s\t%s\n", diagnostic.Source, diagnostic.Status)
	}
	return 0
}

func runList(args []string, stdout io.Writer, stderr io.Writer, service Service) int {
	if hasHelpFlag(args) {
		writeListUsage(stdout)
		return 0
	}
	flags := flag.NewFlagSet("list", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var sourceText string
	flags.StringVar(&sourceText, "source", "", "source filter")
	flags.StringVar(&sourceText, "s", "", "source filter")
	cwd := flags.String("cwd", "", "exact working directory")
	under := flags.String("under", "", "working directory subtree")
	var limit int
	flags.IntVar(&limit, "limit", 50, "maximum sessions")
	flags.IntVar(&limit, "l", 50, "maximum sessions")
	var jsonOut bool
	flags.BoolVar(&jsonOut, "json", false, "write JSON output")
	flags.BoolVar(&jsonOut, "j", false, "write JSON output")
	here := flags.Bool("here", false, "use current working directory subtree")
	_ = flags.String("config", "", "config path")
	_ = flags.String("c", "", "config path")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *here {
		if *cwd != "" || *under != "" {
			fmt.Fprintln(stderr, "--here cannot be combined with --cwd or --under")
			return 2
		}
		workingDir, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(stderr, "cannot resolve current directory: %v\n", err)
			return 1
		}
		*under = workingDir
	}
	if service == nil {
		fmt.Fprintln(stderr, "service is not configured")
		return 1
	}
	source := core.Source(sourceText)
	if source != "" && !core.IsSource(source) {
		fmt.Fprintf(stderr, "invalid source: %s\n", source)
		return 2
	}
	result := service.List(core.ListOptions{
		Source: source,
		CWD:    *cwd,
		Under:  *under,
		Limit:  limit,
	})
	if jsonOut {
		return writeJSON(stdout, result)
	}
	for _, session := range result.Sessions {
		fmt.Fprintf(stdout, "%s\t%s\t%s\t%s\n", session.ID, session.Source, session.Title, session.CWD)
	}
	return 0
}

func runShow(args []string, stdout io.Writer, stderr io.Writer, service Service) int {
	if len(args) == 0 {
		writeShowUsage(stderr)
		return 2
	}
	if len(args) == 1 && isHelpArg(args[0]) {
		writeShowUsage(stdout)
		return 0
	}
	sessionID := args[0]
	if hasHelpFlag(args[1:]) {
		writeShowUsage(stdout)
		return 0
	}
	flags := flag.NewFlagSet("show", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var modeText string
	flags.StringVar(&modeText, "mode", string(core.ModeClean), "content mode")
	flags.StringVar(&modeText, "m", string(core.ModeClean), "content mode")
	var maxChars int
	flags.IntVar(&maxChars, "max-chars", 0, "maximum output characters")
	flags.IntVar(&maxChars, "n", 0, "maximum output characters")
	var jsonOut bool
	flags.BoolVar(&jsonOut, "json", false, "write JSON output")
	flags.BoolVar(&jsonOut, "j", false, "write JSON output")
	_ = flags.String("config", "", "config path")
	_ = flags.String("c", "", "config path")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		writeShowUsage(stderr)
		return 2
	}
	if service == nil {
		fmt.Fprintln(stderr, "service is not configured")
		return 1
	}
	mode := core.ContentMode(modeText)
	if mode != core.ModeClean && mode != core.ModeSummary && mode != core.ModeRaw {
		fmt.Fprintf(stderr, "invalid mode: %s\n", mode)
		return 2
	}
	detail, err := service.Show(sessionID, core.ShowOptions{Mode: mode, MaxChars: maxChars})
	if err != nil {
		writeError(stderr, err)
		return 1
	}
	if jsonOut {
		return writeJSON(stdout, detail)
	}
	for _, turn := range detail.Turns {
		fmt.Fprintf(stdout, "%s: %s\n\n", turn.Role, turn.Text)
	}
	return 0
}

func runContext(args []string, stdout io.Writer, stderr io.Writer, service Service) int {
	if len(args) == 0 {
		writeContextUsage(stderr)
		return 2
	}
	if len(args) == 1 && isHelpArg(args[0]) {
		writeContextUsage(stdout)
		return 0
	}
	sessionID := args[0]
	if hasHelpFlag(args[1:]) {
		writeContextUsage(stdout)
		return 0
	}
	flags := flag.NewFlagSet("context", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var targetCWD string
	flags.StringVar(&targetCWD, "target-cwd", "", "target working directory")
	flags.StringVar(&targetCWD, "t", "", "target working directory")
	var maxChars int
	flags.IntVar(&maxChars, "max-chars", 0, "maximum output characters")
	flags.IntVar(&maxChars, "n", 0, "maximum output characters")
	var jsonOut bool
	flags.BoolVar(&jsonOut, "json", false, "write JSON output")
	flags.BoolVar(&jsonOut, "j", false, "write JSON output")
	_ = flags.String("config", "", "config path")
	_ = flags.String("c", "", "config path")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		writeContextUsage(stderr)
		return 2
	}
	if service == nil {
		fmt.Fprintln(stderr, "service is not configured")
		return 1
	}
	if jsonOut {
		handoff, err := service.ContextHandoff(sessionID, core.ContextOptions{
			TargetCWD: targetCWD,
			MaxChars:  maxChars,
		})
		if err != nil {
			writeError(stderr, err)
			return 1
		}
		return writeJSON(stdout, handoff)
	}
	text, err := service.Context(sessionID, core.ContextOptions{
		TargetCWD: targetCWD,
		MaxChars:  maxChars,
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
		if arg == "-c" && index+1 < len(args) {
			index++
			return args[index], append(cleaned, args[index+1:]...)
		}
		if value, ok := strings.CutPrefix(arg, "-c="); ok {
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

func isTopLevelVersion(args []string) bool {
	return len(args) == 1 && args[0] == "--version"
}

func isHelpArg(arg string) bool {
	return arg == "--help" || arg == "-h"
}

func hasHelpFlag(args []string) bool {
	for _, arg := range args {
		if isHelpArg(arg) {
			return true
		}
	}
	return false
}

func handleHelp(args []string, stdout io.Writer, stderr io.Writer) (bool, int) {
	if len(args) == 1 && isHelpArg(args[0]) {
		writeTopLevelUsage(stdout)
		return true, 0
	}
	if len(args) > 0 && args[0] == "help" {
		if len(args) == 1 {
			writeTopLevelUsage(stdout)
			return true, 0
		}
		if writeCommandUsage(args[1], stdout) {
			return true, 0
		}
		fmt.Fprintf(stderr, "unknown help topic: %s\n", strings.Join(args[1:], " "))
		return true, 2
	}
	if len(args) > 1 && hasHelpFlag(args[1:]) {
		if writeCommandUsage(args[0], stdout) {
			return true, 0
		}
	}
	return false, 0
}

func writeTopLevelUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: ai-history <command> [flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  doctor     Check source availability")
	fmt.Fprintln(w, "  list       List sessions")
	fmt.Fprintln(w, "  show       Show session detail")
	fmt.Fprintln(w, "  context    Render Markdown handoff context")
	fmt.Fprintln(w, "  version    Show version information")
	fmt.Fprintln(w, "  help       Show help")
}

func writeCommandUsage(command string, w io.Writer) bool {
	switch command {
	case "doctor":
		writeDoctorUsage(w)
	case "list":
		writeListUsage(w)
	case "show":
		writeShowUsage(w)
	case "context":
		writeContextUsage(w)
	case "version":
		fmt.Fprintln(w, "Usage: ai-history version")
	default:
		return false
	}
	return true
}

func writeDoctorUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: ai-history doctor [--json] [--config path]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "  --json, -j       write JSON output")
	fmt.Fprintln(w, "  --config, -c     config path")
}

func writeListUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: ai-history list [--source source] [--cwd path] [--under path] [--here] [--limit n] [--json] [--config path]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "  --source, -s     source filter")
	fmt.Fprintln(w, "  --cwd            exact working directory")
	fmt.Fprintln(w, "  --under          working directory subtree")
	fmt.Fprintln(w, "  --here           use current working directory subtree")
	fmt.Fprintln(w, "  --limit, -l      maximum sessions")
	fmt.Fprintln(w, "  --json, -j       write JSON output")
	fmt.Fprintln(w, "  --config, -c     config path")
}

func writeShowUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: ai-history show <session-id> [--mode clean|summary|raw] [--max-chars n] [--json] [--config path]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "  --mode, -m       content mode")
	fmt.Fprintln(w, "  --max-chars, -n  maximum output characters")
	fmt.Fprintln(w, "  --json, -j       write JSON output")
	fmt.Fprintln(w, "  --config, -c     config path")
}

func writeContextUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: ai-history context <session-id> [--target-cwd path] [--max-chars n] [--json] [--config path]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "  --target-cwd, -t target working directory")
	fmt.Fprintln(w, "  --max-chars, -n  maximum output characters")
	fmt.Fprintln(w, "  --json, -j       write JSON output")
	fmt.Fprintln(w, "  --config, -c     config path")
}

func writeVersion(w io.Writer) {
	fmt.Fprintf(w, "ai-history %s", version)
	if commit != "" {
		fmt.Fprintf(w, " commit=%s", commit)
	}
	if buildDate != "" {
		fmt.Fprintf(w, " buildDate=%s", buildDate)
	}
	fmt.Fprintln(w)
}
