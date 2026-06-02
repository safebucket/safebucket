//go:build integration

package auth_test

import (
	"errors"
	"html"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"testing"

	"github.com/safebucket/safebucket/internal/tests/integration/bootstrap"
	"github.com/stretchr/testify/require"
)

const (
	cookieAccessToken       = "safebucket_access_token"
	cookieRefreshToken      = "safebucket_refresh_token"
	oidcProviderKey         = "dex"
	oidcSafebucketBeginPath = "/api/v1/auth/providers/" + oidcProviderKey + "/begin"
)

func runOIDCAuthCodeFlow(
	t *testing.T, app *bootstrap.TestApp, email, password string,
) (*http.Response, *cookiejar.Jar) {
	t.Helper()

	jar, err := cookiejar.New(nil)
	require.NoError(t, err)

	webURL, err := url.Parse(app.Config.App.WebURL)
	require.NoError(t, err)

	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if req.URL.Host == webURL.Host {
				return http.ErrUseLastResponse
			}
			if len(via) > 15 {
				return errors.New("too many redirects in OIDC flow")
			}
			return nil
		},
	}

	resp, err := client.Get(app.URL(oidcSafebucketBeginPath))
	require.NoError(t, err, "GET /begin -> follow to Dex login form")
	require.Equal(t, http.StatusOK, resp.StatusCode, "expected Dex login form")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, resp.Body.Close())
	require.NoError(t, err)

	re := regexp.MustCompile(`<form[^>]*action="([^"]+)"`)
	m := re.FindSubmatch(body)
	require.Len(t, m, 2, "could not find Dex login form action in response body")

	u, _ := url.Parse(html.UnescapeString(string(m[1])))
	formURL := resp.Request.URL.ResolveReference(u)

	resp2, err := client.PostForm(formURL.String(), url.Values{
		"login":    {email},
		"password": {password},
	})
	require.NoError(t, err, "POST Dex login form")

	return resp2, jar
}

func cookieFromJar(jar *cookiejar.Jar, base *url.URL, name string) *http.Cookie {
	for _, c := range jar.Cookies(base) {
		if c.Name == name {
			return c
		}
	}
	return nil
}
