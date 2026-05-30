//go:build integration

package user_test

import (
	"net/http"
	"testing"

	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/tests/integration/bootstrap"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUser_CreateAndLogin(t *testing.T) {
	for _, scenario := range bootstrap.ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			app := bootstrap.BootScenario(t, scenario)

			user := app.CreateUser(t, "invitee_simple@example.com")

			status, token := app.DoGetAuthCookie(t, http.MethodPost, "/api/v1/auth/login", "",
				models.AuthLoginBody{Email: user.Email, Password: bootstrap.TestPassword})

			require.Equal(t, http.StatusOK, status)
			assert.NotEmpty(t, token)
		})
	}
}
