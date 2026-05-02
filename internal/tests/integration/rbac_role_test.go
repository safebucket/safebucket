//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/safebucket/safebucket/internal/models"
)

func TestRBAC_Roles(t *testing.T) {
	for _, scenario := range ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			app := BootScenario(t, scenario)

			guest := app.CreateUser(t, "guest@example.com")
			app.SetUserRole(t, guest.Email, models.RoleGuest)
			userA := app.CreateUser(t, "usera@example.com")
			userB := app.CreateUser(t, "userb@example.com")

			guestToken := app.LoginAs(t, guest.Email)
			userAToken := app.LoginAs(t, userA.Email)
			adminToken := app.LoginAdmin(t)

			tests := []struct {
				name   string
				actor  string
				token  string
				method string
				path   string
				body   any
				expect int
			}{

				{"guest POST buckets", "guest", guestToken, http.MethodPost, "/api/v1/buckets", models.BucketCreateUpdateBody{Name: "x"}, 403},
				{"guest GET buckets", "guest", guestToken, http.MethodGet, "/api/v1/buckets", nil, 200},
				{"guest POST users", "guest", guestToken, http.MethodPost, "/api/v1/users", models.UserCreateBody{FirstName: "A", LastName: "B", Email: "new1@example.com", Password: "password123"}, 403},
				{"guest GET users", "guest", guestToken, http.MethodGet, "/api/v1/users", nil, 403},
				{"guest DELETE userB", "guest", guestToken, http.MethodDelete, fmt.Sprintf("/api/v1/users/%s", userB.ID), nil, 403},
				{"guest GET admin stats", "guest", guestToken, http.MethodGet, "/api/v1/admin/stats", nil, 403},

				{"user POST buckets", "user", userAToken, http.MethodPost, "/api/v1/buckets", models.BucketCreateUpdateBody{Name: "x"}, 201},
				{"user GET users", "user", userAToken, http.MethodGet, "/api/v1/users", nil, 403},
				{"user POST users", "user", userAToken, http.MethodPost, "/api/v1/users", models.UserCreateBody{FirstName: "A", LastName: "B", Email: "new2@example.com", Password: "password123"}, 403},
				{"user GET admin stats", "user", userAToken, http.MethodGet, "/api/v1/admin/stats", nil, 403},
				{"user GET admin activity", "user", userAToken, http.MethodGet, "/api/v1/admin/activity", nil, 403},
				{"user GET admin buckets", "user", userAToken, http.MethodGet, "/api/v1/admin/buckets", nil, 403},
				{"user GET userA (self)", "user", userAToken, http.MethodGet, fmt.Sprintf("/api/v1/users/%s", userA.ID), nil, 200},
				{"user GET userB (other)", "user", userAToken, http.MethodGet, fmt.Sprintf("/api/v1/users/%s", userB.ID), nil, 403},
				{"user PATCH userA (self)", "user", userAToken, http.MethodPatch, fmt.Sprintf("/api/v1/users/%s", userA.ID), models.UserUpdateBody{FirstName: "x"}, 204},
				{"user PATCH userB (other)", "user", userAToken, http.MethodPatch, fmt.Sprintf("/api/v1/users/%s", userB.ID), models.UserUpdateBody{FirstName: "x"}, 403},
				{"user GET userA sessions (self)", "user", userAToken, http.MethodGet, fmt.Sprintf("/api/v1/users/%s/sessions", userA.ID), nil, 200},
				{"user DELETE userB session (other)", "user", userAToken, http.MethodDelete, fmt.Sprintf("/api/v1/users/%s/sessions/00000000-0000-0000-0000-000000000000", userB.ID), nil, 403},

				{"admin GET admin stats", "admin", adminToken, http.MethodGet, "/api/v1/admin/stats", nil, 200},
				{"admin POST users", "admin", adminToken, http.MethodPost, "/api/v1/users", models.UserCreateBody{FirstName: "A", LastName: "B", Email: "new3@example.com", Password: "password123"}, 201},
				{"admin GET userA", "admin", adminToken, http.MethodGet, fmt.Sprintf("/api/v1/users/%s", userA.ID), nil, 200},
				{"admin DELETE userB", "admin", adminToken, http.MethodDelete, fmt.Sprintf("/api/v1/users/%s", userB.ID), nil, 204},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					status := app.DoStatus(t, tt.method, tt.path, tt.token, tt.body)
					if status != tt.expect {
						t.Errorf("%s %s by %s: expected %d, got %d", tt.method, tt.path, tt.actor, tt.expect, status)
					}
				})
			}
		})
	}
}
