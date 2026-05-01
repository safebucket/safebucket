//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/safebucket/safebucket/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBucket_NonMemberCannotAccess(t *testing.T) {
	for _, scenario := range ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			app := BootScenario(t, scenario)

			adminToken := app.LoginAdmin(t)
			app.CreateUserAPI(t, adminToken, "owner@example.com")
			app.CreateUserAPI(t, adminToken, "outsider@example.com")

			bucket := app.CreateBucketAPI(t, adminToken, "private-bucket")
			app.SetBucketMembersAPI(t, adminToken, bucket.ID, []models.BucketMemberBody{
				{Email: "owner@example.com", Group: models.GroupOwner},
			})

			outsiderToken := app.LoginAs(t, "outsider@example.com")
			ownerToken := app.LoginAs(t, "owner@example.com")

			paths := []string{
				fmt.Sprintf("/api/v1/buckets/%s", bucket.ID),
				fmt.Sprintf("/api/v1/buckets/%s/activity", bucket.ID),
				fmt.Sprintf("/api/v1/buckets/%s/members", bucket.ID),
			}

			for _, path := range paths {
				t.Run(path, func(t *testing.T) {
					status := app.Do(t, http.MethodGet, path, outsiderToken, nil, nil)
					require.Equal(t, http.StatusForbidden, status)
				})
			}

			var got models.Bucket
			status := app.Do(t, http.MethodGet,
				fmt.Sprintf("/api/v1/buckets/%s", bucket.ID),
				ownerToken, nil, &got)
			require.Equal(t, http.StatusOK, status)
			assert.Equal(t, bucket.ID, got.ID)
		})
	}
}

func TestBucket_ViewerAndContributorCannotMutate(t *testing.T) {
	for _, scenario := range ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			app := BootScenario(t, scenario)

			adminToken := app.LoginAdmin(t)
			app.CreateUserAPI(t, adminToken, "owner@example.com")
			app.CreateUserAPI(t, adminToken, "viewer@example.com")
			app.CreateUserAPI(t, adminToken, "contributor@example.com")

			bucket := app.CreateBucketAPI(t, adminToken, "shared-bucket")
			app.SetBucketMembersAPI(t, adminToken, bucket.ID, []models.BucketMemberBody{
				{Email: "owner@example.com", Group: models.GroupOwner},
				{Email: "viewer@example.com", Group: models.GroupViewer},
				{Email: "contributor@example.com", Group: models.GroupContributor},
			})

			bucketPath := fmt.Sprintf("/api/v1/buckets/%s", bucket.ID)
			updateBody := models.BucketCreateUpdateBody{Name: "renamed"}

			cases := []struct {
				name  string
				email string
			}{
				{"viewer", "viewer@example.com"},
				{"contributor", "contributor@example.com"},
			}

			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					token := app.LoginAs(t, tc.email)

					patchStatus := app.Do(t, http.MethodPatch, bucketPath, token, updateBody, nil)
					require.Equal(t, http.StatusForbidden, patchStatus, "PATCH should be forbidden")

					deleteStatus := app.Do(t, http.MethodDelete, bucketPath, token, nil, nil)
					require.Equal(t, http.StatusForbidden, deleteStatus, "DELETE should be forbidden")
				})
			}

			ownerToken := app.LoginAs(t, "owner@example.com")
			ownerPatch := app.Do(t, http.MethodPatch, bucketPath, ownerToken, updateBody, nil)
			require.Equal(t, http.StatusNoContent, ownerPatch, "owner PATCH should succeed")
		})
	}
}

func TestBucket_GuestCannotCreate(t *testing.T) {
	for _, scenario := range ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			app := BootScenario(t, scenario)

			// POST /api/v1/users hardcodes RoleUser, so a Guest must be seeded directly.
			guest := CreateTestUser(t, app.DB, "guest@example.com", models.RoleGuest)
			token := app.LoginAs(t, guest.Email)

			status := app.Do(t, http.MethodPost, "/api/v1/buckets", token,
				models.BucketCreateUpdateBody{Name: "should-not-exist"}, nil)
			require.Equal(t, http.StatusForbidden, status)

			adminToken := app.LoginAdmin(t)
			user := app.CreateUserAPI(t, adminToken, "user@example.com")
			userToken := app.LoginAs(t, user.Email)
			var created models.Bucket
			userStatus := app.Do(t, http.MethodPost, "/api/v1/buckets", userToken,
				models.BucketCreateUpdateBody{Name: "ok"}, &created)
			require.Equal(t, http.StatusCreated, userStatus, "RoleUser POST should succeed")
		})
	}
}
