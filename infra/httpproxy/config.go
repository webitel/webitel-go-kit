package httpproxy

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"

	xproxy "golang.org/x/net/http/httpproxy"
)

// Config is the effective set of forward-proxy rules for outbound HTTP
// requests. Field semantics match the conventional HTTP_PROXY / HTTPS_PROXY /
// NO_PROXY environment variables (see golang.org/x/net/http/httpproxy):
// an empty value means "no proxy" for that class of requests.
type Config struct {
	// HTTPProxy is the proxy URL used for http:// requests.
	// A bare "host:port" is accepted and treated as an http:// URL.
	HTTPProxy string `yaml:"http_proxy" json:"http_proxy"`
	// HTTPSProxy is the proxy URL used for https:// requests.
	HTTPSProxy string `yaml:"https_proxy" json:"https_proxy"`
	// NoProxy is a comma-separated list of hosts, domain suffixes
	// (".example.com"), IPs and CIDRs ("10.0.0.0/8") that are always
	// reached directly, bypassing the proxy. A single "*" disables
	// proxying entirely.
	NoProxy string `yaml:"no_proxy" json:"no_proxy"`
}

// FromEnvironment returns the proxy settings of the current process
// environment: HTTP_PROXY, HTTPS_PROXY and NO_PROXY, or their lowercase
// variants. Note that the process environment cannot change at runtime;
// for on-the-fly updates use Manager.WatchFile or Manager.Update.
func FromEnvironment() Config {
	env := xproxy.FromEnvironment()
	return Config{
		HTTPProxy:  env.HTTPProxy,
		HTTPSProxy: env.HTTPSProxy,
		NoProxy:    env.NoProxy,
	}
}

// Validate reports whether the settings are usable. It exists because
// golang.org/x/net/http/httpproxy silently tolerates malformed values,
// which would turn a typo in the config into "no proxy" or a garbage host —
// Manager.Update rejects such configs instead, keeping the last good
// settings. It is deliberately stricter than x/net: proxy URLs must carry a
// non-empty host, and no_proxy entries that look like CIDRs or IPs must
// parse as such. Error messages never contain proxy credentials.
func (c Config) Validate() error {
	if _, err := parseProxyURL(c.HTTPProxy); err != nil {
		return fmt.Errorf("httpproxy: http_proxy: %w", err)
	}
	if _, err := parseProxyURL(c.HTTPSProxy); err != nil {
		return fmt.Errorf("httpproxy: https_proxy: %w", err)
	}
	if err := validateNoProxy(c.NoProxy); err != nil {
		return fmt.Errorf("httpproxy: no_proxy: %w", err)
	}
	return nil
}

// proxyFunc compiles the config into a per-URL proxy resolver with the
// standard no_proxy matching semantics.
func (c Config) proxyFunc() func(*url.URL) (*url.URL, error) {
	x := xproxy.Config{
		HTTPProxy:  c.HTTPProxy,
		HTTPSProxy: c.HTTPSProxy,
		NoProxy:    c.NoProxy,
		CGI:        os.Getenv("REQUEST_METHOD") != "",
	}
	return x.ProxyFunc()
}

// parseProxyURL mirrors the tolerant parsing of
// golang.org/x/net/http/httpproxy — an empty value means no proxy and a bare
// "host:port" is treated as an http:// proxy URL — and additionally rejects
// addresses without a host, which x/net would silently turn into a broken
// proxy.
func parseProxyURL(addr string) (*url.URL, error) {
	if addr == "" {
		return nil, nil
	}
	if strings.ContainsAny(addr, " \t\r\n") {
		return nil, fmt.Errorf("invalid proxy address %q: contains whitespace", redactAddr(addr))
	}
	proxyURL, err := url.Parse(addr)
	if err != nil || proxyURL.Scheme == "" || proxyURL.Host == "" {
		proxyURL, err = url.Parse("http://" + addr)
	}
	if err != nil {
		// Unwrap url.Error: its message embeds the raw URL, credentials
		// included.
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			err = urlErr.Err
		}
		return nil, fmt.Errorf("invalid proxy address %q: %w", redactAddr(addr), err)
	}
	if proxyURL.Hostname() == "" || strings.HasSuffix(proxyURL.Host, ":") {
		return nil, fmt.Errorf("invalid proxy address %q: missing host", redactAddr(addr))
	}
	return proxyURL, nil
}

// validateNoProxy rejects the realistic typos that x/net would silently
// drop: entries with "/" must be valid CIDRs, and entries that look like an
// IP must parse as one. Hostnames and domain suffixes are passed through
// untouched — x/net accepts almost anything there.
func validateNoProxy(list string) error {
	for _, entry := range strings.Split(list, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" || entry == "*" {
			continue
		}
		if strings.Contains(entry, "/") {
			if _, _, err := net.ParseCIDR(entry); err != nil {
				return fmt.Errorf("entry %q: invalid CIDR", entry)
			}
			continue
		}
		host := entry
		if h, _, err := net.SplitHostPort(entry); err == nil {
			host = h
		}
		if looksLikeIP(host) && net.ParseIP(host) == nil {
			return fmt.Errorf("entry %q: invalid IP address", entry)
		}
	}
	return nil
}

// looksLikeIP reports whether s consists only of digits, dots and colons —
// i.e. was meant to be an IP address rather than a hostname.
func looksLikeIP(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if (r < '0' || r > '9') && r != '.' && r != ':' {
			return false
		}
	}
	return true
}

// maskProxyURL renders a proxy URL for logging, hiding credentials.
func maskProxyURL(addr string) string {
	if addr == "" {
		return ""
	}
	proxyURL, err := parseProxyURL(addr)
	if err != nil || proxyURL == nil {
		return redactAddr(addr)
	}
	if proxyURL.User != nil {
		proxyURL.User = nil
		masked := proxyURL.String()
		if i := strings.Index(masked, "://"); i >= 0 {
			masked = masked[:i+3] + "***@" + masked[i+3:]
		}
		return masked
	}
	return proxyURL.String()
}

// redactAddr strips any userinfo from a possibly-malformed address, for use
// in error messages and logs where the address cannot be parsed.
func redactAddr(addr string) string {
	at := strings.LastIndex(addr, "@")
	if at < 0 {
		return addr
	}
	start := 0
	if i := strings.Index(addr, "://"); i >= 0 {
		start = i + 3
	}
	if at < start {
		return addr
	}
	return addr[:start] + "***" + addr[at:]
}
