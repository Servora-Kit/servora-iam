package biz

import (
	"context"

	projectpb "github.com/Servora-Kit/servora/api/gen/go/project/service/v1"
	"github.com/Servora-Kit/servora/app/iam/service/internal/biz/entity"
	dataent "github.com/Servora-Kit/servora/app/iam/service/internal/data/ent"
	"github.com/Servora-Kit/servora/pkg/actor"
	"github.com/Servora-Kit/servora/pkg/logger"
	"github.com/Servora-Kit/servora/pkg/openfga"
)

type ProjectRepo interface {
	Create(ctx context.Context, p *entity.Project) (*entity.Project, error)
	GetByID(ctx context.Context, id string) (*entity.Project, error)
	ListByOrgID(ctx context.Context, orgID string, page, pageSize int32) ([]*entity.Project, int64, error)
	Update(ctx context.Context, p *entity.Project) (*entity.Project, error)
	Delete(ctx context.Context, id string) error
	AddMember(ctx context.Context, m *entity.ProjectMember) (*entity.ProjectMember, error)
	RemoveMember(ctx context.Context, projID, userID string) error
	ListMembers(ctx context.Context, projID string, page, pageSize int32) ([]*entity.ProjectMember, int64, error)
	GetMember(ctx context.Context, projID, userID string) (*entity.ProjectMember, error)
	UpdateMemberRole(ctx context.Context, projID, userID, role string) (*entity.ProjectMember, error)
}

type ProjectUsecase struct {
	repo ProjectRepo
	fga  *openfga.Client
	log  *logger.Helper
}

func NewProjectUsecase(repo ProjectRepo, fga *openfga.Client, l logger.Logger) *ProjectUsecase {
	return &ProjectUsecase{
		repo: repo,
		fga:  fga,
		log:  logger.NewHelper(l, logger.WithModule("project/biz/iam-service")),
	}
}

func (uc *ProjectUsecase) Create(ctx context.Context, p *entity.Project) (*entity.Project, error) {
	a, ok := actor.FromContext(ctx)
	if !ok || a.Type() != actor.TypeUser {
		return nil, projectpb.ErrorProjectCreateFailed("user not authenticated")
	}
	userID := a.ID()

	created, err := uc.repo.Create(ctx, p)
	if err != nil {
		return nil, projectpb.ErrorProjectCreateFailed("create: %v", err)
	}

	if uc.fga != nil {
		_ = uc.fga.WriteTuples(ctx,
			openfga.Tuple{User: "organization:" + p.OrganizationID, Relation: "organization", Object: "project:" + created.ID},
			openfga.Tuple{User: "user:" + userID, Relation: "admin", Object: "project:" + created.ID},
		)
	}

	if _, err := uc.repo.AddMember(ctx, &entity.ProjectMember{
		ProjectID: created.ID,
		UserID:    userID,
		Role:      "admin",
	}); err != nil {
		uc.log.Warnf("auto-add creator as admin failed: %v", err)
	}

	return created, nil
}

func (uc *ProjectUsecase) CreateDefault(ctx context.Context, userID, orgID, name, slug string) (*entity.Project, error) {
	p := &entity.Project{
		OrganizationID: orgID,
		Name:           name,
		Slug:           slug,
		Description:    "Default project",
	}
	created, err := uc.repo.Create(ctx, p)
	if err != nil {
		return nil, err
	}

	if uc.fga != nil {
		_ = uc.fga.WriteTuples(ctx,
			openfga.Tuple{User: "organization:" + orgID, Relation: "organization", Object: "project:" + created.ID},
			openfga.Tuple{User: "user:" + userID, Relation: "admin", Object: "project:" + created.ID},
		)
	}

	if _, err := uc.repo.AddMember(ctx, &entity.ProjectMember{
		ProjectID: created.ID,
		UserID:    userID,
		Role:      "admin",
	}); err != nil {
		uc.log.Warnf("auto-add admin failed: %v", err)
	}

	return created, nil
}

