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

			user := CreateTestUser(t, app.DB, "invitee_simple@example.com", models.RoleUser)

			var loginResp models.AuthLoginResponse
			status := app.Do(t, http.MethodPost, "/api/v1/auth/login", "", models.AuthLoginBody{
				Email:    user.Email,
				Password: testPassword,
			}, &loginResp)

			require.Equal(t, http.StatusCreated, status)
			assert.NotEmpty(t, loginResp.AccessToken)
		})
	}
}
