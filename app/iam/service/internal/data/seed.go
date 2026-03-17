package data

import (
	"context"
	"time"

	"github.com/google/uuid"

	iamconf "github.com/Servora-Kit/servora/api/gen/go/iam/conf/v1"
	"github.com/Servora-Kit/servora/app/iam/service/internal/biz"
	"github.com/Servora-Kit/servora/app/iam/service/internal/biz/entity"
	"github.com/Servora-Kit/servora/app/iam/service/internal/data/ent"
	"github.com/Servora-Kit/servora/app/iam/service/internal/data/ent/tenant"
	"github.com/Servora-Kit/servora/app/iam/service/internal/data/ent/tenantmember"
	"github.com/Servora-Kit/servora/app/iam/service/internal/data/ent/user"
	"github.com/Servora-Kit/servora/pkg/helpers"
	"github.com/Servora-Kit/servora/pkg/logger"
	"github.com/Servora-Kit/servora/pkg/openfga"
)

const (
	defaultBusinessTenantSlug = "default"
	platformObjectID          = "default"
)

// Seeder performs one-time data initialization for the IAM service.
// It runs as a Kratos BeforeStart hook, after all Wire DI is complete,
// giving it access to biz-layer use cases.
type Seeder struct {
	ec       *ent.Client
	tenantUC *biz.TenantUsecase
	fga      *openfga.Client
	seed     *iamconf.Biz_Seed
	log      *logger.Helper
}

func NewSeeder(ec *ent.Client, tenantUC *biz.TenantUsecase, fga *openfga.Client, bizConf *iamconf.Biz, l logger.Logger) *Seeder {
	return &Seeder{
		ec:       ec,
		tenantUC: tenantUC,
		fga:      fga,
		seed:     bizConf.GetSeed(),
		log:      logger.NewHelper(l, logger.WithModule("seed/data/iam-service")),
	}
}

// Run executes all seed steps. Each step is idempotent.
func (s *Seeder) Run(ctx context.Context) error {
	if s.seed == nil || s.seed.AdminEmail == "" {
		s.log.Info("no seed config provided, skipping")
		return nil
	}

	adminUser, err := s.ensureAdminUser(ctx)
	if err != nil {
		return err
	}
	userID := adminUser.ID.String()

	if err := s.ensureBusinessTenant(ctx, userID); err != nil {
		s.log.Warnf("ensure business tenant: %v", err)
	}

	if _, err := s.tenantUC.EnsurePersonalTenant(ctx, userID, adminUser.Name); err != nil {
		s.log.Warnf("ensure personal tenant: %v", err)
	}

	s.ensurePlatformAdmin(ctx, userID)
	return nil
}

// ensureAdminUser creates the seed admin user if it does not already exist.
func (s *Seeder) ensureAdminUser(ctx context.Context) (*ent.User, error) {
	existing, err := s.ec.User.Query().Where(user.EmailEQ(s.seed.AdminEmail)).Only(ctx)
	if err == nil {
		return existing, nil
	}
	if !ent.IsNotFound(err) {
		return nil, err
	}

	pw, err := helpers.BcryptHash(s.seed.AdminPassword)
	if err != nil {
		return nil, err
	}

	name := s.seed.AdminName
	if name == "" {
		name = "admin"
	}

	created, err := s.ec.User.Create().
		SetName(name).
		SetEmail(s.seed.AdminEmail).
		SetPassword(pw).
		SetRole("admin").
		Save(ctx)
	if err != nil {
		return nil, err
	}
	s.log.Infof("seeded admin user: %s", s.seed.AdminEmail)
	return created, nil
}

// ensureBusinessTenant creates the default business tenant (with default org
// and project) via TenantUsecase, or ensures the admin is a member if it
// already exists.
func (s *Seeder) ensureBusinessTenant(ctx context.Context, adminUserID string) error {
	existing, err := s.ec.Tenant.Query().
		Where(tenant.Slug(defaultBusinessTenantSlug)).
		Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return err
	}

	if ent.IsNotFound(err) {
		if _, err := s.tenantUC.CreateWithDefaults(ctx, &entity.Tenant{
			Slug:   defaultBusinessTenantSlug,
			Name:   "Default Tenant",
			Kind:   "business",
			Status: "active",
		}, adminUserID); err != nil {
			return err
		}
		s.log.Info("seeded default business tenant with defaults")
		return nil
	}

	tenantID := existing.ID.String()
	if err := s.ensureTenantMembership(ctx, tenantID, adminUserID); err != nil {
		return err
	}
	return nil
}

// ensureTenantMembership ensures the admin user is a member of the given
// tenant, creating the TenantMember record and FGA tuple if missing.
func (s *Seeder) ensureTenantMembership(ctx context.Context, tenantID, userID string) error {
	tid, _ := uuid.Parse(tenantID)
	uid, _ := uuid.Parse(userID)

	exists, err := s.ec.TenantMember.Query().
		Where(tenantmember.TenantIDEQ(tid), tenantmember.UserIDEQ(uid)).
		Exist(ctx)
	if err != nil {
		return err
	}
	if !exists {
		now := time.Now()
		if _, err := s.ec.TenantMember.Create().
			SetTenantID(tid).
			SetUserID(uid).
			SetRole(tenantmember.RoleOwner).
			SetStatus(tenantmember.StatusActive).
			SetJoinedAt(now).
			Save(ctx); err != nil {
			return err
		}
		s.log.Infof("seeded tenant member: user %s → tenant %s", userID, tenantID)
	}

	if s.fga != nil {
		allowed, err := s.fga.Check(ctx, userID, "owner", "tenant", tenantID)
		if err != nil {
			s.log.Warnf("FGA check tenant owner: %v", err)
		}
		if !allowed {
			if err := s.fga.WriteTuples(ctx, openfga.Tuple{
				User:     "user:" + userID,
				Relation: "owner",
				Object:   "tenant:" + tenantID,
			}); err != nil {
				s.log.Warnf("FGA write tenant owner tuple: %v", err)
			}
		}
	}
	return nil
}

// ensurePlatformAdmin writes the platform admin FGA tuple if not already present.
func (s *Seeder) ensurePlatformAdmin(ctx context.Context, userID string) {
	if s.fga == nil {
		return
	}

	allowed, err := s.fga.Check(ctx, userID, "admin", "platform", platformObjectID)
	if err != nil {
		s.log.Warnf("FGA check platform admin: %v", err)
	}
	if !allowed {
		if err := s.fga.WriteTuples(ctx, openfga.Tuple{
			User:     "user:" + userID,
			Relation: "admin",
			Object:   "platform:" + platformObjectID,
		}); err != nil {
			s.log.Warnf("FGA write platform admin tuple: %v", err)
		} else {
			s.log.Infof("seeded platform admin FGA tuple for user %s", userID)
		}
	}
}
