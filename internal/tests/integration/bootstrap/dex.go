//go:build integration

package bootstrap

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"text/template"

	"github.com/google/uuid"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"golang.org/x/crypto/bcrypt"
)

const (
	defaultDexImage = "dexidp/dex:v2.45.1"
	dexClientID     = "safebucket-test"
	dexInternalPort = "5556/tcp"
	dexHealthPath   = "/dex/healthz"
	dexBcryptCost   = 10
)

type DexUser struct {
	Email    string
	Password string
	Username string
}

type DexInstance struct {
	Issuer       string
	ClientID     string
	ClientSecret string
}

func StartDex(t *testing.T, safebucketCallbackURL string, users []DexUser) DexInstance {
	t.Helper()
	require.NotEmpty(t, users, "dex needs at least one static user")

	hostPort := reserveLocalPort(t)
	issuer := fmt.Sprintf("http://127.0.0.1:%d/dex", hostPort)
	clientSecret := uuid.NewString()

	cfg := renderDexConfig(t, dexConfigInput{
		Issuer:       issuer,
		ClientID:     dexClientID,
		ClientSecret: clientSecret,
		RedirectURI:  safebucketCallbackURL,
		Users:        users,
	})

	cfgFile := filepath.Join(t.TempDir(), "dex.yaml")
	require.NoError(t, os.WriteFile(cfgFile, []byte(cfg), 0o600))

	image := os.Getenv("DEX_IMAGE")
	if image == "" {
		image = defaultDexImage
	}

	req := testcontainers.ContainerRequest{
		Image:        image,
		ExposedPorts: []string{dexInternalPort},
		Cmd:          []string{"dex", "serve", "/etc/dex/config.yaml"},
		Files: []testcontainers.ContainerFile{{
			HostFilePath:      cfgFile,
			ContainerFilePath: "/etc/dex/config.yaml",
			FileMode:          0o644,
		}},
		WaitingFor: wait.ForHTTP(dexHealthPath).
			WithPort(dexInternalPort).
			WithStatusCodeMatcher(func(s int) bool { return s == 200 }),
		HostConfigModifier: func(hc *container.HostConfig) {
			hc.PortBindings = network.PortMap{
				network.MustParsePort(dexInternalPort): []network.PortBinding{{
					HostIP:   netip.MustParseAddr("127.0.0.1"),
					HostPort: strconv.Itoa(hostPort),
				}},
			}
		},
	}

	ctx := context.Background()
	dexC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err, "start dex container")
	t.Cleanup(func() {
		if t.Failed() {
			if logs, lerr := dexC.Logs(ctx); lerr == nil {
				if data, rerr := io.ReadAll(logs); rerr == nil {
					t.Logf("dex container logs:\n%s", string(data))
				}
				_ = logs.Close()
			}
		}
		_ = testcontainers.TerminateContainer(dexC)
	})

	return DexInstance{
		Issuer:       issuer,
		ClientID:     dexClientID,
		ClientSecret: clientSecret,
	}
}

func reserveLocalPort(t *testing.T) int {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err, "reserve local port")
	port := lis.Addr().(*net.TCPAddr).Port
	require.NoError(t, lis.Close())
	return port
}

type dexConfigInput struct {
	Issuer       string
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Users        []DexUser
}

const dexConfigTemplate = `issuer: {{ .Issuer }}
storage:
  type: memory
web:
  http: 0.0.0.0:5556
oauth2:
  skipApprovalScreen: true
  passwordConnector: local
enablePasswordDB: true
staticClients:
  - id: {{ .ClientID }}
    name: Safebucket Test
    secret: {{ .ClientSecret }}
    redirectURIs:
      - {{ .RedirectURI }}
staticPasswords:
{{- range .Users }}
  - email: {{ .Email }}
    hash: {{ .Hash }}
    username: {{ .Username }}
    userID: {{ .UserID }}
{{- end }}
`

func renderDexConfig(t *testing.T, in dexConfigInput) string {
	t.Helper()

	type staticPassword struct{ Email, Hash, Username, UserID string }
	passwords := make([]staticPassword, 0, len(in.Users))
	for _, u := range in.Users {
		hash, err := bcrypt.GenerateFromPassword([]byte(u.Password), dexBcryptCost)
		require.NoError(t, err, "bcrypt user %s", u.Email)
		username := u.Username
		if username == "" {
			username = u.Email
		}
		passwords = append(passwords, staticPassword{u.Email, string(hash), username, uuid.NewString()})
	}

	var buf bytes.Buffer
	err := template.Must(template.New("dex").Parse(dexConfigTemplate)).Execute(&buf, struct {
		dexConfigInput
		Users []staticPassword
	}{in, passwords})
	require.NoError(t, err, "render dex config")
	return buf.String()
}
