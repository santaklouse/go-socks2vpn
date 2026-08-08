// Package deeplink parses go-socks2vpn configuration links.
package deeplink

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	proxyconfig "github.com/santaklouse/go-socks2vpn/internal/proxy"
)

const (
	Scheme       = "socks2vpn"
	LegacyScheme = "socks2vps"
)

// Parse accepts links in the form:
//
//	socks2vpn://socks5-username:password@proxy.example:1080
//
// The socks2vps scheme is accepted as a backwards-compatible alias. Parsing a
// link never connects to the proxy and never persists its password.
func Parse(raw string) (proxyconfig.Settings, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return proxyconfig.Settings{}, errors.New("invalid go-socks2vpn link")
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != Scheme && scheme != LegacyScheme {
		return proxyconfig.Settings{}, fmt.Errorf("unsupported link scheme %q", u.Scheme)
	}
	if u.Opaque != "" || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" {
		return proxyconfig.Settings{}, errors.New("go-socks2vpn link must not contain a path, query, or fragment")
	}
	if u.User == nil {
		return proxyconfig.Settings{}, errors.New("go-socks2vpn link must specify socks4 or socks5 before the proxy host")
	}

	identity := u.User.Username()
	proxyScheme, username, err := parseIdentity(identity)
	if err != nil {
		return proxyconfig.Settings{}, err
	}
	password, passwordSet := u.User.Password()
	if proxyScheme == proxyconfig.SchemeSOCKS4 && passwordSet {
		return proxyconfig.Settings{}, errors.New("SOCKS4 links must not contain a password")
	}
	if username == "" && password != "" {
		return proxyconfig.Settings{}, errors.New("a deep-link password requires a proxy username")
	}

	port, err := strconv.Atoi(u.Port())
	if err != nil || u.Port() == "" {
		return proxyconfig.Settings{}, errors.New("go-socks2vpn link must contain a numeric proxy port")
	}
	return proxyconfig.NewForScheme(proxyScheme, u.Hostname(), port, username, password)
}

func parseIdentity(identity string) (scheme, username string, err error) {
	lower := strings.ToLower(identity)
	switch {
	case lower == "socks5":
		return proxyconfig.SchemeSOCKS5, "", nil
	case strings.HasPrefix(lower, "socks5-"):
		return proxyconfig.SchemeSOCKS5, identity[len("socks5-"):], nil
	case lower == "socks4":
		return proxyconfig.SchemeSOCKS4, "", nil
	case strings.HasPrefix(lower, "socks4-"):
		return proxyconfig.SchemeSOCKS4, identity[len("socks4-"):], nil
	default:
		return "", "", errors.New("proxy credentials must start with socks4 or socks5")
	}
}
