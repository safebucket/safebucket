//go:build integration

package integration

import (
	"net/http"
	"testing"

	"github.com/safebucket/safebucket/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUser_CreateAndLogin(t *testing.T) {
	for _, scenario := range ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			app := BootScenario(t, scenario)

			user := app.CreateUser(t, "invitee_simple@example.com")

			status, token := app.doGetAuthCookie(t, http.MethodPost, "/api/v1/auth/login", "",
				models.AuthLoginBody{Email: user.Email, Password: testPassword})

			require.Equal(t, http.StatusOK, status)
			assert.NotEmpty(t, token)
		})
	}
}
