package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jeheskielSunloy77/knockknock/internal/detect"
	"github.com/jeheskielSunloy77/knockknock/internal/fetch"
	"github.com/jeheskielSunloy77/knockknock/internal/signature"
)

var protectedTemplates = []string{
	"who's there? {vendors}",
	"who's there? {vendors}",
	"who's there? {vendors}",
}

var notProtectedTemplates = []string{
	"who's there? {answer}",
	"who's there? {answer}",
	"who's there? {answer}",
}

type Config struct {
	Args    []string
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
	Version string
}

type options struct {
	challenge bool
	naive     bool
	json      bool
	profile   string
	timeout   time.Duration
	bodyCap   int
	url       string
}

var banner = "\n" +
	"██╗  ██╗███╗   ██╗ ██████╗  ██████╗██╗  ██╗██╗  ██╗███╗   ██╗ ██████╗  ██████╗██╗  ██╗\n" +
	"██║ ██╔╝████╗  ██║██╔═══██╗██╔════╝██║ ██╔╝██║ ██╔╝████╗  ██║██╔═══██╗██╔════╝██║ ██╔╝\n" +
	"█████╔╝ ██╔██╗ ██║██║   ██║██║     █████╔╝ █████╔╝ ██╔██╗ ██║██║   ██║██║     █████╔╝ \n" +
	"██╔═██╗ ██║╚██╗██║██║   ██║██║     ██╔═██╗ ██╔═██╗ ██║╚██╗██║██║   ██║██║     ██╔═██╗ \n" +
	"██║  ██╗██║ ╚████║╚██████╔╝╚██████╗██║  ██╗██║  ██╗██║ ╚████║╚██████╔╝╚██████╗██║  ██╗\n" +
	"╚═╝  ╚═╝╚═╝  ╚═══╝ ╚═════╝  ╚═════╝╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═══╝ ╚═════╝  ╚═════╝╚═╝  ╚═╝\n" +
	"\n" +
	"detect antibot, WAF, and CAPTCHA protections on web pages\n" +
	"\n" +
	"examples:\n" +
	"  knockknock https://example.com\n" +
	"  curl -isS -L https://example.com | knockknock\n"

func printBanner(w io.Writer) {
	fmt.Fprint(w, banner)
}

const usage = `knockknock - detect antibot/WAF/CAPTCHA protection on a page.

Usage:
  knockknock https://example.com
  curl -isS -L https://example.com | knockknock

Options:
  -c, --challenge    only report active challenge/block signals
  -n, --naive        fetch with Go's default TLS/HTTP fingerprint
  -p, --profile P    tls-client browser profile for URL fetches (default %s)
      --json         emit machine-readable JSON
      --timeout D    fetch timeout (default 30s)
      --max-body N   max body bytes to scan; supports KiB/MiB suffixes (default 64KiB)
  -h, --help         show help and exit
  -v, --version      show version and exit

Commands:
  skill [install|show] [-g|--global] [--dir path]
          install or inspect the AI agent skill
  compile [--dir signatures] [--out internal/detect/regex_presence.re2.txt]
          [--challenge-out internal/detect/regex_challenge.re2.txt]
`

func Run(cfg Config) int {
	if cfg.Stdin == nil {
		cfg.Stdin = strings.NewReader("")
	}
	if cfg.Stdout == nil {
		cfg.Stdout = io.Discard
	}
	if cfg.Stderr == nil {
		cfg.Stderr = io.Discard
	}

	if len(cfg.Args) > 0 && (cfg.Args[0] == "skill" || cfg.Args[0] == "install-skill") {
		return runSkill(cfg, cfg.Args)
	}

	if len(cfg.Args) > 0 && cfg.Args[0] == "compile" {
		return runCompile(cfg, cfg.Args[1:])
	}

	opts, action, err := parseOptions(cfg.Args)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "knockknock: %v\n", err)
		return 2
	}
	switch action {
	case "help":
		fmt.Fprintf(cfg.Stdout, usage, fetch.DefaultProfile)
		return 0
	case "version":
		fmt.Fprintf(cfg.Stdout, "knockknock %s\n", cfg.Version)
		return 0
	}

	if opts.url == "" {
		if f, ok := cfg.Stdin.(*os.File); ok {
			if stat, err := f.Stat(); err == nil && (stat.Mode()&os.ModeCharDevice) != 0 {
				printBanner(cfg.Stdout)
				return 0
			}
		}
	}

	regexText := detect.PresenceRegexText
	mode := "presence"
	if opts.challenge {
		regexText = detect.ChallengeRegexText
		mode = "challenge"
	}

	source := "stdin"
	var raw []byte
	if opts.url != "" {
		source = opts.url
		raw, err = fetch.Fetch(opts.url, fetch.Options{
			Profile: opts.profile,
			Naive:   opts.naive,
			Timeout: opts.timeout,
		})
	} else {
		raw, err = io.ReadAll(cfg.Stdin)
	}
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "knockknock: %v\n", err)
		return 2
	}

	verdict, err := detect.Detect(raw, regexText, opts.bodyCap)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "knockknock: embedded regex is invalid: %v\n", err)
		return 2
	}
	verdict.Mode = mode
	verdict.Source = source
	if err := writeVerdict(cfg.Stdout, verdict, opts.json); err != nil {
		fmt.Fprintf(cfg.Stderr, "knockknock: writing output: %v\n", err)
		return 2
	}
	if verdict.Protected {
		return 0
	}
	return 1
}

