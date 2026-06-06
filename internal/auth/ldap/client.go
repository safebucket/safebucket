package ldap

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrServiceUnavailable = errors.New("ldap service unavailable")
)

const (
	defaultEmailAttribute   = "mail"
	defaultConnectTimeoutMS = 5000
)

type Config struct {
	URL              string
	BindDN           string
	BindPassword     string
	BaseDN           string
	UserFilter       string
	AttributeMap     AttributeMap
	StartTLS         bool
	TLSInsecureSkip  bool
	ConnectTimeoutMS int
}

type AttributeMap struct {
	Email string
}

type User struct {
	DN    string
	Email string
}

func AuthenticateAndFetch(cfg Config, username, password string) (User, error) {
	conn, err := dial(cfg)
	if err != nil {
		return User{}, fmt.Errorf("%w: %w", ErrServiceUnavailable, err)
	}
	defer conn.Close()

	if err = conn.Bind(cfg.BindDN, cfg.BindPassword); err != nil {
		return User{}, fmt.Errorf("%w: service bind failed: %w", ErrServiceUnavailable, err)
	}

	emailAttr := cfg.AttributeMap.Email
	if emailAttr == "" {
		emailAttr = defaultEmailAttribute
	}

	filter := fmt.Sprintf(cfg.UserFilter, ldap.EscapeFilter(username))
	searchReq := ldap.NewSearchRequest(
		cfg.BaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		1, // size limit
		0, // time limit
		false,
		filter,
		[]string{"dn", emailAttr},
		nil,
	)

	result, err := conn.Search(searchReq)
	if err != nil {
		return User{}, fmt.Errorf("%w: user search failed: %w", ErrServiceUnavailable, err)
	}
	if len(result.Entries) == 0 {
		return User{}, ErrInvalidCredentials
	}

	entry := result.Entries[0]
	userDN := entry.DN
	email := entry.GetAttributeValue(emailAttr)

	if err = conn.Bind(userDN, password); err != nil {
		var ldapErr *ldap.Error
		if errors.As(err, &ldapErr) && ldapErr.ResultCode == ldap.LDAPResultInvalidCredentials {
			return User{}, ErrInvalidCredentials
		}
		return User{}, fmt.Errorf("%w: user bind failed: %w", ErrServiceUnavailable, err)
	}

	if strings.TrimSpace(email) == "" {
		return User{}, fmt.Errorf("%w: user entry is missing the %q attribute", ErrServiceUnavailable, emailAttr)
	}

	return User{DN: userDN, Email: email}, nil
}

func VerifyServiceBind(cfg Config) error {
	conn, err := dial(cfg)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrServiceUnavailable, err)
	}
	defer conn.Close()

	if err = conn.Bind(cfg.BindDN, cfg.BindPassword); err != nil {
		return fmt.Errorf("service bind failed: %w", err)
	}
	return nil
}

func dial(cfg Config) (*ldap.Conn, error) {
	connectTimeoutMS := cfg.ConnectTimeoutMS
	if connectTimeoutMS == 0 {
		connectTimeoutMS = defaultConnectTimeoutMS
	}
	timeout := time.Duration(connectTimeoutMS) * time.Millisecond
	tlsCfg := &tls.Config{InsecureSkipVerify: cfg.TLSInsecureSkip} //nolint:gosec // controlled by config

	var conn *ldap.Conn
	var err error

	if cfg.StartTLS {
		conn, err = ldap.DialURL(cfg.URL, ldap.DialWithDialer(&net.Dialer{Timeout: timeout}))
		if err != nil {
			return nil, err
		}
		if err = conn.StartTLS(tlsCfg); err != nil {
			_ = conn.Close()
			return nil, err
		}
		return conn, nil
	}

	if strings.HasPrefix(cfg.URL, "ldaps://") {
		conn, err = ldap.DialURL(cfg.URL,
			ldap.DialWithDialer(&net.Dialer{Timeout: timeout}),
			ldap.DialWithTLSConfig(tlsCfg),
		)
	} else {
		conn, err = ldap.DialURL(cfg.URL, ldap.DialWithDialer(&net.Dialer{Timeout: timeout}))
	}
	return conn, err
}
