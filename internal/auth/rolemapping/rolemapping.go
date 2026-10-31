package rolemapping

import (
	"encoding/json"
	"errors"

	"github.com/safebucket/safebucket/internal/models"
)

var errMalformedClaim = errors.New("malformed groups claim")

type Mapping struct {
	groups map[string]struct{}
}

func New(adminGroups []string) *Mapping {
	if len(adminGroups) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(adminGroups))
	for _, g := range adminGroups {
		set[g] = struct{}{}
	}
	return &Mapping{groups: set}
}

func (m *Mapping) IsAdmin(groups []string) bool {
	for _, g := range groups {
		if _, ok := m.groups[g]; ok {
			return true
		}
	}
	return false
}

func Resolve(current models.Role, inAdminGroup bool) models.Role {
	switch {
	case current == models.RoleGuest:
		return current
	case inAdminGroup:
		return models.RoleAdmin
	case current == models.RoleAdmin:
		return models.RoleUser
	default:
		return current
	}
}

func readClaim(raw json.RawMessage) ([]string, error) {
	var arr []string
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr, nil
	}

	var scalar string
	if err := json.Unmarshal(raw, &scalar); err == nil {
		return []string{scalar}, nil
	}
	return nil, errMalformedClaim
}

func mapClaim(claims map[string]json.RawMessage, claim string) ([]string, bool, error) {
	raw, ok := claims[claim]
	if !ok {
		return nil, false, nil
	}
	if len(raw) == 0 || string(raw) == "null" {
		return []string{}, true, nil
	}

	groups, err := readClaim(raw)
	if err != nil {
		return nil, true, err
	}
	return groups, true, nil
}

func OIDCGroups(
	idToken map[string]json.RawMessage,
	userInfo map[string]json.RawMessage,
	claim string,
) ([]string, bool, error) {
	if idToken != nil {
		if groups, found, err := mapClaim(idToken, claim); found || err != nil {
			return groups, found, err
		}
	}
	if userInfo != nil {
		return mapClaim(userInfo, claim)
	}
	return nil, false, nil
}
