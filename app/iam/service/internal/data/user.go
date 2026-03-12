package data

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/google/uuid"

	"github.com/Servora-Kit/servora/app/iam/service/internal/biz"
	"github.com/Servora-Kit/servora/app/iam/service/internal/biz/entity"
	"github.com/Servora-Kit/servora/app/iam/service/internal/data/ent/user"
	"github.com/Servora-Kit/servora/pkg/helpers"
	"github.com/Servora-Kit/servora/pkg/logger"
)

type userRepo struct {
	data *Data
	log  *logger.Helper
}

func NewUserRepo(data *Data, l logger.Logger) biz.UserRepo {
	return &userRepo{
		data: data,
		log:  logger.NewHelper(l, logger.WithModule("user/data/iam-service")),
	}
}

func (r *userRepo) SaveUser(ctx context.Context, u *entity.User) (*entity.User, error) {
	if !helpers.BcryptIsHashed(u.Password) {
		bcryptPassword, err := helpers.BcryptHash(u.Password)
		if err != nil {
			return nil, err
		}
		u.Password = bcryptPassword
	}
	b := r.data.entClient.User.Create().
		SetName(u.Name).
		SetEmail(u.Email).
		SetPassword(u.Password).
		SetRole(u.Role)

	if u.ID != "" {
		uid, err := uuid.Parse(u.ID)
		if err != nil {
			return nil, fmt.Errorf("invalid user ID: %w", err)
		}
		b.SetID(uid)
	}

	created, err := b.Save(ctx)
	if err != nil {
		r.log.Errorf("SaveUser failed: %v", err)
		return nil, err
	}
	return entUserToEntity(created), nil
}

func (r *userRepo) GetUserById(ctx context.Context, id string) (*entity.User, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}
	entUser, err := r.data.entClient.User.Query().Where(user.IDEQ(uid)).Where(user.DeletedAtIsNil()).Only(ctx)
	if err != nil {
		return nil, err
	}
	return entUserToEntity(entUser), nil
}

func (r *userRepo) DeleteUser(ctx context.Context, u *entity.User) (*entity.User, error) {
	uid, err := uuid.Parse(u.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}
	err = r.data.entClient.User.UpdateOneID(uid).
		SetDeletedAt(time.Now()).
		Exec(ctx)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *userRepo) PurgeUser(ctx context.Context, u *entity.User) (*entity.User, error) {
	uid, err := uuid.Parse(u.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}
	err = r.data.entClient.User.DeleteOneID(uid).Exec(ctx)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *userRepo) RestoreUser(ctx context.Context, id string) (*entity.User, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}
	u, err := r.data.entClient.User.UpdateOneID(uid).
		ClearDeletedAt().
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return entUserToEntity(u), nil
}

func (r *userRepo) GetUserByIdIncludingDeleted(ctx context.Context, id string) (*entity.User, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}
	entUser, err := r.data.entClient.User.Query().Where(user.IDEQ(uid)).Only(ctx)
	if err != nil {
		return nil, err
	}
	return entUserToEntity(entUser), nil
}

func (r *userRepo) UpdateUser(ctx context.Context, u *entity.User) (*entity.User, error) {
	uid, err := uuid.Parse(u.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}
	if !helpers.BcryptIsHashed(u.Password) {
		bcryptPassword, err := helpers.BcryptHash(u.Password)
		if err != nil {
			return nil, err
		}
		u.Password = bcryptPassword
	}
	updated, err := r.data.entClient.User.UpdateOneID(uid).
		SetName(u.Name).
		SetEmail(u.Email).
		SetPassword(u.Password).
		SetRole(u.Role).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return entUserToEntity(updated), nil
}

func (r *userRepo) ListUsers(ctx context.Context, page int32, pageSize int32) ([]*entity.User, int64, error) {
	offset := int((page - 1) * pageSize)
	limit := int(pageSize)

	query := r.data.entClient.User.Query().Where(user.DeletedAtIsNil()).Order(user.ByID(sql.OrderDesc()))
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	entUsers, err := query.Offset(offset).Limit(limit).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	users := make([]*entity.User, 0, len(entUsers))
	for _, eu := range entUsers {
		users = append(users, entUserToEntity(eu))
	}

	return users, int64(total), nil
}