func (uc *ProjectUsecase) Get(ctx context.Context, id string) (*entity.Project, error) {
	p, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		if dataent.IsNotFound(err) {
			return nil, projectpb.ErrorProjectNotFound("project %s not found", id)
		}
		return nil, err
	}
	return p, nil
}

func (uc *ProjectUsecase) List(ctx context.Context, orgID string, page, pageSize int32) ([]*entity.Project, int64, error) {
	_, ok := actor.FromContext(ctx)
	if !ok {
		return nil, 0, projectpb.ErrorProjectNotFound("user not authenticated")
	}
	return uc.repo.ListByOrgID(ctx, orgID, page, pageSize)
}

func (uc *ProjectUsecase) Update(ctx context.Context, p *entity.Project) (*entity.Project, error) {
	updated, err := uc.repo.Update(ctx, p)
	if err != nil {
		if dataent.IsNotFound(err) {
			return nil, projectpb.ErrorProjectNotFound("project %s not found", p.ID)
		}
		return nil, projectpb.ErrorProjectUpdateFailed("update: %v", err)
	}
	return updated, nil
}

func (uc *ProjectUsecase) Delete(ctx context.Context, id string) error {
	if err := uc.repo.Delete(ctx, id); err != nil {
		if dataent.IsNotFound(err) {
			return projectpb.ErrorProjectNotFound("project %s not found", id)
		}
		return projectpb.ErrorProjectDeleteFailed("delete: %v", err)
	}
	return nil
}

func (uc *ProjectUsecase) AddMember(ctx context.Context, m *entity.ProjectMember) (*entity.ProjectMember, error) {
	if _, err := uc.repo.GetMember(ctx, m.ProjectID, m.UserID); err == nil {
		return nil, projectpb.ErrorProjectMemberAlreadyExists("user is already a member")
	}

	created, err := uc.repo.AddMember(ctx, m)
	if err != nil {
		return nil, err
	}

	if uc.fga != nil {
		_ = uc.fga.WriteTuples(ctx,
			openfga.Tuple{User: "user:" + m.UserID, Relation: m.Role, Object: "project:" + m.ProjectID},
		)
	}
	return created, nil
}

func (uc *ProjectUsecase) RemoveMember(ctx context.Context, projID, userID string) error {
	member, err := uc.repo.GetMember(ctx, projID, userID)
	if err != nil {
		return projectpb.ErrorProjectMemberNotFound("member not found")
	}

	if err := uc.repo.RemoveMember(ctx, projID, userID); err != nil {
		return err
	}

	if uc.fga != nil {
		_ = uc.fga.DeleteTuples(ctx,
			openfga.Tuple{User: "user:" + userID, Relation: member.Role, Object: "project:" + projID},
		)
	}
	return nil
}

func (uc *ProjectUsecase) ListMembers(ctx context.Context, projID string, page, pageSize int32) ([]*entity.ProjectMember, int64, error) {
	return uc.repo.ListMembers(ctx, projID, page, pageSize)
}

func (uc *ProjectUsecase) UpdateMemberRole(ctx context.Context, projID, userID, newRole string) (*entity.ProjectMember, error) {
	oldMember, err := uc.repo.GetMember(ctx, projID, userID)
	if err != nil {
		return nil, projectpb.ErrorProjectMemberNotFound("member not found")
	}

	updated, err := uc.repo.UpdateMemberRole(ctx, projID, userID, newRole)
	if err != nil {
		return nil, err
	}

	if uc.fga != nil && oldMember.Role != newRole {
		_ = uc.fga.DeleteTuples(ctx,
			openfga.Tuple{User: "user:" + userID, Relation: oldMember.Role, Object: "project:" + projID},
		)
		_ = uc.fga.WriteTuples(ctx,
			openfga.Tuple{User: "user:" + userID, Relation: newRole, Object: "project:" + projID},
		)
	}
	return updated, nil
}
