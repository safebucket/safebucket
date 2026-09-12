package services

import (
	"testing"

	"github.com/safebucket/safebucket/internal/helpers"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestAuthenticateShareCookieScope(t *testing.T) {
	const password = "share-password"
	hash, err := helpers.CreateHash(password)
	require.NoError(t, err)
	customID := "project-files"
	service := PublicShareService{TokenSecret: "test-secret"}

	for _, identifier := range []*string{nil, &customID} {
		name := "ordinary"
		if identifier != nil {
			name = "custom"
		}
		t.Run(name, func(t *testing.T) {
			share := models.Share{ID: uuid.New(), CustomID: identifier, HashedPassword: hash}
			result, authErr := service.AuthenticateShare(true, zap.NewNop(), share, nil,
				models.ShareAuthBody{Password: password})
			require.NoError(t, authErr)
			require.Len(t, result.Cookies, 1)
			publicID := share.ID.String()
			if identifier != nil {
				publicID = *identifier
			}
			assert.Equal(t, "/api/v1/shares/"+publicID, result.Cookies[0].Path)
			claims, parseErr := helpers.ParseShareToken(service.TokenSecret, result.Cookies[0].Value)
			require.NoError(t, parseErr)
			assert.Equal(t, share.ID, claims.ShareID)
		})
	}
}
