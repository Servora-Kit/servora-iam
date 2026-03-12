package biz

import (
	"context"

	authpb "github.com/Servora-Kit/servora/api/gen/go/auth/service/v1"
	"github.com/Servora-Kit/servora/api/gen/go/conf/v1"
	paginationpb "github.com/Servora-Kit/servora/api/gen/go/pagination/v1"
	userpb "github.com/Servora-Kit/servora/api/gen/go/user/service/v1"
	"github.com/Servora-Kit/servora/app/iam/service/internal/biz/entity"
	"github.com/Servora-Kit/servora/pkg/actor"
	"github.com/Servora-Kit/servora/pkg/logger"
	"github.com/Servora-Kit/servora/pkg/openfga"
)

type UserRepo interface {
	SaveUser(context.Context, *entity.User) (*entity.User, error)
	GetUserById(context.Context, string) (*entity.User, error)
	DeleteUser(context.Context, *entity.User) (*entity.User, error)
	PurgeUser(context.Context, *entity.User) (*entity.User, error)
	RestoreUser(context.Context, string) (*entity.User, error)
	GetUserByIdIncludingDeleted(context.Context, string) (*entity.User, error)
	UpdateUser(context.Context, *entity.User) (*entity.User, error)
	ListUsers(context.Context, int32, int32) ([]*entity.User, int64, error)
}

type UserUsecase struct {
	repo     UserRepo
	log      *logger.Helper
	cfg      *conf.App
	authRepo AuthRepo
	orgRepo  OrganizationRepo
	projRepo ProjectRepo
	fga      *openfga.Client
	platID   string
}

func NewUserUsecase(
	repo UserRepo,
	l logger.Logger,
	cfg *conf.App,
	authRepo AuthRepo,
	orgRepo OrganizationRepo,
	projRepo ProjectRepo,
	fga *openfga.Client,
	platID PlatformRootID,
) *UserUsecase {
	return &UserUsecase{
		repo:     repo,
		log:      logger.NewHelper(l, logger.WithModule("user/biz/iam-service")),
		cfg:      cfg,
		authRepo: authRepo,
		orgRepo:  orgRepo,
		projRepo: projRepo,
		fga:      fga,
		platID:   string(platID),
	}
}

func (uc *UserUsecase) CurrentUserInfo(ctx context.Context) (*entity.User, error) {
	a, ok := actor.FromContext(ctx)
	if !ok || a.Type() != actor.TypeUser {
		return nil, authpb.ErrorUnauthorized("user not authenticated")
	}

	u, err := uc.repo.GetUserById(ctx, a.ID())
	if err != nil {
		return nil, userpb.ErrorUserNotFound("user not found")
	}
	return u, nil
}

func (uc *UserUsecase) UpdateUser(ctx context.Context, user *entity.User) (*entity.User, error) {
	origUser, err := uc.repo.GetUserById(ctx, user.ID)
	if err != nil {
		return nil, userpb.ErrorUserNotFound("user not found: %v", err)
	}

	if user.Name != origUser.Name {
		userWithSameName, err := uc.authRepo.GetUserByUserName(ctx, user.Name)
		if err != nil {
			return nil, authpb.ErrorUserNotFound("failed to check username: %v", err)
		}
		if userWithSameName != nil {
			return nil, authpb.ErrorUserAlreadyExists("username already exists")
		}
	}

	if user.Email != origUser.Email {
		userWithSameEmail, err := uc.authRepo.GetUserByEmail(ctx, user.Email)
		if err != nil {
			return nil, authpb.ErrorUserNotFound("failed to check email: %v", err)
		}
		if userWithSameEmail != nil {
			return nil, authpb.ErrorUserAlreadyExists("email already exists")
		}
	}

	updatedUser, err := uc.repo.UpdateUser(ctx, user)
	if err != nil {
		return nil, userpb.ErrorUpdateUserFailed("failed to update user: %v", err)
	}
	return updatedUser, nil
}

func (uc *UserUsecase) SaveUser(ctx context.Context, user *entity.User) (*entity.User, error) {
	if err := uc.checkUserExists(ctx, user); err != nil {
		return nil, err
	}

	savedUser, err := uc.repo.SaveUser(ctx, user)
	if err != nil {
		return nil, userpb.ErrorSaveUserFailed("failed to save user: %v", err)
	}
	return savedUser, nil
}

