package biz

import (
	"context"
	"fmt"
	"time"

	tenantpb "github.com/Servora-Kit/servora/api/gen/go/tenant/service/v1"
	"github.com/Servora-Kit/servora/app/iam/service/internal/biz/entity"
	"github.com/Servora-Kit/servora/app/iam/service/internal/data/ent"
	"github.com/Servora-Kit/servora/pkg/logger"
)

type TenantRepo interface {
	Create(ctx context.Context, t *entity.Tenant) (*entity.Tenant, error)
	GetByID(ctx context.Context, id string) (*entity.Tenant, error)
	GetBySlug(ctx context.Context, slug string) (*entity.Tenant, error)
	GetByDomain(ctx context.Context, domain string) (*entity.Tenant, error)
	List(ctx context.Context, userID string, page, pageSize int32) ([]*entity.Tenant, int64, error)
	Update(ctx context.Context, t *entity.Tenant) (*entity.Tenant, error)
	Delete(ctx context.Context, id string) error
	Purge(ctx context.Context, id string) error

	AddMember(ctx context.Context, m *entity.TenantMember) (*entity.TenantMember, error)
	RemoveMember(ctx context.Context, tenantID, userID string) error
	GetMember(ctx context.Context, tenantID, userID string) (*entity.TenantMember, error)
	ListMembers(ctx context.Context, tenantID string, page, pageSize int32) ([]*entity.TenantMember, int64, error)
	UpdateMemberRole(ctx context.Context, tenantID, userID, role string) (*entity.TenantMember, error)
	UpdateMemberStatus(ctx context.Context, tenantID, userID, status string) (*entity.TenantMember, error)
	ListMembershipsByUserID(ctx context.Context, userID string) ([]*entity.TenantMember, error)
	GetPersonalTenantByUserID(ctx context.Context, userID string) (*entity.Tenant, error)
}

type TenantUsecase struct {
	repo    TenantRepo
	orgUC   *OrganizationUsecase
	projUC  *ProjectUsecase
	authz   AuthZRepo
	log     *logger.Helper
}

func NewTenantUsecase(repo TenantRepo, orgUC *OrganizationUsecase, projUC *ProjectUsecase, authz AuthZRepo, l logger.Logger) *TenantUsecase {
	return &TenantUsecase{
		repo:   repo,
		orgUC:  orgUC,
		projUC: projUC,
		authz:  authz,
		log:    logger.NewHelper(l, logger.WithModule("tenant/biz/iam-service")),
	}
}

func (uc *TenantUsecase) Create(ctx context.Context, t *entity.Tenant, creatorUserID string) (*entity.Tenant, error) {
	if _, err := uc.repo.GetBySlug(ctx, t.Slug); err == nil {
		return nil, tenantpb.ErrorTenantAlreadyExists("slug '%s' already taken", t.Slug)
	} else if !ent.IsNotFound(err) {
		uc.log.Errorf("check slug failed: %v", err)
		return nil, tenantpb.ErrorTenantCreateFailed("internal error")
	}

	if t.Kind == "" {
		t.Kind = "business"
	}
	if t.Status == "" {
		t.Status = "active"
	}

	created, err := uc.repo.Create(ctx, t)
	if err != nil {
		uc.log.Errorf("create tenant failed: %v", err)
		return nil, tenantpb.ErrorTenantCreateFailed("failed to create tenant")
	}

	member, err := uc.repo.AddMember(ctx, &entity.TenantMember{
		TenantID: created.ID,
		UserID:   creatorUserID,
		Role:     "owner",
		Status:   "active",
	})
	if err != nil {
		uc.log.Errorf("add owner member failed, rolling back tenant: %v", err)
		if delErr := uc.repo.Purge(ctx, created.ID); delErr != nil {
			uc.log.Errorf("rollback purge tenant failed: %v", delErr)
		}
		return nil, tenantpb.ErrorTenantCreateFailed("failed to add owner member")
	}

	if uc.authz != nil {
		if err := uc.authz.WriteTuples(ctx,
			Tuple{User: "user:" + creatorUserID, Relation: "owner", Object: "tenant:" + created.ID},
		); err != nil {
			uc.log.Errorf("write FGA tuples failed, rolling back: %v", err)
			_ = uc.repo.RemoveMember(ctx, created.ID, creatorUserID)
			if delErr := uc.repo.Purge(ctx, created.ID); delErr != nil {
				uc.log.Errorf("rollback purge tenant failed: %v", delErr)
			}
			return nil, tenantpb.ErrorTenantCreateFailed("failed to write authorization tuples")
		}
	}
	_ = member

	return created, nil
}

