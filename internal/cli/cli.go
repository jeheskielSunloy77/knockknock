package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/jeheskielSunloy77/knockknock/internal/detect"
	"github.com/jeheskielSunloy77/knockknock/internal/fetch"
	"github.com/jeheskielSunloy77/knockknock/internal/signature"
)

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
  -V, --version      show version and exit

Commands:
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
		case arg == "-V" || arg == "--version":
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

func writeVerdict(w io.Writer, verdict detect.Verdict, jsonOut bool) error {
	if jsonOut {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(verdict)
	}
	if verdict.Protected {
		if _, err := fmt.Fprintln(w, "protected: yes"); err != nil {
			return err
		}
		_, err := fmt.Fprintf(w, "vendors: %s\n", strings.Join(verdict.Vendors, ", "))
		return err
	}
	_, err := fmt.Fprintln(w, "protected: no")
	return err
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
