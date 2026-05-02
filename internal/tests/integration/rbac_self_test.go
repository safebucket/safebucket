//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/safebucket/safebucket/internal/models"
	"github.com/stretchr/testify/require"
)

func TestRBAC_SelfOrAdmin(t *testing.T) {
	for _, scenario := range ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			app := BootScenario(t, scenario)

			userA := app.CreateUser(t, "usera@example.com")
			userB := app.CreateUser(t, "userb@example.com")

			userAToken := app.LoginAs(t, userA.Email)
			userBToken := app.LoginAs(t, userB.Email)
			adminToken := app.LoginAdmin(t)

			t.Run("User reads/patches own profile", func(t *testing.T) {
				status := app.DoStatus(t, http.MethodGet, fmt.Sprintf("/api/v1/users/%s", userA.ID), userAToken, nil)
				require.Equal(t, 200, status)

				status = app.DoStatus(t, http.MethodPatch, fmt.Sprintf("/api/v1/users/%s", userA.ID), userAToken, models.UserUpdateBody{FirstName: "Updated"})
				require.Equal(t, 204, status)
			})

			t.Run("User reads another user's profile/sessions", func(t *testing.T) {
				status := app.DoStatus(t, http.MethodGet, fmt.Sprintf("/api/v1/users/%s", userB.ID), userAToken, nil)
				require.Equal(t, 403, status)

				status = app.DoStatus(t, http.MethodGet, fmt.Sprintf("/api/v1/users/%s/sessions", userB.ID), userAToken, nil)
				require.Equal(t, 403, status)
			})

			t.Run("Admin reads any user's profile/sessions", func(t *testing.T) {
				status := app.DoStatus(t, http.MethodGet, fmt.Sprintf("/api/v1/users/%s", userA.ID), adminToken, nil)
				require.Equal(t, 200, status)

				status = app.DoStatus(t, http.MethodGet, fmt.Sprintf("/api/v1/users/%s/sessions", userA.ID), adminToken, nil)
				require.Equal(t, 200, status)
			})

			t.Run("User deletes own session vs other user session", func(t *testing.T) {
				var sessionsResp struct {
					Sessions []struct {
						ID string `json:"id"`
					} `json:"sessions"`
				}
				status := app.Do(t, http.MethodGet, fmt.Sprintf("/api/v1/users/%s/sessions", userA.ID), userAToken, nil, &sessionsResp)
				require.Equal(t, 200, status)
				require.NotEmpty(t, sessionsResp.Sessions)
				sessionA := sessionsResp.Sessions[0]

				status = app.Do(t, http.MethodGet, fmt.Sprintf("/api/v1/users/%s/sessions", userB.ID), userBToken, nil, &sessionsResp)
				require.Equal(t, 200, status)
				require.NotEmpty(t, sessionsResp.Sessions)
				sessionB := sessionsResp.Sessions[0]

				status = app.DoStatus(t, http.MethodDelete, fmt.Sprintf("/api/v1/users/%s/sessions/%s", userB.ID, sessionB.ID), userAToken, nil)
				require.Equal(t, 403, status)

				status = app.DoStatus(t, http.MethodDelete, fmt.Sprintf("/api/v1/users/%s/sessions/%s", userA.ID, sessionA.ID), userAToken, nil)
				require.Equal(t, 204, status)
			})
		})
	}
}