func (uc *TenantUsecase) CreateWithDefaults(ctx context.Context, t *entity.Tenant, creatorUserID string) (*entity.Tenant, error) {
	created, err := uc.Create(ctx, t, creatorUserID)
	if err != nil {
		return nil, err
	}

	defaultOrgSlug := t.Slug + "-default"
	defaultOrgName := t.Name + " Default"
	org, err := uc.orgUC.CreateDefault(ctx, creatorUserID, defaultOrgName, defaultOrgSlug, created.ID)
	if err != nil {
		uc.log.Errorf("create default org failed, rolling back tenant: %v", err)
		uc.rollbackTenantCreate(ctx, created.ID, creatorUserID)
		return nil, tenantpb.ErrorTenantCreateFailed("failed to create default organization")
	}

	defaultProjSlug := defaultOrgSlug + "-default"
	defaultProjName := "Default Project"
	if _, err := uc.projUC.CreateDefault(ctx, creatorUserID, org.ID, defaultProjName, defaultProjSlug); err != nil {
		uc.log.Errorf("create default project failed, rolling back: %v", err)
		_ = uc.orgUC.Purge(ctx, org.ID)
		uc.rollbackTenantCreate(ctx, created.ID, creatorUserID)
		return nil, tenantpb.ErrorTenantCreateFailed("failed to create default project")
	}

	return created, nil
}

func (uc *TenantUsecase) rollbackTenantCreate(ctx context.Context, tenantID, userID string) {
	if uc.authz != nil {
		_ = uc.authz.DeleteTuples(ctx,
			Tuple{User: "user:" + userID, Relation: "owner", Object: "tenant:" + tenantID},
		)
	}
	_ = uc.repo.RemoveMember(ctx, tenantID, userID)
	if delErr := uc.repo.Purge(ctx, tenantID); delErr != nil {
		uc.log.Errorf("rollback purge tenant failed: %v", delErr)
	}
}

func (uc *TenantUsecase) EnsurePersonalTenant(ctx context.Context, userID, userName string) (*entity.Tenant, error) {
	existing, err := uc.repo.GetPersonalTenantByUserID(ctx, userID)
	if err == nil {
		return existing, nil
	}
	if !ent.IsNotFound(err) {
		uc.log.Errorf("get personal tenant failed: %v", err)
		return nil, tenantpb.ErrorTenantCreateFailed("internal error")
	}

	slug := fmt.Sprintf("personal-%s", userID)
	name := fmt.Sprintf("%s's Space", userName)
	t := &entity.Tenant{
		Slug:   slug,
		Name:   name,
		Kind:   "personal",
		Status: "active",
	}
	return uc.CreateWithDefaults(ctx, t, userID)
}

func (uc *TenantUsecase) Get(ctx context.Context, id string) (*entity.Tenant, error) {
	t, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, tenantpb.ErrorTenantNotFound("tenant %s not found", id)
		}
		uc.log.Errorf("get tenant failed: %v", err)
		return nil, tenantpb.ErrorTenantCreateFailed("internal error")
	}
	return t, nil
}

func (uc *TenantUsecase) GetBySlug(ctx context.Context, slug string) (*entity.Tenant, error) {
	t, err := uc.repo.GetBySlug(ctx, slug)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, tenantpb.ErrorTenantNotFound("tenant with slug %s not found", slug)
		}
		uc.log.Errorf("get tenant by slug failed: %v", err)
		return nil, tenantpb.ErrorTenantCreateFailed("internal error")
	}
	return t, nil
}

func (uc *TenantUsecase) List(ctx context.Context, userID string, page, pageSize int32) ([]*entity.Tenant, int64, error) {
	tenants, total, err := uc.repo.List(ctx, userID, page, pageSize)
	if err != nil {
		uc.log.Errorf("list tenants failed: %v", err)
		return nil, 0, tenantpb.ErrorTenantCreateFailed("internal error")
	}
	return tenants, total, nil
}

func (uc *TenantUsecase) Update(ctx context.Context, t *entity.Tenant) (*entity.Tenant, error) {
	updated, err := uc.repo.Update(ctx, t)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, tenantpb.ErrorTenantNotFound("tenant %s not found", t.ID)
		}
		uc.log.Errorf("update tenant failed: %v", err)
		return nil, tenantpb.ErrorTenantUpdateFailed("failed to update tenant")
	}
	return updated, nil
}

