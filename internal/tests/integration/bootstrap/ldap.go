//go:build integration

package bootstrap

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/netip"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"text/template"
	"time"

	ldapclient "github.com/safebucket/safebucket/internal/auth/ldap"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	defaultLDAPImage = "osixia/openldap:2.6.10-alpha"
	ldapInternalPort = "3890/tcp"
	ldapBaseSuffix   = "dc=example,dc=org"
	ldapBindDN       = "cn=admin," + ldapBaseSuffix
	ldapBindPassword = "admin"
	ldapAdminHashed  = "{SSHA}VcB1k0xgV8sn8d7drBj8c2/Xwj+IE6WE"
	ldapBootstrap    = "/container/services/openldap-bootstrap/assets/ldif/data/custom/50-bootstrap.ldif"
)

type LDAPUser struct {
	Email     string
	Password  string
	UID       string
	FirstName string
	LastName  string
}

type LDAPInstance struct {
	URL          string
	BindDN       string
	BindPassword string
	BaseDN       string
}

func (i LDAPInstance) clientConfig() ldapclient.Config {
	return ldapclient.Config{
		URL:              i.URL,
		BindDN:           i.BindDN,
		BindPassword:     i.BindPassword,
		BaseDN:           i.BaseDN,
		UserFilter:       "(mail=%s)",
		AttributeMap:     ldapclient.AttributeMap{Email: "mail"},
		ConnectTimeoutMS: 5000,
	}
}

func StartLDAP(t *testing.T, users []LDAPUser) LDAPInstance {
	t.Helper()
	require.NotEmpty(t, users, "ldap needs at least one user")

	hostPort := reserveLocalPort(t)

	ldif := renderLDAPBootstrap(t, users)
	ldifFile := filepath.Join(t.TempDir(), "bootstrap.ldif")
	require.NoError(t, os.WriteFile(ldifFile, []byte(ldif), 0o600))

	image := os.Getenv("LDAP_IMAGE")
	if image == "" {
		image = defaultLDAPImage
	}

	req := testcontainers.ContainerRequest{
		Image:        image,
		ExposedPorts: []string{ldapInternalPort},
		Env: map[string]string{
			"OPENLDAP_BOOTSTRAP_ORGANIZATION":              "Safebucket Test",
			"OPENLDAP_BOOTSTRAP_SUFFIX":                    ldapBaseSuffix,
			"OPENLDAP_BOOTSTRAP_DATA_ROOT_DN":              ldapBindDN,
			"OPENLDAP_BOOTSTRAP_DATA_ROOT_PASSWORD_HASHED": ldapAdminHashed,
		},
		Files: []testcontainers.ContainerFile{{
			HostFilePath:      ldifFile,
			ContainerFilePath: ldapBootstrap,
			FileMode:          0o644,
		}},
		WaitingFor: wait.ForLog("slapd starting").WithStartupTimeout(90 * time.Second),
		HostConfigModifier: func(hc *container.HostConfig) {
			hc.PortBindings = network.PortMap{
				network.MustParsePort(ldapInternalPort): []network.PortBinding{{
					HostIP:   netip.MustParseAddr("127.0.0.1"),
					HostPort: strconv.Itoa(hostPort),
				}},
			}
		},
	}

	ctx := context.Background()
	ldapC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err, "start ldap container")
	t.Cleanup(func() {
		if t.Failed() {
			if logs, lerr := ldapC.Logs(ctx); lerr == nil {
				if data, rerr := io.ReadAll(logs); rerr == nil {
					t.Logf("ldap container logs:\n%s", string(data))
				}
				_ = logs.Close()
			}
		}
		_ = testcontainers.TerminateContainer(ldapC)
	})

	instance := LDAPInstance{
		URL:          fmt.Sprintf("ldap://127.0.0.1:%d", hostPort),
		BindDN:       ldapBindDN,
		BindPassword: ldapBindPassword,
		BaseDN:       "ou=users," + ldapBaseSuffix,
	}

	cfg := instance.clientConfig()
	require.Eventually(t, func() bool {
		return ldapclient.VerifyServiceBind(cfg) == nil
	}, 90*time.Second, time.Second, "ldap directory never became bindable")

	return instance
}

type ldapUserTemplate struct {
	UID       string
	FirstName string
	LastName  string
	Email     string
	Password  string
	UIDNumber int
}

const ldapBootstrapTemplate = `dn: ou=users,{{ .Suffix }}
objectClass: organizationalUnit
ou: users
{{ range .Users }}
dn: uid={{ .UID }},ou=users,{{ $.Suffix }}
objectClass: inetOrgPerson
objectClass: posixAccount
objectClass: shadowAccount
uid: {{ .UID }}
sn: {{ .LastName }}
givenName: {{ .FirstName }}
cn: {{ .FirstName }} {{ .LastName }}
displayName: {{ .FirstName }} {{ .LastName }}
mail: {{ .Email }}
uidNumber: {{ .UIDNumber }}
gidNumber: 1000
homeDirectory: /home/{{ .UID }}
loginShell: /bin/bash
userPassword: {{ .Password }}
{{ end }}`

func renderLDAPBootstrap(t *testing.T, users []LDAPUser) string {
	t.Helper()

	rendered := make([]ldapUserTemplate, 0, len(users))
	for i, u := range users {
		uid := u.UID
		if uid == "" {
			uid = fmt.Sprintf("user%d", i)
		}
		first := u.FirstName
		if first == "" {
			first = uid
		}
		last := u.LastName
		if last == "" {
			last = "User"
		}
		rendered = append(rendered, ldapUserTemplate{
			UID:       uid,
			FirstName: first,
			LastName:  last,
			Email:     u.Email,
			Password:  u.Password,
			UIDNumber: 1000 + i,
		})
	}

	var buf bytes.Buffer
	err := template.Must(template.New("ldif").Parse(ldapBootstrapTemplate)).Execute(&buf, struct {
		Suffix string
		Users  []ldapUserTemplate
	}{ldapBaseSuffix, rendered})
	require.NoError(t, err, "render ldap bootstrap ldif")
	return buf.String()
}
