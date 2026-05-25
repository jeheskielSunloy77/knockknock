package signature

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type Signature struct {
	Vendor    string   `json:"vendor"`
	Signals   []string `json:"signals"`
	Challenge []string `json:"challenge,omitempty"`
}

var (
	slugOK        = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	leadingDotRun = regexp.MustCompile(`^\.[*+]\??`)
	sigPrefixes   = []string{"S:", "H:", "B:"}
)

func Compile(dir, outPath string) (string, error) {
	sigs, err := Load(dir)
	if err != nil {
		return "", err
	}
	pattern := buildRegex(sigs)
	if _, err := regexp.Compile(pattern); err != nil {
		return "", fmt.Errorf("assembled regex is invalid: %w", err)
	}
	if err := writeIfRequested(outPath, pattern); err != nil {
		return "", err
	}
	return pattern, nil
}

func CompileChallenge(dir, outPath string) (string, error) {
	sigs, err := Load(dir)
	if err != nil {
		return "", err
	}
	pattern := buildChallengeRegex(sigs)
	if _, err := regexp.Compile(pattern); err != nil {
		return "", fmt.Errorf("assembled challenge regex is invalid: %w", err)
	}
	if err := writeIfRequested(outPath, pattern); err != nil {
		return "", err
	}
	return pattern, nil
}

func Load(dir string) ([]Signature, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		return nil, fmt.Errorf("no signature files found in %s", dir)
	}

	seen := map[string]bool{}
	sigs := make([]Signature, 0, len(paths))
	for _, path := range paths {
		sig, err := validateFile(path)
		if err != nil {
			return nil, err
		}
		if seen[sig.Vendor] {
			return nil, fmt.Errorf("duplicate vendor slug: %s", sig.Vendor)
		}
		seen[sig.Vendor] = true
		sigs = append(sigs, sig)
	}
	sort.Slice(sigs, func(i, j int) bool { return sigs[i].Vendor < sigs[j].Vendor })
	return sigs, nil
}

func validateFile(path string) (Signature, error) {
	var sig Signature
	raw, err := os.ReadFile(path)
	if err != nil {
		return sig, err
	}
	name := filepath.Base(path)
	if err := json.Unmarshal(raw, &sig); err != nil {
		return sig, fmt.Errorf("%s: invalid JSON: %w", name, err)
	}
	if !slugOK.MatchString(sig.Vendor) {
		return sig, fmt.Errorf("%s: vendor must match [a-z][a-z0-9_]* (got %q)", name, sig.Vendor)
	}
	if stem := strings.TrimSuffix(name, ".json"); stem != sig.Vendor {
		return sig, fmt.Errorf("%s: filename stem must equal vendor slug %q", name, sig.Vendor)
	}
	if len(sig.Signals) == 0 {
		return sig, fmt.Errorf("%s: signals must be a non-empty array", name)
	}

	sigSet := make(map[string]bool, len(sig.Signals))
	for _, signal := range sig.Signals {
		if signal == "" {
			return sig, fmt.Errorf("%s: every signal must be non-empty", name)
		}
		if !hasSigPrefix(signal) {
			return sig, fmt.Errorf("%s: signal must start with S:/H:/B: : %q", name, signal)
		}
		if _, err := regexp.Compile(signal); err != nil {
			return sig, fmt.Errorf("%s: signal is not valid RE2: %q: %w", name, signal, err)
		}
		sigSet[signal] = true
	}
	for _, signal := range sig.Challenge {
		if !sigSet[signal] {
			return sig, fmt.Errorf("%s: challenge signal %q is not in signals", name, signal)
		}
	}
	return sig, nil
}

func hasSigPrefix(sig string) bool {
	for _, prefix := range sigPrefixes {
		if strings.HasPrefix(sig, prefix) {
			return true
		}
	}
	return false
}

func vendorGroup(vendor string, sigs []string) string {
	alts := make([]string, len(sigs))
	for i, sig := range sigs {
		if strings.HasPrefix(sig, "B:") {
			alts[i] = "(?:" + leadingDotRun.ReplaceAllString(sig[2:], "") + ")"
			continue
		}
		alts[i] = "^(?:" + sig + ")"
	}
	return "(?P<" + vendor + ">" + strings.Join(alts, "|") + ")"
}

func buildRegex(sigs []Signature) string {
	groups := make([]string, len(sigs))
	for i, sig := range sigs {
		groups[i] = vendorGroup(sig.Vendor, sig.Signals)
	}
	return "(?m)" + strings.Join(groups, "|")
}

func buildChallengeRegex(sigs []Signature) string {
	groups := make([]string, 0, len(sigs))
	for _, sig := range sigs {
		if len(sig.Challenge) == 0 {
			continue
		}
		groups = append(groups, vendorGroup(sig.Vendor, sig.Challenge))
	}
	if len(groups) == 0 {
		return "(?m)(?P<no_challenge_signatures>B:knockknock-placeholder-never-match)"
	}
	return "(?m)" + strings.Join(groups, "|")
}

func writeIfRequested(path, pattern string) error {
	if path == "" {
		return nil
	}
	return os.WriteFile(path, []byte(pattern+"\n"), 0o644)
}