func (uc *TenantUsecase) Delete(ctx context.Context, id string) error {
	t, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return tenantpb.ErrorTenantNotFound("tenant %s not found", id)
		}
		uc.log.Errorf("get tenant failed: %v", err)
		return tenantpb.ErrorTenantCreateFailed("internal error")
	}

	if t.Kind == "personal" {
		return tenantpb.ErrorTenantDeleteFailed("personal tenant cannot be deleted")
	}

	if err := uc.repo.Delete(ctx, id); err != nil {
		uc.log.Errorf("soft delete tenant failed: %v", err)
		return tenantpb.ErrorTenantDeleteFailed("failed to delete tenant")
	}
	return nil
}

func (uc *TenantUsecase) AddMember(ctx context.Context, tenantID, userID, role string) (*entity.TenantMember, error) {
	if err := ValidateTenantRole(role); err != nil {
		return nil, tenantpb.ErrorTenantCreateFailed("%v", err)
	}

	if _, err := uc.repo.GetMember(ctx, tenantID, userID); err == nil {
		return nil, tenantpb.ErrorTenantMemberAlreadyExists("user is already a member")
	}

	created, err := uc.repo.AddMember(ctx, &entity.TenantMember{
		TenantID: tenantID,
		UserID:   userID,
		Role:     role,
		Status:   "active",
	})
	if err != nil {
		uc.log.Errorf("add member failed: %v", err)
		return nil, tenantpb.ErrorTenantCreateFailed("failed to add member")
	}

	if uc.authz != nil {
		if err := uc.authz.WriteTuples(ctx,
			Tuple{User: "user:" + userID, Relation: role, Object: "tenant:" + tenantID},
		); err != nil {
			uc.log.Errorf("write FGA tuple failed, rolling back member: %v", err)
			if rbErr := uc.repo.RemoveMember(ctx, tenantID, userID); rbErr != nil {
				uc.log.Errorf("rollback remove member failed: %v", rbErr)
			}
			return nil, tenantpb.ErrorTenantCreateFailed("failed to write authorization tuple")
		}
	}
	return created, nil
}

func (uc *TenantUsecase) RemoveMember(ctx context.Context, tenantID, userID string) error {
	member, err := uc.repo.GetMember(ctx, tenantID, userID)
	if err != nil {
		return tenantpb.ErrorTenantMemberNotFound("member not found")
	}

	if err := uc.repo.RemoveMember(ctx, tenantID, userID); err != nil {
		uc.log.Errorf("remove member failed: %v", err)
		return tenantpb.ErrorTenantDeleteFailed("failed to remove member")
	}

	if uc.authz != nil {
		if err := uc.authz.DeleteTuples(ctx,
			Tuple{User: "user:" + userID, Relation: member.Role, Object: "tenant:" + tenantID},
		); err != nil {
			uc.log.Errorf("delete FGA tuple failed, rolling back: %v", err)
			if _, rbErr := uc.repo.AddMember(ctx, &entity.TenantMember{
				TenantID: tenantID,
				UserID:   userID,
				Role:     member.Role,
				Status:   member.Status,
			}); rbErr != nil {
				uc.log.Errorf("rollback re-add member failed: %v", rbErr)
			}
			return tenantpb.ErrorTenantDeleteFailed("failed to delete authorization tuple")
		}
	}
	return nil
}

func (uc *TenantUsecase) UpdateMemberRole(ctx context.Context, tenantID, userID, newRole string) (*entity.TenantMember, error) {
	if err := ValidateTenantRole(newRole); err != nil {
		return nil, tenantpb.ErrorTenantCreateFailed("%v", err)
	}

	oldMember, err := uc.repo.GetMember(ctx, tenantID, userID)
	if err != nil {
		return nil, tenantpb.ErrorTenantMemberNotFound("member not found")
	}

	updated, err := uc.repo.UpdateMemberRole(ctx, tenantID, userID, newRole)
	if err != nil {
		uc.log.Errorf("update member role failed: %v", err)
		return nil, tenantpb.ErrorTenantUpdateFailed("failed to update member role")
	}

	if uc.authz != nil && oldMember.Role != newRole {
		if err := uc.authz.DeleteTuples(ctx,
			Tuple{User: "user:" + userID, Relation: oldMember.Role, Object: "tenant:" + tenantID},
		); err != nil {
			uc.log.Errorf("delete old FGA tuple failed, rolling back: %v", err)
			if _, rbErr := uc.repo.UpdateMemberRole(ctx, tenantID, userID, oldMember.Role); rbErr != nil {
				uc.log.Errorf("rollback role update failed: %v", rbErr)
			}
			return nil, tenantpb.ErrorTenantUpdateFailed("failed to update authorization")
		}
		if err := uc.authz.WriteTuples(ctx,
			Tuple{User: "user:" + userID, Relation: newRole, Object: "tenant:" + tenantID},
		); err != nil {
			uc.log.Errorf("write new FGA tuple failed, rolling back: %v", err)
			_ = uc.authz.WriteTuples(ctx,
				Tuple{User: "user:" + userID, Relation: oldMember.Role, Object: "tenant:" + tenantID},
			)
			if _, rbErr := uc.repo.UpdateMemberRole(ctx, tenantID, userID, oldMember.Role); rbErr != nil {
				uc.log.Errorf("rollback role update failed: %v", rbErr)
			}
			return nil, tenantpb.ErrorTenantUpdateFailed("failed to update authorization")
		}
	}
	return updated, nil
}

