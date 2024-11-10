package authorization

import (
	"github.com/casbin/casbin/v2/model"
)

// Initialize the model from a string.

func GetModel() model.Model {
	data :=
		`
[request_definition]
r = sub, obj_type, obj, act

[policy_definition]
p = sub, obj_type, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = (r.sub == p.sub && r.obj_type == p.obj_type && r.obj == p.obj && keyMatch(r.act, p.act)) || ((g(r.sub, p.sub) && r.sub != p.sub) && r.obj_type == p.obj_type && keyMatch(r.act, p.act))
`
	m, err := model.NewModelFromString(data)
	if err != nil {
		panic(err)
	}
	return m
}
