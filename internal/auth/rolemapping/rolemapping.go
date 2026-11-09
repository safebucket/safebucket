package rolemapping

import (
	"encoding/json"
	"errors"

	"github.com/safebucket/safebucket/internal/models"
)

var (
	ErrGroupsClaimMissing = errors.New("groups claim missing")
	errMalformedClaim     = errors.New("malformed groups claim")
)

type claimsSource interface {
	Claims(v any) error
}

// OIDCGroups reads the groups claim from the first source that has it.
func OIDCGroups(claim string, sources ...claimsSource) ([]string, error) {
	for _, source := range sources {
		claims := map[string]json.RawMessage{}
		if err := source.Claims(&claims); err != nil {
			return nil, err
		}
		if raw, ok := claims[claim]; ok {
			return readClaim(raw)
		}
	}
	return nil, ErrGroupsClaimMissing
}

func Resolve(current models.Role, groups []string, adminGroups map[string]struct{}) models.Role {
	if current == models.RoleGuest {
		return current
	}
	for _, group := range groups {
		if _, ok := adminGroups[group]; ok {
			return models.RoleAdmin
		}
	}
	return models.RoleUser
}

func readClaim(raw json.RawMessage) ([]string, error) {
	var groups []string
	if err := json.Unmarshal(raw, &groups); err == nil {
		return groups, nil
	}

	var group string
	if err := json.Unmarshal(raw, &group); err == nil {
		return []string{group}, nil
	}
	return nil, errMalformedClaim
}
