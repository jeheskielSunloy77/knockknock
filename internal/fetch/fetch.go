package fetch

import (
	"bytes"
	"fmt"
	"io"
	stdhttp "net/http"
	"net/url"
	"strings"
	"time"

	http "github.com/bogdanfinn/fhttp"
	tlsclient "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
)

const DefaultProfile = "chrome_133"
const DefaultTimeout = 30 * time.Second
const maxRedirects = 10

type Options struct {
	Profile string
	Naive   bool
	Timeout time.Duration
}

type hopResponse struct {
	proto      string
	statusCode int
	status     string
	header     map[string][]string
	body       []byte
}

type doer func(rawURL string) (*hopResponse, error)

func Fetch(rawURL string, opts Options) ([]byte, error) {
	if opts.Timeout <= 0 {
		opts.Timeout = DefaultTimeout
	}
	if opts.Profile == "" {
		opts.Profile = DefaultProfile
	}

	var (
		do  doer
		err error
	)
	if opts.Naive {
		do, err = naiveDoer(opts.Timeout)
	} else {
		do, err = browserDoer(opts.Profile, opts.Timeout)
	}
	if err != nil {
		return nil, err
	}
	return fetchChain(rawURL, do)
}

func fetchChain(rawURL string, do doer) ([]byte, error) {
	var chain bytes.Buffer
	current := rawURL
	for hop := 0; ; hop++ {
		resp, err := do(current)
		if err != nil {
			return nil, err
		}
		writeRawResponse(&chain, resp)

		loc := HeaderGet(resp.header, "Location")
		if !IsRedirect(resp.statusCode) || loc == "" || hop >= maxRedirects {
			break
		}
		next, err := ResolveLocation(current, loc)
		if err != nil {
			break
		}
		current = next
	}
	return chain.Bytes(), nil
}

func browserDoer(profileName string, timeout time.Duration) (doer, error) {
	profile, ok := profiles.MappedTLSClients[profileName]
	if !ok {
		return nil, fmt.Errorf("unknown profile %q", profileName)
	}
	client, err := tlsclient.NewHttpClient(tlsclient.NewNoopLogger(),
		tlsclient.WithClientProfile(profile),
		tlsclient.WithTimeoutSeconds(int(timeout/time.Second)),
		tlsclient.WithCookieJar(tlsclient.NewCookieJar()),
		tlsclient.WithNotFollowRedirects(),
	)
	if err != nil {
		return nil, err
	}
	return func(rawURL string) (*hopResponse, error) {
		req, err := http.NewRequest(http.MethodGet, rawURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header = chromeRequestHeader()
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetching %s: %w", rawURL, err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("reading body of %s: %w", rawURL, err)
		}
		return &hopResponse{resp.Proto, resp.StatusCode, resp.Status, map[string][]string(resp.Header), body}, nil
	}, nil
}

func naiveDoer(timeout time.Duration) (doer, error) {
	client := &stdhttp.Client{
		Timeout: timeout,
		CheckRedirect: func(*stdhttp.Request, []*stdhttp.Request) error {
			return stdhttp.ErrUseLastResponse
		},
	}
	return func(rawURL string) (*hopResponse, error) {
		resp, err := client.Get(rawURL)
		if err != nil {
			return nil, fmt.Errorf("fetching %s: %w", rawURL, err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("reading body of %s: %w", rawURL, err)
		}
		return &hopResponse{resp.Proto, resp.StatusCode, resp.Status, map[string][]string(resp.Header), body}, nil
	}, nil
}

func chromeRequestHeader() http.Header {
	return http.Header{
		"sec-ch-ua":                 {`"Chromium";v="133", "Google Chrome";v="133", "Not?A_Brand";v="99"`},
		"sec-ch-ua-mobile":          {"?0"},
		"sec-ch-ua-platform":        {`"Windows"`},
		"upgrade-insecure-requests": {"1"},
		"user-agent":                {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36"},
		"accept":                    {"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7"},
		"sec-fetch-site":            {"none"},
		"sec-fetch-mode":            {"navigate"},
		"sec-fetch-user":            {"?1"},
		"sec-fetch-dest":            {"document"},
		"accept-encoding":           {"gzip, deflate, br, zstd"},
		"accept-language":           {"en-US,en;q=0.9"},
		"priority":                  {"u=0, i"},
		http.HeaderOrderKey: {
			"sec-ch-ua", "sec-ch-ua-mobile", "sec-ch-ua-platform",
			"upgrade-insecure-requests", "user-agent", "accept",
			"sec-fetch-site", "sec-fetch-mode", "sec-fetch-user", "sec-fetch-dest",
			"accept-encoding", "accept-language", "priority",
		},
	}
}

func writeRawResponse(buf *bytes.Buffer, resp *hopResponse) {
	proto := resp.proto
	if proto == "" {
		proto = "HTTP/1.1"
	}
	status := resp.status
	if status == "" {
		status = fmt.Sprintf("%d %s", resp.statusCode, stdhttp.StatusText(resp.statusCode))
	}
	fmt.Fprintf(buf, "%s %s\r\n", proto, status)
	for key, vals := range resp.header {
		if key == "" || key[0] == ':' {
			continue
		}
		for _, val := range vals {
			fmt.Fprintf(buf, "%s: %s\r\n", key, val)
		}
	}
	buf.WriteString("\r\n")
	buf.Write(resp.body)
	buf.WriteString("\r\n")
}

func HeaderGet(headers map[string][]string, key string) string {
	if values := headers[key]; len(values) > 0 {
		return values[0]
	}
	for k, values := range headers {
		if strings.EqualFold(k, key) && len(values) > 0 {
			return values[0]
		}
	}
	return ""
}

func IsRedirect(code int) bool {
	switch code {
	case 301, 302, 303, 307, 308:
		return true
	default:
		return false
	}
}

func ResolveLocation(base, loc string) (string, error) {
	b, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	l, err := url.Parse(loc)
	if err != nil {
		return "", err
	}
	return b.ResolveReference(l).String(), nil
}
