//go:build integration

package invite_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/safebucket/safebucket/internal/configuration"
	apierrors "github.com/safebucket/safebucket/internal/errors"
	"github.com/safebucket/safebucket/internal/events"
	"github.com/safebucket/safebucket/internal/models"
	"github.com/safebucket/safebucket/internal/tests/integration/bootstrap"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const invitedPassword = "invited-correct-horse-staple"

type inviteFixture struct {
	invite models.Invite
	email  string
	code   string
}

func TestInviteFlow(t *testing.T) {
	for _, scenario := range bootstrap.ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			cfg := bootstrap.WithLocalSharing(bootstrap.LoadScenario(t, scenario), true)
			app := bootstrap.BootTestApp(t, cfg)
			owner := app.CreateUser(t, "invite-owner@example.com")
			ownerToken := app.LoginAs(t, owner.Email)
			bucket := app.CreateBucket(t, ownerToken, "invite-bucket")

			t.Run("unknown user accepts invite and gains bucket access", func(t *testing.T) {
				fixture := createInviteChallenge(t, app, ownerToken, bucket, uniqueEmail("accept"))
				validatePath := fmt.Sprintf("/api/v1/invites/%s/challenges/%s/validate",
					fixture.invite.ID, challengeID(t, app, fixture.invite.ID))

				status, accessToken := app.DoGetAuthCookie(t, http.MethodPost, validatePath, "",
					models.InviteChallengeValidateBody{Code: fixture.code, NewPassword: invitedPassword})
				require.Equal(t, http.StatusNoContent, status)
				require.NotEmpty(t, accessToken)

				var user models.User
				require.NoError(t, app.DB().Where("email = ?", fixture.email).First(&user).Error)
				assert.Equal(t, models.RoleGuest, user.Role)
				assert.Equal(t, models.LocalProviderType, user.ProviderType)

				var membership models.Membership
				require.NoError(t, app.DB().Where("user_id = ? AND bucket_id = ?", user.ID, bucket.ID).
					First(&membership).Error)
				assert.Equal(t, models.GroupViewer, membership.Group)

				var inviteCount int64
				require.NoError(t, app.DB().Model(&models.Invite{}).
					Where("id = ?", fixture.invite.ID).Count(&inviteCount).Error)
				assert.Zero(t, inviteCount)

				assert.Equal(t, http.StatusOK, app.DoStatus(t, http.MethodGet,
					fmt.Sprintf("/api/v1/buckets/%s", bucket.ID), accessToken, nil))

				loginStatus, loginToken := app.DoGetAuthCookie(t, http.MethodPost, "/api/v1/auth/login", "",
					models.AuthLoginBody{Email: fixture.email, Password: invitedPassword})
				assert.Equal(t, http.StatusOK, loginStatus)
				assert.NotEmpty(t, loginToken)

				welcome := waitForNotification[events.UserWelcomePayload](t, app, fixture.email, "user_welcome")
				assert.Equal(t, fixture.email, welcome.Email)

				status, codes := app.DoExpectError(t, http.MethodPost, validatePath, "",
					models.InviteChallengeValidateBody{Code: fixture.code, NewPassword: invitedPassword})
				assert.Equal(t, http.StatusNotFound, status)
				assert.Contains(t, codes, apierrors.CodeChallengeNotFound)
			})

			t.Run("wrong code decrements attempts and the correct code still succeeds", func(t *testing.T) {
				fixture := createInviteChallenge(t, app, ownerToken, bucket, uniqueEmail("wrong-code"))
				challenge := getChallenge(t, app, fixture.invite.ID)
				validatePath := fmt.Sprintf("/api/v1/invites/%s/challenges/%s/validate",
					fixture.invite.ID, challenge.ID)

				status, codes := app.DoExpectError(t, http.MethodPost, validatePath, "",
					models.InviteChallengeValidateBody{Code: wrongCode(fixture.code), NewPassword: invitedPassword})
				assert.Equal(t, http.StatusUnauthorized, status)
				assert.Contains(t, codes, apierrors.CodeWrongCode)

				var updated models.Challenge
				require.NoError(t, app.DB().First(&updated, "id = ?", challenge.ID).Error)
				assert.Equal(t, configuration.SecurityChallengeMaxFailedAttempts-1, updated.AttemptsLeft)

				status, accessToken := app.DoGetAuthCookie(t, http.MethodPost, validatePath, "",
					models.InviteChallengeValidateBody{Code: fixture.code, NewPassword: invitedPassword})
				assert.Equal(t, http.StatusNoContent, status)
				assert.NotEmpty(t, accessToken)
			})

			t.Run("expired challenge cannot create a user", func(t *testing.T) {
				fixture := createInviteChallenge(t, app, ownerToken, bucket, uniqueEmail("expired"))
				challenge := getChallenge(t, app, fixture.invite.ID)
				require.NoError(t, app.DB().Model(&models.Challenge{}).Where("id = ?", challenge.ID).
					Update("expires_at", time.Now().UTC().Add(-24*time.Hour)).Error)

				status, codes := app.DoExpectError(t, http.MethodPost,
					fmt.Sprintf("/api/v1/invites/%s/challenges/%s/validate", fixture.invite.ID, challenge.ID), "",
					models.InviteChallengeValidateBody{Code: fixture.code, NewPassword: invitedPassword})
				assert.Equal(t, http.StatusGone, status)
				assert.Contains(t, codes, apierrors.CodeChallengeExpired)
				assertUserDoesNotExist(t, app, fixture.email)

				var challengeCount int64
				require.NoError(t, app.DB().Model(&models.Challenge{}).
					Where("id = ?", challenge.ID).Count(&challengeCount).Error)
				assert.Zero(t, challengeCount)
			})

			t.Run("failed attempts lock the challenge", func(t *testing.T) {
				fixture := createInviteChallenge(t, app, ownerToken, bucket, uniqueEmail("locked"))
				challenge := getChallenge(t, app, fixture.invite.ID)
				validatePath := fmt.Sprintf("/api/v1/invites/%s/challenges/%s/validate",
					fixture.invite.ID, challenge.ID)

				for attempt := 1; attempt <= configuration.SecurityChallengeMaxFailedAttempts; attempt++ {
					status, codes := app.DoExpectError(t, http.MethodPost, validatePath, "",
						models.InviteChallengeValidateBody{
							Code: wrongCode(fixture.code), NewPassword: invitedPassword,
						})
					if attempt < configuration.SecurityChallengeMaxFailedAttempts {
						assert.Equal(t, http.StatusUnauthorized, status)
						assert.Contains(t, codes, apierrors.CodeWrongCode)
					} else {
						assert.Equal(t, http.StatusForbidden, status)
						assert.Contains(t, codes, apierrors.CodeChallengeLocked)
					}
				}

				status, codes := app.DoExpectError(t, http.MethodPost, validatePath, "",
					models.InviteChallengeValidateBody{Code: fixture.code, NewPassword: invitedPassword})
				assert.Equal(t, http.StatusNotFound, status)
				assert.Contains(t, codes, apierrors.CodeChallengeNotFound)
				assertUserDoesNotExist(t, app, fixture.email)
			})
		})
	}
}

