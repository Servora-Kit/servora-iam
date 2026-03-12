package data

import (
	"context"
	"fmt"

	"entgo.io/ent/dialect/sql"
	"github.com/google/uuid"

	"github.com/Servora-Kit/servora/app/iam/service/internal/biz"
	"github.com/Servora-Kit/servora/app/iam/service/internal/biz/entity"
	"github.com/Servora-Kit/servora/app/iam/service/internal/data/ent"
	"github.com/Servora-Kit/servora/app/iam/service/internal/data/ent/project"
	"github.com/Servora-Kit/servora/app/iam/service/internal/data/ent/projectmember"
	"github.com/Servora-Kit/servora/app/iam/service/internal/data/ent/user"
	"github.com/Servora-Kit/servora/pkg/logger"
)

type projectRepo struct {
	data *Data
	log  *logger.Helper
}

func NewProjectRepo(data *Data, l logger.Logger) biz.ProjectRepo {
	return &projectRepo{
		data: data,
		log:  logger.NewHelper(l, logger.WithModule("project/data/iam-service")),
	}
}

func (r *projectRepo) Create(ctx context.Context, p *entity.Project) (*entity.Project, error) {
	orgID, err := uuid.Parse(p.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("invalid organization ID: %w", err)
	}
	b := r.data.entClient.Project.Create().
		SetOrganizationID(orgID).
		SetName(p.Name).
		SetSlug(p.Slug)
	if p.Description != "" {
		b.SetDescription(p.Description)
	}
	created, err := b.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create project: %w", err)
	}
	return entProjectToEntity(created), nil
}

func (r *projectRepo) GetByID(ctx context.Context, id string) (*entity.Project, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}
	p, err := r.data.entClient.Project.Query().
		Where(project.IDEQ(uid)).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return entProjectToEntity(p), nil
}

func (r *projectRepo) ListByOrgID(ctx context.Context, orgID string, page, pageSize int32) ([]*entity.Project, int64, error) {
	oid, err := uuid.Parse(orgID)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid organization ID: %w", err)
	}
	query := r.data.entClient.Project.Query().
		Where(project.OrganizationIDEQ(oid)).
		Order(project.ByCreatedAt(sql.OrderDesc()))

	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	offset := int((page - 1) * pageSize)
	projects, err := query.Offset(offset).Limit(int(pageSize)).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*entity.Project, 0, len(projects))
	for _, p := range projects {
		result = append(result, entProjectToEntity(p))
	}
	return result, int64(total), nil
}

func (r *projectRepo) Update(ctx context.Context, p *entity.Project) (*entity.Project, error) {
	uid, err := uuid.Parse(p.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}
	b := r.data.entClient.Project.UpdateOneID(uid)
	if p.Name != "" {
		b.SetName(p.Name)
	}
	if p.Description != "" {
		b.SetDescription(p.Description)
	}
	updated, err := b.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("update project: %w", err)
	}
	return entProjectToEntity(updated), nil
}

func (r *projectRepo) Delete(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid project ID: %w", err)
	}
	return r.data.entClient.Project.DeleteOneID(uid).Exec(ctx)
}

func (r *projectRepo) AddMember(ctx context.Context, m *entity.ProjectMember) (*entity.ProjectMember, error) {
	projID, err := uuid.Parse(m.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}
	userID, err := uuid.Parse(m.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}
	created, err := r.data.entClient.ProjectMember.Create().
		SetProjectID(projID).
		SetUserID(userID).
		SetRole(m.Role).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("add project member: %w", err)
	}
	return r.enrichMember(ctx, created)
}

func (r *projectRepo) RemoveMember(ctx context.Context, projID, userID string) error {
	pid, err := uuid.Parse(projID)
	if err != nil {
		return fmt.Errorf("invalid project ID: %w", err)
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}
	_, err = r.data.entClient.ProjectMember.Delete().
		Where(
			projectmember.ProjectIDEQ(pid),
			projectmember.UserIDEQ(uid),
		).Exec(ctx)
	return err
}

func (r *projectRepo) ListMembers(ctx context.Context, projID string, page, pageSize int32) ([]*entity.ProjectMember, int64, error) {
	pid, err := uuid.Parse(projID)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid project ID: %w", err)
	}
	query := r.data.entClient.ProjectMember.Query().
		Where(projectmember.ProjectIDEQ(pid)).
		Order(projectmember.ByCreatedAt(sql.OrderDesc()))

	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	offset := int((page - 1) * pageSize)
	members, err := query.Offset(offset).Limit(int(pageSize)).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*entity.ProjectMember, 0, len(members))
	for _, m := range members {
		enriched, err := r.enrichMember(ctx, m)
		if err != nil {
			r.log.Warnf("enrich member %s failed: %v", m.ID, err)
			continue
		}
		result = append(result, enriched)
	}
	return result, int64(total), nil
}

func (r *projectRepo) GetMember(ctx context.Context, projID, userID string) (*entity.ProjectMember, error) {
	pid, err := uuid.Parse(projID)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}
	m, err := r.data.entClient.ProjectMember.Query().
		Where(
			projectmember.ProjectIDEQ(pid),
			projectmember.UserIDEQ(uid),
		).Only(ctx)
	if err != nil {
		return nil, err
	}
	return r.enrichMember(ctx, m)
}

func (r *projectRepo) UpdateMemberRole(ctx context.Context, projID, userID, role string) (*entity.ProjectMember, error) {
	pid, err := uuid.Parse(projID)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %w", err)
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}
	affected, err := r.data.entClient.ProjectMember.Update().
		Where(
			projectmember.ProjectIDEQ(pid),
			projectmember.UserIDEQ(uid),
		).
		SetRole(role).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("update member role: %w", err)
	}
	if affected == 0 {
		return nil, fmt.Errorf("member not found")
	}
	return r.GetMember(ctx, projID, userID)
}

func (r *projectRepo) enrichMember(ctx context.Context, m *ent.ProjectMember) (*entity.ProjectMember, error) {
	u, err := r.data.entClient.User.Query().Where(user.IDEQ(m.UserID)).Only(ctx)
	if err != nil {
		return &entity.ProjectMember{
			ID:        m.ID.String(),
			ProjectID: m.ProjectID.String(),
			UserID:    m.UserID.String(),
			Role:      m.Role,
			CreatedAt: m.CreatedAt,
		}, nil
	}
	return &entity.ProjectMember{
		ID:        m.ID.String(),
		ProjectID: m.ProjectID.String(),
		UserID:    m.UserID.String(),
		UserName:  u.Name,
		UserEmail: u.Email,
		Role:      m.Role,
		CreatedAt: m.CreatedAt,
	}, nil
}

func entProjectToEntity(p *ent.Project) *entity.Project {
	e := &entity.Project{
		ID:             p.ID.String(),
		OrganizationID: p.OrganizationID.String(),
		Name:           p.Name,
		Slug:           p.Slug,
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
	}
	if p.Description != nil {
		e.Description = *p.Description
	}
	return e
}
