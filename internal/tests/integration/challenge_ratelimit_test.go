//go:build integration

package integration

import (
	"net/http"
	"testing"

	"github.com/safebucket/safebucket/internal/configuration"
	"github.com/safebucket/safebucket/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPasswordResetIssuanceRateLimit(t *testing.T) {
	for _, scenario := range ActiveScenarios() {
		t.Run(scenario, func(t *testing.T) {
			app := BootScenario(t, scenario)

			body := models.PasswordResetRequestBody{Email: "issuance-cap@example.com"}

			for i := 1; i <= configuration.SecurityPasswordResetMaxPerEmailPerHour; i++ {
				status := app.DoStatus(t, http.MethodPost, "/api/v1/auth/reset-password", "", body)
				require.Equalf(t, http.StatusCreated, status, "request %d should be accepted", i)
			}

			status := app.DoStatus(t, http.MethodPost, "/api/v1/auth/reset-password", "", body)
			assert.Equal(t, http.StatusTooManyRequests, status,
				"issuance beyond the per-email cap must be rate limited")
		})
	}
}
