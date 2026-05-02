//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/stretchr/testify/require"
)

func TestRBAC_Group(t *testing.T) {
	for _, scenario := range ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			cfg := LoadScenario(t, scenario)
			cfg = WithLocalSharing(cfg, true)
			app := BootTestApp(t, cfg)

			ownerA := app.CreateUser(t, "ownera@example.com")
			viewerA := app.CreateUser(t, "viewera@example.com")
			contribA := app.CreateUser(t, "contriba@example.com")
			ownerB := app.CreateUser(t, "ownerb@example.com")
			nonMember := app.CreateUser(t, "nonmember@example.com")

			ownerAToken := app.LoginAs(t, ownerA.Email)
			ownerBToken := app.LoginAs(t, ownerB.Email)
			viewerAToken := app.LoginAs(t, viewerA.Email)
			contribAToken := app.LoginAs(t, contribA.Email)
			nonMemberToken := app.LoginAs(t, nonMember.Email)
			adminToken := app.LoginAdmin(t)

			bucketA := app.CreateBucket(t, ownerAToken, "bucketA")
			app.AddMembers(t, ownerAToken, bucketA.ID.String(), []models.BucketMemberBody{
				{Email: viewerA.Email, Group: models.GroupViewer},
				{Email: contribA.Email, Group: models.GroupContributor},
				{Email: app.Config.App.AdminEmail, Group: models.GroupViewer},
			})

			bucketB := app.CreateBucket(t, ownerBToken, "bucketB")

			actors := []struct {
				name          string
				token         string
				isViewerPlus  bool
				isContribPlus bool
				isOwnerPlus   bool
			}{
				{"viewer", viewerAToken, true, false, false},
				{"contrib", contribAToken, true, true, false},
				{"owner", ownerAToken, true, true, true},
				{"admin", adminToken, true, true, true},
				{"nonmember", nonMemberToken, false, false, false},
			}

			createFile := func() string {
				return app.UploadTestFile(t, ownerAToken, bucketA.ID.String(), fmt.Sprintf("file_%s.txt", uuid.New()))
			}
			createTrashedFile := func() string {
				fileID := app.UploadTestFile(t, ownerAToken, bucketA.ID.String(), fmt.Sprintf("file_%s.txt", uuid.New()))
				require.Equal(t, http.StatusNoContent, app.DoStatus(t, http.MethodPatch,
					fmt.Sprintf("/api/v1/buckets/%s/files/%s", bucketA.ID, fileID), ownerAToken,
					models.FilePatchBody{Status: string(models.FileStatusDeleted)}))
				return fileID
			}
			createFolder := func() string {
				var folder models.Folder
				require.Equal(t, http.StatusCreated, app.Do(t, http.MethodPost,
					fmt.Sprintf("/api/v1/buckets/%s/folders", bucketA.ID), ownerAToken,
					models.FolderCreateBody{Name: fmt.Sprintf("folder_%s", uuid.New())}, &folder))
				return folder.ID.String()
			}
			createTrashedFolder := func() string {
				folderID := createFolder()
				require.Equal(t, http.StatusNoContent, app.DoStatus(t, http.MethodPatch,
					fmt.Sprintf("/api/v1/buckets/%s/folders/%s", bucketA.ID, folderID), ownerAToken,
					models.FolderPatchBody{Status: models.FolderStatusDeleted}))
				return folderID
			}
			createShare := func() string {
				var share models.Share
				require.Equal(t, http.StatusCreated, app.Do(t, http.MethodPost,
					fmt.Sprintf("/api/v1/buckets/%s/shares", bucketA.ID), ownerAToken,
					models.ShareCreateBody{Name: fmt.Sprintf("share_%s", uuid.New()), Type: models.ShareTypeBucket}, &share))
				return share.ID.String()
			}
			createBucket := func() string {
				b := app.CreateBucket(t, ownerAToken, fmt.Sprintf("temp_bucket_%s", uuid.New()))
				app.AddMembers(t, ownerAToken, b.ID.String(), []models.BucketMemberBody{
					{Email: viewerA.Email, Group: models.GroupViewer},
					{Email: contribA.Email, Group: models.GroupContributor},
				})
				return b.ID.String()
			}

			trueVal := true

			routes := []struct {
				name        string
				method      string
				pathFunc    func() string
				bodyFunc    func() any
				level       string
				successCode int
			}{
				{"GET bucket", http.MethodGet, func() string { return fmt.Sprintf("/api/v1/buckets/%s", bucketA.ID) }, func() any { return nil }, "viewer", 200},
				{"GET activity", http.MethodGet, func() string { return fmt.Sprintf("/api/v1/buckets/%s/activity", bucketA.ID) }, func() any { return nil }, "viewer", 200},
				{"GET members", http.MethodGet, func() string { return fmt.Sprintf("/api/v1/buckets/%s/members", bucketA.ID) }, func() any { return nil }, "viewer", 200},
				{"GET file", http.MethodGet, func() string { return fmt.Sprintf("/api/v1/buckets/%s/files/%s/download", bucketA.ID, createFile()) }, func() any { return nil }, "viewer", 200},
				{"PATCH notifications", http.MethodPatch, func() string { return fmt.Sprintf("/api/v1/buckets/%s/members/notifications", bucketA.ID) }, func() any {
					return models.MembershipNotificationBody{UploadNotifications: &trueVal, DownloadNotifications: &trueVal}
				}, "viewer", 204},

				{"POST file", http.MethodPost, func() string { return fmt.Sprintf("/api/v1/buckets/%s/files", bucketA.ID) }, func() any {
					return models.FileTransferBody{Name: fmt.Sprintf("new_%s.txt", uuid.New()), Size: 1}
				}, "contrib", 201},
				{"PATCH file", http.MethodPatch, func() string { return fmt.Sprintf("/api/v1/buckets/%s/files/%s", bucketA.ID, createFile()) }, func() any {
					return models.FilePatchBody{Status: string(models.FileStatusDeleted)}
				}, "contrib", 204},
				{"DELETE file", http.MethodDelete, func() string { return fmt.Sprintf("/api/v1/buckets/%s/files/%s", bucketA.ID, createTrashedFile()) }, func() any { return nil }, "contrib", 204},
				{"POST folder", http.MethodPost, func() string { return fmt.Sprintf("/api/v1/buckets/%s/folders", bucketA.ID) }, func() any {
					return models.FolderCreateBody{Name: fmt.Sprintf("newf_%s", uuid.New())}
				}, "contrib", 201},
				{"PUT folder", http.MethodPut, func() string { return fmt.Sprintf("/api/v1/buckets/%s/folders/%s", bucketA.ID, createFolder()) }, func() any {
					return models.FolderUpdateBody{Name: fmt.Sprintf("newf2_%s", uuid.New())}
				}, "contrib", 204},
				{"PATCH folder", http.MethodPatch, func() string { return fmt.Sprintf("/api/v1/buckets/%s/folders/%s", bucketA.ID, createFolder()) }, func() any {
					return models.FolderPatchBody{Status: models.FolderStatusDeleted}
				}, "contrib", 204},
				{"DELETE folder", http.MethodDelete, func() string { return fmt.Sprintf("/api/v1/buckets/%s/folders/%s", bucketA.ID, createTrashedFolder()) }, func() any { return nil }, "contrib", 204},

				{"PATCH bucket", http.MethodPatch, func() string { return fmt.Sprintf("/api/v1/buckets/%s", createBucket()) }, func() any {
					return models.BucketCreateUpdateBody{Name: "newname"}
				}, "owner", 204},
				{"DELETE bucket", http.MethodDelete, func() string { return fmt.Sprintf("/api/v1/buckets/%s", createBucket()) }, func() any { return nil }, "owner", 204},
				{"PUT members", http.MethodPut, func() string { return fmt.Sprintf("/api/v1/buckets/%s/members", createBucket()) }, func() any {
					return models.UpdateMembersBody{Members: []models.BucketMemberBody{{Email: ownerA.Email, Group: models.GroupOwner}}}
				}, "owner", 204},
				{"GET shares", http.MethodGet, func() string { return fmt.Sprintf("/api/v1/buckets/%s/shares", bucketA.ID) }, func() any { return nil }, "owner", 200},
				{"POST shares", http.MethodPost, func() string { return fmt.Sprintf("/api/v1/buckets/%s/shares", bucketA.ID) }, func() any {
					return models.ShareCreateBody{Name: fmt.Sprintf("s_%s", uuid.New()), Type: models.ShareTypeBucket}
				}, "owner", 201},
				{"DELETE shares", http.MethodDelete, func() string { return fmt.Sprintf("/api/v1/buckets/%s/shares/%s", bucketA.ID, createShare()) }, func() any { return nil }, "owner", 204},
			}

			for _, route := range routes {
				for _, actor := range actors {
					t.Run(fmt.Sprintf("%s by %s", route.name, actor.name), func(t *testing.T) {
						path := route.pathFunc()

						allowed := false
						switch route.level {
						case "viewer":
							allowed = actor.isViewerPlus
						case "contrib":
							allowed = actor.isContribPlus
						case "owner":
							allowed = actor.isOwnerPlus
						}

						status := app.DoStatus(t, route.method, path, actor.token, route.bodyFunc())

						if allowed {
							require.Equal(t, route.successCode, status, "%s by %s should succeed", route.name, actor.name)
						} else {
							require.Equal(t, 403, status, "%s by %s should be forbidden", route.name, actor.name)
						}
					})
				}
			}

			t.Run("admin platform bypass", func(t *testing.T) {
				status := app.DoStatus(t, http.MethodGet, fmt.Sprintf("/api/v1/buckets/%s", bucketB.ID), adminToken, nil)
				require.Equal(t, 200, status)
			})

			t.Run("cross-bucket isolation", func(t *testing.T) {
				status := app.DoStatus(t, http.MethodGet, fmt.Sprintf("/api/v1/buckets/%s", bucketB.ID), contribAToken, nil)
				require.Equal(t, 403, status)
			})

			t.Run("soft-deleted bucket", func(t *testing.T) {
				b := app.CreateBucket(t, ownerAToken, "del_bucket")
				require.Equal(t, 204, app.DoStatus(t, http.MethodDelete, fmt.Sprintf("/api/v1/buckets/%s", b.ID), ownerAToken, nil))

				status := app.DoStatus(t, http.MethodGet, fmt.Sprintf("/api/v1/buckets/%s", b.ID), ownerAToken, nil)
				require.Equal(t, 404, status)
			})

			t.Run("unknown bucket uuid", func(t *testing.T) {
				status := app.DoStatus(t, http.MethodGet, "/api/v1/buckets/00000000-0000-0000-0000-000000000000", ownerAToken, nil)
				require.Equal(t, 403, status)
			})
		})
	}
}
