package service

import (
	"context"

	authpb "github.com/Servora-Kit/servora/api/gen/go/auth/service/v1"
	paginationpb "github.com/Servora-Kit/servora/api/gen/go/pagination/v1"
	userpb "github.com/Servora-Kit/servora/api/gen/go/user/service/v1"

	"github.com/Servora-Kit/servora/app/iam/service/internal/biz"
	"github.com/Servora-Kit/servora/app/iam/service/internal/biz/entity"
)

type UserService struct {
	userpb.UnimplementedUserServiceServer

	uc *biz.UserUsecase
}

func NewUserService(uc *biz.UserUsecase) *UserService {
	return &UserService{uc: uc}
}

func (s *UserService) CurrentUserInfo(ctx context.Context, req *userpb.CurrentUserInfoRequest) (*userpb.CurrentUserInfoResponse, error) {
	user, err := s.uc.CurrentUserInfo(ctx)
	if err != nil {
		return nil, err
	}
	return &userpb.CurrentUserInfoResponse{
		Id:   user.ID,
		Name: user.Name,
		Role: user.Role,
	}, nil
}

func (s *UserService) ListUsers(ctx context.Context, req *userpb.ListUsersRequest) (*userpb.ListUsersResponse, error) {
	users, pagination, err := s.uc.ListUsers(ctx, req.GetPagination())
	if err != nil {
		return nil, err
	}

	respUsers := make([]*userpb.UserInfo, 0, len(users))
	for _, user := range users {
		respUsers = append(respUsers, &userpb.UserInfo{
			Id:    user.ID,
			Name:  user.Name,
			Email: user.Email,
			Role:  user.Role,
		})
	}

	if pagination == nil {
		pagination = &paginationpb.PaginationResponse{
			Mode: &paginationpb.PaginationResponse_Page{
				Page: &paginationpb.PagePaginationResponse{},
			},
		}
	}

	return &userpb.ListUsersResponse{
		Users:      respUsers,
		Pagination: pagination,
	}, nil
}

func (s *UserService) UpdateUser(ctx context.Context, req *userpb.UpdateUserRequest) (*userpb.UpdateUserResponse, error) {
	currentUser, err := s.uc.CurrentUserInfo(ctx)
	if err != nil {
		return nil, err
	}

	switch currentUser.Role {
	case "user":
		if currentUser.ID != req.Id {
			return nil, authpb.ErrorUnauthorized("you only can update your own information")
		}
		if req.Role != "" && req.Role != "user" {
			return nil, authpb.ErrorUnauthorized("you do not have permission to change your role")
		}
	case "admin":
		// admin can update any user
	case "operator":
		// operator can update any user
	default:
		return nil, authpb.ErrorUnauthorized("insufficient permissions")
	}

	user := &entity.User{
		ID:       req.Id,
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
		Role:     req.Role,
	}
	_, err = s.uc.UpdateUser(ctx, user)
	if err != nil {
		return nil, err
	}
	return &userpb.UpdateUserResponse{
		Success: "true",
	}, nil
}

func (s *UserService) SaveUser(ctx context.Context, req *userpb.SaveUserRequest) (*userpb.SaveUserResponse, error) {
	user := &entity.User{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
		Role:     req.Role,
	}
	user, err := s.uc.SaveUser(ctx, user)
	if err != nil {
		return nil, err
	}
	return &userpb.SaveUserResponse{Id: user.ID}, nil
}

func (s *UserService) DeleteUser(ctx context.Context, req *userpb.DeleteUserRequest) (*userpb.DeleteUserResponse, error) {
	success, err := s.uc.DeleteUser(ctx, &entity.User{
		ID: req.Id,
	})
	if err != nil {
		return nil, err
	}
	return &userpb.DeleteUserResponse{Success: success}, err
}

func (s *UserService) PurgeUser(ctx context.Context, req *userpb.PurgeUserRequest) (*userpb.PurgeUserResponse, error) {
	success, err := s.uc.PurgeUser(ctx, &entity.User{ID: req.Id})
	if err != nil {
		return nil, err
	}
	return &userpb.PurgeUserResponse{Success: success}, nil
}

func (s *UserService) RestoreUser(ctx context.Context, req *userpb.RestoreUserRequest) (*userpb.RestoreUserResponse, error) {
	u, err := s.uc.RestoreUser(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return &userpb.RestoreUserResponse{User: &userpb.UserInfo{
		Id:    u.ID,
		Name:  u.Name,
		Email: u.Email,
		Role:  u.Role,
	}}, nil
}
