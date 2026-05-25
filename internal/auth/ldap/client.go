package ldap

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	ldapv3 "github.com/go-ldap/ldap/v3"
)

var (
	ErrUserNotFound         = errors.New("ldap: user not found")
	ErrInvalidCredentials   = errors.New("ldap: invalid credentials")
	ErrDirectoryUnavailable = errors.New("ldap: directory unavailable")
	ErrServiceBindFailed    = errors.New("ldap: service-account bind failed")
	ErrMissingEmail         = errors.New("ldap: user entry missing email attribute")
)

type Config struct {
	URL            string
	StartTLS       bool
	SkipTLSVerify  bool
	BindDN         string
	BindPassword   string
	SearchBase     string
	SearchFilter   string
	EmailAttribute string
	ConnectTimeout time.Duration
}

type User struct {
	DN    string
	Email string
}

const UsernamePlaceholder = "{username}"

const searchSizeLimit = 2

func AuthenticateAndFetch(cfg Config, login, password string) (*User, error) {
	if password == "" {
		return nil, ErrInvalidCredentials
	}

	conn, err := dial(cfg)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDirectoryUnavailable, err)
	}
	defer func() { _ = conn.Close() }()

	if bindErr := conn.Bind(cfg.BindDN, cfg.BindPassword); bindErr != nil {
		return nil, fmt.Errorf("%w: %w", ErrServiceBindFailed, bindErr)
	}

	filter := strings.ReplaceAll(cfg.SearchFilter, UsernamePlaceholder, ldapv3.EscapeFilter(login))

	req := ldapv3.NewSearchRequest(
		cfg.SearchBase,
		ldapv3.ScopeWholeSubtree,
		ldapv3.NeverDerefAliases,
		searchSizeLimit,
		0,
		false,
		filter,
		[]string{cfg.EmailAttribute},
		nil,
	)

	result, err := conn.Search(req)
	if err != nil {
		return nil, fmt.Errorf("ldap: search failed: %w", err)
	}
	if len(result.Entries) == 0 {
		return nil, ErrUserNotFound
	}
	if len(result.Entries) > 1 {
		return nil, fmt.Errorf("ldap: search filter matched %d entries", len(result.Entries))
	}

	entry := result.Entries[0]

	if userBindErr := conn.Bind(entry.DN, password); userBindErr != nil {
		if ldapv3.IsErrorWithCode(userBindErr, ldapv3.LDAPResultInvalidCredentials) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("ldap: user bind failed: %w", userBindErr)
	}

	email := entry.GetAttributeValue(cfg.EmailAttribute)
	if email == "" {
		return nil, ErrMissingEmail
	}

	return &User{
		DN:    entry.DN,
		Email: email,
	}, nil
}

func VerifyServiceBind(cfg Config) error {
	conn, err := dial(cfg)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrDirectoryUnavailable, err)
	}
	defer func() { _ = conn.Close() }()
	if bindErr := conn.Bind(cfg.BindDN, cfg.BindPassword); bindErr != nil {
		return fmt.Errorf("%w: %w", ErrServiceBindFailed, bindErr)
	}
	return nil
}

func dial(cfg Config) (*ldapv3.Conn, error) {
	parsed, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid ldap url %q: %w", cfg.URL, err)
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "ldap" && scheme != "ldaps" {
		return nil, fmt.Errorf("unsupported ldap scheme %q", parsed.Scheme)
	}

	tlsCfg := &tls.Config{
		ServerName:         parsed.Hostname(),
		InsecureSkipVerify: cfg.SkipTLSVerify, //nolint:gosec // opt-in flag.
		MinVersion:         tls.VersionTLS12,
	}

	opts := []ldapv3.DialOpt{}
	if cfg.ConnectTimeout > 0 {
		opts = append(opts, ldapv3.DialWithDialer(&net.Dialer{Timeout: cfg.ConnectTimeout}))
	}
	if scheme == "ldaps" {
		opts = append(opts, ldapv3.DialWithTLSConfig(tlsCfg))
	}

	conn, err := ldapv3.DialURL(cfg.URL, opts...)
	if err != nil {
		return nil, err
	}
	if cfg.ConnectTimeout > 0 {
		conn.SetTimeout(cfg.ConnectTimeout)
	}

	if scheme == "ldap" && cfg.StartTLS {
		if startErr := conn.StartTLS(tlsCfg); startErr != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("starttls failed: %w", startErr)
		}
	}

	return conn, nil
}
