// Package rbac provides role-based access control (RBAC) definitions and helper
// functions for initializing Casbin models, and managing actions, resources, roles,
// groups, and scopes.

package rbac

import (
	"github.com/casbin/casbin/v2/model"
)

// GetModel initializes and returns a Casbin model.Model configured for multi-tenant
// (domain) RBAC, with request, policy definitions, role hierarchy, effect, and matcher.
// It loads the model configuration from a hard-coded string and panics if parsing fails.
// Groups should be used to give access to specific resources (ex: bucket contributor)
// Roles should be used to give platform access (user / guest / admin)
// r: request = domain, subject, object type, object ID, action
// p: policy = domain, subject (role/group), object type, object ID, action
// g: grouping = user/group, role/group, domain
// e: effect = allow if any matching policy allows
// m: matcher checks role membership, domain, object type, object ID, and action
func GetModel() model.Model {
	data :=
		`
[request_definition]
r = dom, sub, obj_type, obj, act

[policy_definition]
p = dom, sub, obj_type, obj, act

[role_definition]
g = _, _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub, r.dom) && r.dom == p.dom && keyMatch(r.obj_type, p.obj_type) && keyMatch(r.obj, p.obj) && keyMatch(r.act, p.act)
`
	m, err := model.NewModelFromString(data)
	if err != nil {
		panic(err)
	}
	return m
}
