package services

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/safebucket/safebucket/internal/cache"
	"github.com/safebucket/safebucket/internal/configuration"
	apierrors "github.com/safebucket/safebucket/internal/errors"
)

func enforceEmailIssuanceLimit(c cache.ICache, challengeType, email string, maxPerEmail int) error {
	key := fmt.Sprintf(
		configuration.CacheChallengeIssuanceEmailKey,
		challengeType,
		strings.ToLower(strings.TrimSpace(email)),
	)

	allowed, err := cache.RecordChallengeIssuance(c, key, maxPerEmail, configuration.SecurityChallengeIssuanceWindow)
	if err != nil {
		return apierrors.New(http.StatusServiceUnavailable, apierrors.CodeServiceUnavailable)
	}
	if !allowed {
		return apierrors.New(http.StatusTooManyRequests, apierrors.CodeRateLimitExceeded)
	}

	return nil
}
