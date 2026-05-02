//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/safebucket/safebucket/internal/models"
	"github.com/stretchr/testify/require"
)

func TestBucketMember_Invariants(t *testing.T) {
	for _, scenario := range ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			t.Run("sharing.allowed=false", func(t *testing.T) {
				cfg := LoadScenario(t, scenario)
				app := BootTestApp(t, cfg)
				ownerA := app.CreateUser(t, "ownera@example.com")
				userB := app.CreateUser(t, "userb@example.com")
				ownerAToken := app.LoginAs(t, ownerA.Email)
				bucketA := app.CreateBucket(t, ownerAToken, "bucketA")

				body := models.UpdateMembersBody{
					Members: []models.BucketMemberBody{
						{Email: userB.Email, Group: models.GroupViewer},
					},
				}
				status := app.DoStatus(t, http.MethodPut, fmt.Sprintf("/api/v1/buckets/%s/members", bucketA.ID), ownerAToken, body)
				require.Equal(t, 403, status)

				members := app.GetMembers(t, ownerAToken, bucketA.ID.String())
				for _, m := range members {
					require.NotEqual(t, userB.Email, m.Email, "userB should not be a member after rejected PUT")
				}
			})

			t.Run("sharing.allowed=true, no domains", func(t *testing.T) {
				cfg := LoadScenario(t, scenario)
				cfg = WithLocalSharing(cfg, true)
				app := BootTestApp(t, cfg)
				ownerA := app.CreateUser(t, "ownera@example.com")
				userB := app.CreateUser(t, "userb@example.com")
				ownerAToken := app.LoginAs(t, ownerA.Email)
				bucketA := app.CreateBucket(t, ownerAToken, "bucketA")

				body := models.UpdateMembersBody{
					Members: []models.BucketMemberBody{
						{Email: userB.Email, Group: models.GroupViewer},
					},
				}
				status := app.DoStatus(t, http.MethodPut, fmt.Sprintf("/api/v1/buckets/%s/members", bucketA.ID), ownerAToken, body)
				require.Equal(t, 204, status)

				members := app.GetMembers(t, ownerAToken, bucketA.ID.String())
				found := false
				for _, m := range members {
					if m.Email == userB.Email {
						found = true
						break
					}
				}
				require.True(t, found, "userB should be a member after successful PUT")

				app.Eventually(t, func() bool {
					nots := app.ReadNotifications(t)
					for _, n := range nots {
						if n.To == userB.Email && n.TemplateName == "bucket_shared_with" {
							return true
						}
					}
					return false
				}, "should emit BucketSharedWith notification")
			})

			t.Run("sharing.allowed=true, domains restricted", func(t *testing.T) {
				cfg := LoadScenario(t, scenario)
				cfg = WithLocalSharing(cfg, true, "allowed.example")
				app := BootTestApp(t, cfg)
				ownerA := app.CreateUser(t, "ownera@example.com")
				foo := app.CreateUser(t, "foo@allowed.example")
				bar := app.CreateUser(t, "bar@other.example")
				ownerAToken := app.LoginAs(t, ownerA.Email)
				bucketA := app.CreateBucket(t, ownerAToken, "bucketA")

				body := models.UpdateMembersBody{
					Members: []models.BucketMemberBody{
						{Email: foo.Email, Group: models.GroupViewer},
						{Email: bar.Email, Group: models.GroupViewer},
					},
				}
				status := app.DoStatus(t, http.MethodPut, fmt.Sprintf("/api/v1/buckets/%s/members", bucketA.ID), ownerAToken, body)
				require.Equal(t, 204, status)

				members := app.GetMembers(t, ownerAToken, bucketA.ID.String())
				var fooFound, barFound bool
				for _, m := range members {
					switch m.Email {
					case foo.Email:
						fooFound = true
					case bar.Email:
						barFound = true
					}
				}
				require.True(t, fooFound, "foo (allowed domain) should be a member")
				require.False(t, barFound, "bar (restricted domain) should not be a member")
			})

			t.Run("Unknown email + sharing.allowed=true", func(t *testing.T) {
				cfg := LoadScenario(t, scenario)
				cfg = WithLocalSharing(cfg, true)
				app := BootTestApp(t, cfg)
				ownerA := app.CreateUser(t, "ownera@example.com")
				ownerAToken := app.LoginAs(t, ownerA.Email)
				bucketA := app.CreateBucket(t, ownerAToken, "bucketA")

				body := models.UpdateMembersBody{
					Members: []models.BucketMemberBody{
						{Email: "nobody@example.com", Group: models.GroupViewer},
					},
				}
				status := app.DoStatus(t, http.MethodPut, fmt.Sprintf("/api/v1/buckets/%s/members", bucketA.ID), ownerAToken, body)
				require.Equal(t, 204, status)

				app.Eventually(t, func() bool {
					nots := app.ReadNotifications(t)
					for _, n := range nots {
						if n.To == "nobody@example.com" && n.TemplateName == "user_invitation" {
							return true
						}
					}
					return false
				}, "should emit UserInvitation notification")
			})

			t.Run("Owner preservation when self is omitted from body", func(t *testing.T) {
				cfg := LoadScenario(t, scenario)
				cfg = WithLocalSharing(cfg, true)
				app := BootTestApp(t, cfg)
				ownerA := app.CreateUser(t, "ownera@example.com")
				userB := app.CreateUser(t, "userb@example.com")
				ownerAToken := app.LoginAs(t, ownerA.Email)
				bucketA := app.CreateBucket(t, ownerAToken, "bucketA")

				body := models.UpdateMembersBody{
					Members: []models.BucketMemberBody{
						{Email: userB.Email, Group: models.GroupOwner},
					},
				}
				status := app.DoStatus(t, http.MethodPut, fmt.Sprintf("/api/v1/buckets/%s/members", bucketA.ID), ownerAToken, body)
				require.Equal(t, 204, status)

				members := app.GetMembers(t, ownerAToken, bucketA.ID.String())
				for _, m := range members {
					switch m.Email {
					case ownerA.Email:
						require.Equal(t, models.GroupOwner, m.Group, "ownerA should remain owner")
					case userB.Email:
						require.Equal(t, models.GroupOwner, m.Group, "userB should be owner")
					}
				}
			})

			t.Run("self-demotion silently ignored when other members exist", func(t *testing.T) {
				cfg := LoadScenario(t, scenario)
				cfg = WithLocalSharing(cfg, true)
				app := BootTestApp(t, cfg)
				ownerA := app.CreateUser(t, "ownera@example.com")
				ownerB := app.CreateUser(t, "ownerb@example.com")
				ownerAToken := app.LoginAs(t, ownerA.Email)
				bucketA := app.CreateBucket(t, ownerAToken, "bucketA")
				app.AddMembers(t, ownerAToken, bucketA.ID.String(), []models.BucketMemberBody{
					{Email: ownerB.Email, Group: models.GroupOwner},
				})

				body := models.UpdateMembersBody{
					Members: []models.BucketMemberBody{
						{Email: ownerA.Email, Group: models.GroupViewer},
						{Email: ownerB.Email, Group: models.GroupOwner},
					},
				}
				status := app.DoStatus(t, http.MethodPut, fmt.Sprintf("/api/v1/buckets/%s/members", bucketA.ID), ownerAToken, body)
				require.Equal(t, 204, status)

				members := app.GetMembers(t, ownerAToken, bucketA.ID.String())
				for _, m := range members {
					switch m.Email {
					case ownerA.Email:
						require.Equal(t, models.GroupOwner, m.Group, "A should remain owner because they are filtered out of the update request")
					case ownerB.Email:
						require.Equal(t, models.GroupOwner, m.Group)
					}
				}
			})
		})
	}
}
