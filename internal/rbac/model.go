package rbac

import (
	"github.com/casbin/casbin/v2/model"
)

// Initialize the model from a string.

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
m = g(r.sub, p.sub, r.dom) && r.dom == p.dom && r.obj_type == p.obj_type && r.obj == p.obj && keyMatch(r.act, p.act)
`
	m, err := model.NewModelFromString(data)
	if err != nil {
		panic(err)
	}
	return m
}
