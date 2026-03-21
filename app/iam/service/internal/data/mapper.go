package data

import (
	apppb "github.com/Servora-Kit/servora/api/gen/go/application/service/v1"
	userpb "github.com/Servora-Kit/servora/api/gen/go/user/service/v1"
	"github.com/Servora-Kit/servora/app/iam/service/internal/data/ent"
	"github.com/Servora-Kit/servora/pkg/mapper"
)

func newUserMapper() *mapper.CopierMapper[userpb.User, ent.User] {
	m := mapper.NewCopierMapper[userpb.User, ent.User]()

	hooks := mapper.NewHookRegistry()
	hooks.Register("user_profile")

	if err := mapper.ApplyPlan(userpb.UserMapperPlan(), m, mapper.DefaultPresets(), hooks); err != nil {
		panic("mapper: apply user plan: " + err.Error())
	}
	return m
}

// mapUser converts ent.User to proto User with profile post-processing.
// Profile uses map[string]any in ent but structured *UserProfile in proto,
// which copier cannot handle -- so we apply it after the copier pass.
func mapUser(m *mapper.CopierMapper[userpb.User, ent.User], u *ent.User) *userpb.User {
	pb := m.MustToProto(u)
	if pb != nil && u.Profile != nil {
		pb.Profile = profileFromJSON(u.Profile)
	}
	return pb
}

func mapUsers(m *mapper.CopierMapper[userpb.User, ent.User], users []*ent.User) []*userpb.User {
	result := make([]*userpb.User, 0, len(users))
	for _, u := range users {
		if u != nil {
			result = append(result, mapUser(m, u))
		}
	}
	return result
}

func profileFromJSON(m map[string]any) *userpb.UserProfile {
	if m == nil {
		return nil
	}
	p := &userpb.UserProfile{}
	if v, ok := m["name"].(string); ok {
		p.Name = v
	}
	if v, ok := m["given_name"].(string); ok {
		p.GivenName = v
	}
	if v, ok := m["family_name"].(string); ok {
		p.FamilyName = v
	}
	if v, ok := m["nickname"].(string); ok {
		p.Nickname = v
	}
	if v, ok := m["picture"].(string); ok {
		p.Picture = v
	}
	if v, ok := m["gender"].(string); ok {
		p.Gender = v
	}
	if v, ok := m["birthdate"].(string); ok {
		p.Birthdate = v
	}
	if v, ok := m["zoneinfo"].(string); ok {
		p.Zoneinfo = v
	}
	if v, ok := m["locale"].(string); ok {
		p.Locale = v
	}
	return p
}

func newApplicationMapper() *mapper.CopierMapper[apppb.Application, ent.Application] {
	m := mapper.NewCopierMapper[apppb.Application, ent.Application]()

	hooks := mapper.NewHookRegistry()
	if err := mapper.ApplyPlan(apppb.ApplicationMapperPlan(), m, mapper.DefaultPresets(), hooks); err != nil {
		panic("mapper: apply application plan: " + err.Error())
	}
	return m
}