func (uc *UserUsecase) ListUsers(ctx context.Context, pagination *paginationpb.PaginationRequest) ([]*entity.User, *paginationpb.PaginationResponse, error) {
	page := int32(1)
	pageSize := int32(20)

	if pagination != nil {
		if pageMode := pagination.GetPage(); pageMode != nil {
			if pageMode.Page > 0 {
				page = pageMode.Page
			}
			if pageMode.PageSize > 0 {
				pageSize = pageMode.PageSize
			}
		}
	}

	users, total, err := uc.repo.ListUsers(ctx, page, pageSize)
	if err != nil {
		return nil, nil, userpb.ErrorUserNotFound("failed to list users: %v", err)
	}

	return users, &paginationpb.PaginationResponse{
		Mode: &paginationpb.PaginationResponse_Page{
			Page: &paginationpb.PagePaginationResponse{
				Total:    total,
				Page:     page,
				PageSize: pageSize,
			},
		},
	}, nil
}

func (uc *UserUsecase) DeleteUser(ctx context.Context, user *entity.User) (bool, error) {
	if _, err := uc.repo.GetUserById(ctx, user.ID); err != nil {
		return false, userpb.ErrorUserNotFound("user not found")
	}
	if _, err := uc.repo.DeleteUser(ctx, user); err != nil {
		return false, userpb.ErrorDeleteUserFailed("soft delete: %v", err)
	}
	return true, nil
}

func (uc *UserUsecase) PurgeUser(ctx context.Context, user *entity.User) (bool, error) {
	orgMemberships, _ := uc.orgRepo.ListMembershipsByUserID(ctx, user.ID)
	if uc.fga != nil && len(orgMemberships) > 0 {
		var tuples []openfga.Tuple
		for _, m := range orgMemberships {
			tuples = append(tuples,
				openfga.Tuple{User: "user:" + user.ID, Relation: m.Role, Object: "organization:" + m.OrganizationID},
				openfga.Tuple{User: "user:" + user.ID, Relation: "member", Object: "organization:" + m.OrganizationID},
			)
		}
		if err := uc.fga.DeleteTuples(ctx, tuples...); err != nil {
			uc.log.Warnf("delete user org FGA tuples: %v", err)
		}
	}
	if _, err := uc.orgRepo.DeleteMembershipsByUserID(ctx, user.ID); err != nil {
		uc.log.Warnf("delete user org memberships: %v", err)
	}

	projMemberships, _ := uc.projRepo.ListMembershipsByUserID(ctx, user.ID)
	if uc.fga != nil && len(projMemberships) > 0 {
		var tuples []openfga.Tuple
		for _, m := range projMemberships {
			tuples = append(tuples,
				openfga.Tuple{User: "user:" + user.ID, Relation: m.Role, Object: "project:" + m.ProjectID},
			)
		}
		if err := uc.fga.DeleteTuples(ctx, tuples...); err != nil {
			uc.log.Warnf("delete user project FGA tuples: %v", err)
		}
	}
	if _, err := uc.projRepo.DeleteMembershipsByUserID(ctx, user.ID); err != nil {
		uc.log.Warnf("delete user project memberships: %v", err)
	}

	if uc.fga != nil && uc.platID != "" {
		_ = uc.fga.DeleteTuples(ctx,
			openfga.Tuple{User: "user:" + user.ID, Relation: "admin", Object: "platform:" + uc.platID},
		)
	}

	if err := uc.authRepo.DeleteUserRefreshTokens(ctx, user.ID); err != nil {
		uc.log.Warnf("delete user refresh tokens: %v", err)
	}

	if _, err := uc.repo.PurgeUser(ctx, user); err != nil {
		return false, userpb.ErrorDeleteUserFailed("failed to delete user: %v", err)
	}
	return true, nil
}

func (uc *UserUsecase) RestoreUser(ctx context.Context, id string) (*entity.User, error) {
	if _, err := uc.repo.GetUserByIdIncludingDeleted(ctx, id); err != nil {
		return nil, userpb.ErrorUserNotFound("user not found")
	}
	return uc.repo.RestoreUser(ctx, id)
}

func (uc *UserUsecase) checkUserExists(ctx context.Context, user *entity.User) error {
	if existingUser, err := uc.authRepo.GetUserByUserName(ctx, user.Name); err != nil {
		return authpb.ErrorUserNotFound("failed to check username: %v", err)
	} else if existingUser != nil {
		return authpb.ErrorUserAlreadyExists("username already exists")
	}

	if existingEmail, err := uc.authRepo.GetUserByEmail(ctx, user.Email); err != nil {
		return authpb.ErrorUserNotFound("failed to check email: %v", err)
	} else if existingEmail != nil {
		return authpb.ErrorUserAlreadyExists("email already exists")
	}
	return nil
}
