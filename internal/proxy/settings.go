package proxy

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

const (
	SchemeSOCKS4 = "socks4"
	SchemeSOCKS5 = "socks5"
)

// Settings describes a remote SOCKS4 or SOCKS5 endpoint.
type Settings struct {
	Scheme   string
	Host     string
	Port     int
	Username string
	Password string
}

// Parse accepts SOCKS URLs and the legacy host:port:user:password form.
// IPv6 addresses in the legacy form must be enclosed in square brackets.
func Parse(input string) (Settings, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return Settings{}, errors.New("SOCKS proxy address is not specified")
	}

	if strings.Contains(input, "://") {
		return parseURL(input)
	}
	return parseLegacy(input)
}

// New validates individual proxy fields.
func New(host string, port int, username, password string) (Settings, error) {
	return NewForScheme(SchemeSOCKS5, host, port, username, password)
}

// NewForScheme validates individual proxy fields for a specific protocol.
func NewForScheme(scheme, host string, port int, username, password string) (Settings, error) {
	scheme = normalizeScheme(scheme)
	if scheme != SchemeSOCKS4 && scheme != SchemeSOCKS5 {
		return Settings{}, fmt.Errorf("only socks4:// and socks5:// schemes are supported, got %q", scheme)
	}
	host = strings.TrimSpace(host)
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		host = strings.TrimSuffix(strings.TrimPrefix(host, "["), "]")
	}
	if host == "" {
		return Settings{}, errors.New("SOCKS proxy address is not specified")
	}
	if strings.ContainsAny(host, " /\\\t\r\n") {
		return Settings{}, fmt.Errorf("invalid proxy address %q", host)
	}
	if port < 1 || port > 65535 {
		return Settings{}, fmt.Errorf("proxy port must be in the range 1..65535, got %d", port)
	}
	if username == "" && password != "" {
		return Settings{}, errors.New("a password was provided without a username")
	}
	if scheme == SchemeSOCKS4 && password != "" {
		return Settings{}, errors.New("SOCKS4 supports a user ID but not a password")
	}
	return Settings{Scheme: scheme, Host: host, Port: port, Username: username, Password: password}, nil
}

func parseURL(input string) (Settings, error) {
	u, err := url.Parse(input)
	if err != nil {
		return Settings{}, fmt.Errorf("could not parse proxy URL: %w", err)
	}
	scheme := normalizeScheme(u.Scheme)
	if scheme != SchemeSOCKS4 && scheme != SchemeSOCKS5 {
		return Settings{}, fmt.Errorf("only socks4:// and socks5:// schemes are supported, got %q", u.Scheme)
	}
	if u.Path != "" && u.Path != "/" || u.RawQuery != "" || u.Fragment != "" {
		return Settings{}, errors.New("SOCKS proxy URL must not contain a path, query parameters, or fragment")
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil || u.Port() == "" {
		return Settings{}, errors.New("SOCKS proxy URL must include a numeric port")
	}
	username, password := "", ""
	if u.User != nil {
		username = u.User.Username()
		password, _ = u.User.Password()
	}
	return NewForScheme(scheme, u.Hostname(), port, username, password)
}

func parseLegacy(input string) (Settings, error) {
	host := ""
	rest := ""
	if strings.HasPrefix(input, "[") {
		end := strings.Index(input, "]:")
		if end < 0 {
			return Settings{}, errors.New("IPv6 address must have the form [2001:db8::1]:1080:user:password")
		}
		host = input[1:end]
		rest = input[end+2:]
	} else {
		parts := strings.SplitN(input, ":", 2)
		if len(parts) != 2 {
			return Settings{}, errors.New("expected host:port or host:port:user:password format")
		}
		host, rest = parts[0], parts[1]
	}

	parts := strings.SplitN(rest, ":", 3)
	if len(parts) < 1 || parts[0] == "" {
		return Settings{}, errors.New("proxy port is not specified")
	}
	port, err := strconv.Atoi(parts[0])
	if err != nil {
		return Settings{}, fmt.Errorf("proxy port %q is not a number", parts[0])
	}
	username, password := "", ""
	if len(parts) >= 2 {
		username = parts[1]
	}
	if len(parts) == 3 {
		password = parts[2]
	}
	return NewForScheme(SchemeSOCKS5, host, port, username, password)
}

// URL returns a safely escaped URL accepted by tun2socks.
func (s Settings) URL() string {
	scheme := normalizeScheme(s.Scheme)
	if scheme == "" {
		scheme = SchemeSOCKS5
	}
	u := &url.URL{Scheme: scheme, Host: net.JoinHostPort(s.Host, strconv.Itoa(s.Port))}
	if s.Username != "" {
		if scheme == SchemeSOCKS4 {
			u.User = url.User(s.Username)
		} else {
			u.User = url.UserPassword(s.Username, s.Password)
		}
	}
	return u.String()
}

func normalizeScheme(scheme string) string {
	switch strings.ToLower(strings.TrimSpace(scheme)) {
	case "socks4", "socks4a":
		return SchemeSOCKS4
	case "", "socks5", "socks5h":
		return SchemeSOCKS5
	default:
		return strings.ToLower(strings.TrimSpace(scheme))
	}
}

// RedactedURL is suitable for logs.
func (s Settings) RedactedURL() string {
	copy := s
	if copy.Username != "" {
		copy.Password = "***"
	}
	return copy.URL()
}