func parseOptions(args []string) (options, string, error) {
	opts := options{
		profile: fetch.DefaultProfile,
		timeout: fetch.DefaultTimeout,
		bodyCap: detect.DefaultBodyCap,
	}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help":
			return opts, "help", nil
		case arg == "-v" || arg == "--version":
			return opts, "version", nil
		case arg == "-c" || arg == "--challenge":
			opts.challenge = true
		case arg == "-n" || arg == "--naive":
			opts.naive = true
		case arg == "--json":
			opts.json = true
		case arg == "-p" || arg == "--profile":
			i++
			if i >= len(args) {
				return opts, "", errors.New("--profile requires a value")
			}
			opts.profile = args[i]
		case arg == "--timeout":
			i++
			if i >= len(args) {
				return opts, "", errors.New("--timeout requires a value")
			}
			d, err := time.ParseDuration(args[i])
			if err != nil || d <= 0 {
				return opts, "", fmt.Errorf("invalid --timeout value %q", args[i])
			}
			opts.timeout = d
		case arg == "--max-body":
			i++
			if i >= len(args) {
				return opts, "", errors.New("--max-body requires a value")
			}
			n, err := parseByteSize(args[i])
			if err != nil || n <= 0 {
				return opts, "", fmt.Errorf("invalid --max-body value %q", args[i])
			}
			opts.bodyCap = n
		case strings.HasPrefix(arg, "-") && arg != "-":
			return opts, "", fmt.Errorf("unknown option %q (try --help)", arg)
		default:
			if opts.url != "" {
				return opts, "", fmt.Errorf("unexpected extra argument %q", arg)
			}
			opts.url = arg
		}
	}
	if opts.naive && opts.profile != fetch.DefaultProfile {
		return opts, "", errors.New("--naive and --profile are mutually exclusive")
	}
	return opts, "", nil
}

func parseByteSize(raw string) (int, error) {
	s := strings.TrimSpace(raw)
	lower := strings.ToLower(s)
	multiplier := 1
	for suffix, mult := range map[string]int{
		"kib": 1024,
		"kb":  1000,
		"mib": 1024 * 1024,
		"mb":  1000 * 1000,
	} {
		if strings.HasSuffix(lower, suffix) {
			multiplier = mult
			s = strings.TrimSpace(s[:len(s)-len(suffix)])
			break
		}
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	return n * multiplier, nil
}

const (
	colorBold  = "\033[1m"
	colorRed   = "\033[31m"
	colorGreen = "\033[32m"

	colorReset = "\033[0m"
)

func writeVerdict(w io.Writer, verdict detect.Verdict, jsonOut bool) error {
	if jsonOut {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(verdict)
	}
	if verdict.Protected {
		vendors := formatVendors(verdict.Vendors)
		if isTerminal(w) {
			vendors = colorBold + colorRed + vendors + colorReset
		}
		tmpl := protectedTemplates[rand.Intn(len(protectedTemplates))]
		_, err := fmt.Fprintln(w, strings.ReplaceAll(tmpl, "{vendors}", vendors))
		return err
	}
	tmpl := notProtectedTemplates[rand.Intn(len(notProtectedTemplates))]
	answer := "No one"
	if isTerminal(w) {
		answer = colorGreen + answer + colorReset
	}
	_, err := fmt.Fprintln(w, strings.ReplaceAll(tmpl, "{answer}", answer))
	return err
}

func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return stat.Mode()&os.ModeCharDevice != 0
}

func formatVendors(vendors []string) string {
	switch len(vendors) {
	case 0:
		return ""
	case 1:
		return vendors[0]
	case 2:
		return vendors[0] + " and " + vendors[1]
	default:
		return strings.Join(vendors[:len(vendors)-1], ", ") + ", and " + vendors[len(vendors)-1]
	}
}

func runCompile(cfg Config, args []string) int {
	dir := "signatures"
	out := "internal/detect/regex_presence.re2.txt"
	challengeOut := "internal/detect/regex_challenge.re2.txt"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--dir":
			i++
			if i >= len(args) {
				fmt.Fprintln(cfg.Stderr, "compile: --dir requires a value")
				return 2
			}
			dir = args[i]
		case "--out":
			i++
			if i >= len(args) {
				fmt.Fprintln(cfg.Stderr, "compile: --out requires a value")
				return 2
			}
			out = args[i]
		case "--challenge-out":
			i++
			if i >= len(args) {
				fmt.Fprintln(cfg.Stderr, "compile: --challenge-out requires a value")
				return 2
			}
			challengeOut = args[i]
		default:
			fmt.Fprintf(cfg.Stderr, "compile: unknown argument %q\n", args[i])
			return 2
		}
	}
	pattern, err := signature.Compile(dir, out)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "compile: %v\n", err)
		return 1
	}
	fmt.Fprintf(cfg.Stderr, "compile: wrote %s (%d bytes)\n", out, len(pattern))

	challengePattern, err := signature.CompileChallenge(dir, challengeOut)
	if err != nil {
		fmt.Fprintf(cfg.Stderr, "compile: %v\n", err)
		return 1
	}
	fmt.Fprintf(cfg.Stderr, "compile: wrote %s (%d bytes)\n", challengeOut, len(challengePattern))
	return 0
}
