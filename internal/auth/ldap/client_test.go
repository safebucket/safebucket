package ldap

import (
	"errors"
	"testing"
	"time"

	ldapv3 "github.com/go-ldap/ldap/v3"
)

type fakeConn struct {
	binds         []bindCall
	bindErrors    map[string]error
	searchResult  *ldapv3.SearchResult
	searchErr     error
	lastSearchReq *ldapv3.SearchRequest
	timeout       time.Duration
	closed        bool
}

type bindCall struct {
	dn       string
	password string
}

func (f *fakeConn) Bind(dn, password string) error {
	f.binds = append(f.binds, bindCall{dn: dn, password: password})
	if err, ok := f.bindErrors[dn]; ok {
		return err
	}
	return nil
}

func (f *fakeConn) Search(req *ldapv3.SearchRequest) (*ldapv3.SearchResult, error) {
	f.lastSearchReq = req
	if f.searchErr != nil {
		return nil, f.searchErr
	}
	return f.searchResult, nil
}

func (f *fakeConn) SetTimeout(timeout time.Duration) {
	f.timeout = timeout
}

func (f *fakeConn) Close() error {
	f.closed = true
	return nil
}

func baseCfg() Config {
	return Config{
		URL:              "ldap://example.test:389",
		BindDN:           "cn=svc,dc=example,dc=test",
		BindPassword:     "svc-secret",
		SearchBase:       "ou=people,dc=example,dc=test",
		SearchFilter:     "(mail={username})",
		EmailAttribute:   "mail",
		DisplayAttribute: "displayName",
	}
}

func userEntry(dn, mail, display string) *ldapv3.Entry {
	return &ldapv3.Entry{
		DN: dn,
		Attributes: []*ldapv3.EntryAttribute{
			{Name: "mail", Values: []string{mail}},
			{Name: "displayName", Values: []string{display}},
		},
	}
}

func TestAuthenticateAndFetch_Success(t *testing.T) {
	conn := &fakeConn{
		searchResult: &ldapv3.SearchResult{
			Entries: []*ldapv3.Entry{
				userEntry("uid=jdoe,ou=people,dc=example,dc=test", "jdoe@example.test", "Jane Doe"),
			},
		},
	}
	dial := func(Config) (Conn, error) { return conn, nil }

	user, err := authenticateAndFetch(baseCfg(), "jdoe@example.test", "user-pw", dial)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Email != "jdoe@example.test" || user.DisplayName != "Jane Doe" {
		t.Fatalf("unexpected user: %+v", user)
	}
	if len(conn.binds) != 2 {
		t.Fatalf("expected 2 binds (service then user), got %d", len(conn.binds))
	}
	if conn.binds[0].dn != "cn=svc,dc=example,dc=test" || conn.binds[0].password != "svc-secret" {
		t.Errorf("first bind not service account: %+v", conn.binds[0])
	}
	if conn.binds[1].dn != "uid=jdoe,ou=people,dc=example,dc=test" ||
		conn.binds[1].password != "user-pw" {
		t.Errorf("second bind not user: %+v", conn.binds[1])
	}
	if !conn.closed {
		t.Error("connection was not closed")
	}
}

func TestAuthenticateAndFetch_EscapesFilter(t *testing.T) {
	conn := &fakeConn{
		searchResult: &ldapv3.SearchResult{
			Entries: []*ldapv3.Entry{userEntry("uid=x,dc=e,dc=t", "x@e.t", "X")},
		},
	}
	dial := func(Config) (Conn, error) { return conn, nil }

	_, err := authenticateAndFetch(baseCfg(), "a*b(c)\\d", "pw", dial)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	gotFilter := conn.lastSearchReq.Filter
	if gotFilter == "(mail=a*b(c)\\d)" {
		t.Fatalf("filter was not escaped: %q", gotFilter)
	}
}

func TestAuthenticateAndFetch_EmptyPassword(t *testing.T) {
	dial := func(Config) (Conn, error) {
		t.Fatal("dial should not be called for empty password")
		return nil, nil
	}
	_, err := authenticateAndFetch(baseCfg(), "jdoe@example.test", "", dial)
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("want ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthenticateAndFetch_UserNotFound(t *testing.T) {
	conn := &fakeConn{searchResult: &ldapv3.SearchResult{Entries: nil}}
	dial := func(Config) (Conn, error) { return conn, nil }

	_, err := authenticateAndFetch(baseCfg(), "ghost@example.test", "pw", dial)
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("want ErrUserNotFound, got %v", err)
	}
}

func TestAuthenticateAndFetch_AmbiguousMatch(t *testing.T) {
	conn := &fakeConn{
		searchResult: &ldapv3.SearchResult{
			Entries: []*ldapv3.Entry{
				userEntry("uid=a,dc=e,dc=t", "a@e.t", "A"),
				userEntry("uid=b,dc=e,dc=t", "b@e.t", "B"),
			},
		},
	}
	dial := func(Config) (Conn, error) { return conn, nil }

	_, err := authenticateAndFetch(baseCfg(), "x", "pw", dial)
	if err == nil {
		t.Fatal("expected ambiguous-match error")
	}
}

func TestAuthenticateAndFetch_ServiceBindFails(t *testing.T) {
	conn := &fakeConn{
		bindErrors: map[string]error{
			"cn=svc,dc=example,dc=test": errors.New("invalid creds"),
		},
	}
	dial := func(Config) (Conn, error) { return conn, nil }

	_, err := authenticateAndFetch(baseCfg(), "jdoe", "pw", dial)
	if !errors.Is(err, ErrServiceBindFailed) {
		t.Fatalf("want ErrServiceBindFailed, got %v", err)
	}
}

func TestAuthenticateAndFetch_InvalidUserCredentials(t *testing.T) {
	conn := &fakeConn{
		searchResult: &ldapv3.SearchResult{
			Entries: []*ldapv3.Entry{userEntry("uid=j,dc=e,dc=t", "j@e.t", "J")},
		},
		bindErrors: map[string]error{
			"uid=j,dc=e,dc=t": ldapv3.NewError(ldapv3.LDAPResultInvalidCredentials, errors.New("nope")),
		},
	}
	dial := func(Config) (Conn, error) { return conn, nil }

	_, err := authenticateAndFetch(baseCfg(), "j", "wrong-pw", dial)
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("want ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthenticateAndFetch_MissingEmailAttr(t *testing.T) {
	conn := &fakeConn{
		searchResult: &ldapv3.SearchResult{
			Entries: []*ldapv3.Entry{
				{
					DN: "uid=j,dc=e,dc=t",
					Attributes: []*ldapv3.EntryAttribute{
						{Name: "displayName", Values: []string{"J"}},
					},
				},
			},
		},
	}
	dial := func(Config) (Conn, error) { return conn, nil }

	_, err := authenticateAndFetch(baseCfg(), "j", "pw", dial)
	if !errors.Is(err, ErrMissingEmail) {
		t.Fatalf("want ErrMissingEmail, got %v", err)
	}
}

func TestAuthenticateAndFetch_DialFailure(t *testing.T) {
	dial := func(Config) (Conn, error) { return nil, errors.New("conn refused") }

	_, err := authenticateAndFetch(baseCfg(), "j", "pw", dial)
	if !errors.Is(err, ErrDirectoryUnavailable) {
		t.Fatalf("want ErrDirectoryUnavailable, got %v", err)
	}
}