func (uc *TenantUsecase) InviteMember(ctx context.Context, tenantID, userID, role string) (*entity.TenantMember, error) {
	if err := ValidateTenantRole(role); err != nil {
		return nil, tenantpb.ErrorTenantCreateFailed("%v", err)
	}

	t, err := uc.repo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, tenantpb.ErrorTenantNotFound("tenant not found")
	}
	if t.Kind == "personal" {
		return nil, tenantpb.ErrorTenantCreateFailed("personal tenant does not allow inviting members")
	}

	if _, err := uc.repo.GetMember(ctx, tenantID, userID); err == nil {
		return nil, tenantpb.ErrorTenantMemberAlreadyExists("user is already a member")
	}

	created, err := uc.repo.AddMember(ctx, &entity.TenantMember{
		TenantID: tenantID,
		UserID:   userID,
		Role:     role,
		Status:   "invited",
	})
	if err != nil {
		uc.log.Errorf("invite member failed: %v", err)
		return nil, tenantpb.ErrorTenantCreateFailed("failed to invite member")
	}

	if uc.authz != nil {
		if err := uc.authz.WriteTuples(ctx,
			Tuple{User: "user:" + userID, Relation: role, Object: "tenant:" + tenantID},
		); err != nil {
			uc.log.Errorf("write FGA tuple failed, rolling back invite: %v", err)
			if rbErr := uc.repo.RemoveMember(ctx, tenantID, userID); rbErr != nil {
				uc.log.Errorf("rollback remove member failed: %v", rbErr)
			}
			return nil, tenantpb.ErrorTenantCreateFailed("failed to write authorization tuple")
		}
	}
	return created, nil
}

func (uc *TenantUsecase) AcceptInvitation(ctx context.Context, tenantID, userID string) error {
	member, err := uc.repo.GetMember(ctx, tenantID, userID)
	if err != nil {
		return tenantpb.ErrorTenantMemberNotFound("invitation not found")
	}

	if member.Status == "active" {
		return nil
	}

	now := time.Now()
	if _, err := uc.repo.UpdateMemberStatus(ctx, tenantID, userID, "active"); err != nil {
		uc.log.Errorf("accept invitation failed: %v", err)
		return tenantpb.ErrorTenantUpdateFailed("failed to accept invitation")
	}
	_ = now

	return nil
}

func (uc *TenantUsecase) RejectInvitation(ctx context.Context, tenantID, userID string) error {
	member, err := uc.repo.GetMember(ctx, tenantID, userID)
	if err != nil {
		return tenantpb.ErrorTenantMemberNotFound("invitation not found")
	}

	if member.Status != "invited" {
		return tenantpb.ErrorTenantUpdateFailed("can only reject pending invitations")
	}

	if err := uc.repo.RemoveMember(ctx, tenantID, userID); err != nil {
		uc.log.Errorf("reject invitation - remove member failed: %v", err)
		return tenantpb.ErrorTenantDeleteFailed("failed to reject invitation")
	}

	if uc.authz != nil {
		if err := uc.authz.DeleteTuples(ctx,
			Tuple{User: "user:" + userID, Relation: member.Role, Object: "tenant:" + tenantID},
		); err != nil {
			uc.log.Warnf("delete FGA tuple on reject failed: %v", err)
		}
	}
	return nil
}

func (uc *TenantUsecase) ListMembers(ctx context.Context, tenantID string, page, pageSize int32) ([]*entity.TenantMember, int64, error) {
	members, total, err := uc.repo.ListMembers(ctx, tenantID, page, pageSize)
	if err != nil {
		uc.log.Errorf("list members failed: %v", err)
		return nil, 0, tenantpb.ErrorTenantCreateFailed("internal error")
	}
	return members, total, nil
}
