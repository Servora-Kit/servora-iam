package service

import (
	apppb "github.com/Servora-Kit/servora/api/gen/go/application/service/v1"
	userpb "github.com/Servora-Kit/servora/api/gen/go/user/service/v1"
	"github.com/Servora-Kit/servora/app/iam/service/internal/biz/entity"
	"github.com/Servora-Kit/servora/pkg/mapper"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var userInfoMapper = mapper.NewForwardMapper(func(u *entity.User) *userpb.UserInfo {
	info := &userpb.UserInfo{
		Id:            u.ID,
		Username:      u.Username,
		Email:         u.Email,
		Role:          u.Role,
		Phone:         u.Phone,
		PhoneVerified: u.PhoneVerified,
		Status:        u.Status,
		EmailVerified: u.EmailVerified,
	}
	info.Profile = &userpb.UserProfile{
		Name:       u.Profile.Name,
		GivenName:  u.Profile.GivenName,
		FamilyName: u.Profile.FamilyName,
		Nickname:   u.Profile.Nickname,
		Picture:    u.Profile.Picture,
		Gender:     u.Profile.Gender,
		Birthdate:  u.Profile.Birthdate,
		Zoneinfo:   u.Profile.Zoneinfo,
		Locale:     u.Profile.Locale,
	}
	return info
})

var applicationInfoMapper = mapper.NewForwardMapper(func(a *entity.Application) *apppb.ApplicationInfo {
	return &apppb.ApplicationInfo{
		Id:              a.ID,
		ClientId:        a.ClientID,
		Name:            a.Name,
		RedirectUris:    a.RedirectURIs,
		Scopes:          a.Scopes,
		GrantTypes:      a.GrantTypes,
		ApplicationType: a.ApplicationType,
		AccessTokenType: a.AccessTokenType,
		Type:            a.Type,
		IdTokenLifetime: int32(a.IDTokenLifetime.Seconds()),
		CreatedAt:       timestamppb.New(a.CreatedAt),
		UpdatedAt:       timestamppb.New(a.UpdatedAt),
	}
})
