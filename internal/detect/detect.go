package detect

import (
	_ "embed"
	"regexp"
	"sort"
	"strings"
)

const DefaultBodyCap = 64 * 1024

//go:embed regex_presence.re2.txt
var PresenceRegexText string

//go:embed regex_challenge.re2.txt
var ChallengeRegexText string

type Verdict struct {
	Protected bool     `json:"protected"`
	Vendors   []string `json:"vendors"`
	Mode      string   `json:"mode"`
	Source    string   `json:"source,omitempty"`
}

func Detect(raw []byte, regexText string, bodyCap int) (Verdict, error) {
	if bodyCap <= 0 {
		bodyCap = DefaultBodyCap
	}
	re, err := regexp.Compile(strings.TrimSpace(regexText))
	if err != nil {
		return Verdict{}, err
	}
	vendors := Match(raw, re, bodyCap)
	return Verdict{Protected: len(vendors) > 0, Vendors: vendors}, nil
}

func Match(raw []byte, re *regexp.Regexp, bodyCap int) []string {
	norm := Normalize(raw, bodyCap)
	names := re.SubexpNames()
	seen := map[string]bool{}
	for _, m := range re.FindAllStringSubmatch(norm, -1) {
		for i, group := range m {
			if i > 0 && group != "" && names[i] != "" {
				seen[names[i]] = true
			}
		}
	}
	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

var (
	statusRe      = regexp.MustCompile(`HTTP/[\d.]+\s+(\d{3})`)
	isStatusLine  = regexp.MustCompile(`^HTTP/\d`)
	stripBodyChar = strings.NewReplacer("\n", " ", "\t", " ")
)

func Normalize(raw []byte, bodyCap int) string {
	if bodyCap <= 0 {
		bodyCap = DefaultBodyCap
	}
	text := latin1(raw)
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	lines := strings.Split(text, "\n")

	var statuses, headers, bodyLines []string
	idx, n, sawBlock := 0, len(lines), false
	for idx < n {
		if isStatusLine.MatchString(lines[idx]) {
			sawBlock = true
			if m := statusRe.FindStringSubmatch(lines[idx]); m != nil {
				statuses = append(statuses, "S:"+m[1])
			}
			idx++
			for idx < n && lines[idx] != "" {
				if name, val, ok := strings.Cut(lines[idx], ":"); ok {
					headers = append(headers, "H:"+strings.ToLower(strings.TrimSpace(name))+":"+strings.TrimSpace(val))
				}
				idx++
			}
			idx++
			start := idx
			for idx < n && !isStatusLine.MatchString(lines[idx]) {
				idx++
			}
			bodyLines = lines[start:idx]
			continue
		}
		if !sawBlock {
			bodyLines = lines[idx:]
		}
		break
	}

	out := make([]string, 0, len(statuses)+len(headers)+1)
	out = append(out, statuses...)
	out = append(out, headers...)
	body := strings.Join(bodyLines, "\n")
	if r := []rune(body); len(r) > bodyCap {
		body = string(r[:bodyCap])
	}
	out = append(out, "B:"+stripBodyChar.Replace(body))
	return strings.Join(out, "\n")
}

func latin1(b []byte) string {
	var sb strings.Builder
	sb.Grow(len(b))
	for _, c := range b {
		sb.WriteRune(rune(c))
	}
	return sb.String()
}