func createInviteChallenge(
	t *testing.T,
	app *bootstrap.TestApp,
	ownerToken string,
	bucket models.Bucket,
	email string,
) inviteFixture {
	t.Helper()

	app.AddMembers(t, ownerToken, bucket.ID.String(), []models.BucketMemberBody{
		{Email: email, Group: models.GroupViewer},
	})

	var invite models.Invite
	require.NoError(t, app.DB().Where("email = ? AND bucket_id = ?", email, bucket.ID).First(&invite).Error)

	invitation := waitForNotification[events.UserInvitationPayload](t, app, email, "user_invitation")
	assert.True(t, strings.HasSuffix(invitation.InviteURL, "/invites/"+invite.ID.String()))
	assert.Equal(t, bucket.Name, invitation.BucketName)
	assert.Equal(t, models.GroupViewer, invitation.Group)

	status := app.DoStatus(t, http.MethodPost,
		fmt.Sprintf("/api/v1/invites/%s/challenges", invite.ID), "",
		models.InviteChallengeCreateBody{Email: email})
	require.Equal(t, http.StatusCreated, status)

	challenge := getChallenge(t, app, invite.ID)
	notification := waitForNotification[events.ChallengeUserInvitePayload](t, app, email, "user_invited")
	assert.True(t, strings.HasSuffix(notification.ChallengeURL,
		fmt.Sprintf("/invites/%s/challenges/%s", invite.ID, challenge.ID)))
	require.NotEmpty(t, notification.Secret)

	return inviteFixture{invite: invite, email: email, code: notification.Secret}
}

func waitForNotification[T any](
	t *testing.T,
	app *bootstrap.TestApp,
	email, template string,
) T {
	t.Helper()

	var payload T
	app.Eventually(t, func() bool {
		for _, notification := range app.ReadNotifications(t) {
			if notification.To != email || notification.TemplateName != template {
				continue
			}
			if err := json.Unmarshal(notification.Args, &payload); err != nil {
				return false
			}
			return true
		}
		return false
	}, fmt.Sprintf("%s notification should be sent to %s", template, email))
	return payload
}

func getChallenge(t *testing.T, app *bootstrap.TestApp, inviteID uuid.UUID) models.Challenge {
	t.Helper()
	var challenge models.Challenge
	require.NoError(t, app.DB().Where("invite_id = ? AND type = ?", inviteID, models.ChallengeTypeInvite).
		First(&challenge).Error)
	return challenge
}

func challengeID(t *testing.T, app *bootstrap.TestApp, inviteID uuid.UUID) uuid.UUID {
	t.Helper()
	return getChallenge(t, app, inviteID).ID
}

func assertUserDoesNotExist(t *testing.T, app *bootstrap.TestApp, email string) {
	t.Helper()
	var count int64
	require.NoError(t, app.DB().Model(&models.User{}).Where("email = ?", email).Count(&count).Error)
	assert.Zero(t, count)
}

func uniqueEmail(prefix string) string {
	return fmt.Sprintf("invite-%s-%s@example.com", prefix, uuid.NewString())
}

func wrongCode(code string) string {
	if strings.EqualFold(code, "ZZZZZZ") {
		return "AAAAAA"
	}
	return "ZZZZZZ"
}
