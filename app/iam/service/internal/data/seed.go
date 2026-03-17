package data

import (
	"context"

	iamconf "github.com/Servora-Kit/servora/api/gen/go/iam/conf/v1"
	"github.com/Servora-Kit/servora/app/iam/service/internal/data/ent"
	"github.com/Servora-Kit/servora/app/iam/service/internal/data/ent/tenant"
	"github.com/Servora-Kit/servora/app/iam/service/internal/data/ent/user"
	"github.com/Servora-Kit/servora/pkg/helpers"
	"github.com/Servora-Kit/servora/pkg/logger"
	"github.com/Servora-Kit/servora/pkg/openfga"
)

const defaultTenantSlug = "default"
const platformObjectID = "default"

func seedTenant(ctx context.Context, ec *ent.Client) (string, error) {
	t, err := ec.Tenant.Query().Where(tenant.Slug(defaultTenantSlug)).Only(ctx)
	if err == nil {
		return t.ID.String(), nil
	}
	if !ent.IsNotFound(err) {
		return "", err
	}
	t, err = ec.Tenant.Create().
		SetSlug(defaultTenantSlug).
		SetName("Default Tenant").
		SetKind("business").
		SetStatus("active").
		Save(ctx)
	if err != nil {
		return "", err
	}
	return t.ID.String(), nil
}

func seedTenantAdmin(ctx context.Context, ec *ent.Client, seed *iamconf.Biz_Seed) error {
	if seed == nil || seed.AdminEmail == "" {
		return nil
	}

	exists, err := ec.User.Query().Where(user.EmailEQ(seed.AdminEmail)).Exist(ctx)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	pw, err := helpers.BcryptHash(seed.AdminPassword)
	if err != nil {
		return err
	}

	name := seed.AdminName
	if name == "" {
		name = "admin"
	}

	_, err = ec.User.Create().
		SetName(name).
		SetEmail(seed.AdminEmail).
		SetPassword(pw).
		SetRole("admin").
		Save(ctx)
	return err
}

func seedFGA(ctx context.Context, ec *ent.Client, fga *openfga.Client, tenantID string, seed *iamconf.Biz_Seed, l logger.Logger) {
	if fga == nil {
		return
	}
	seedLog := logger.NewHelper(l, logger.WithModule("seed/data/iam-service"))

	if seed == nil || seed.AdminEmail == "" {
		return
	}

	u, err := ec.User.Query().Where(user.EmailEQ(seed.AdminEmail)).Only(ctx)
	if err != nil {
		return
	}
	userID := u.ID.String()

	// platform:default admin tuple
	allowed, err := fga.Check(ctx, userID, "admin", "platform", platformObjectID)
	if err != nil {
		seedLog.Warnf("seed FGA check platform admin failed: %v", err)
	}
	if !allowed {
		if err := fga.WriteTuples(ctx, openfga.Tuple{
			User:     "user:" + userID,
			Relation: "admin",
			Object:   "platform:" + platformObjectID,
		}); err != nil {
			seedLog.Warnf("seed platform admin FGA tuple: %v", err)
		} else {
			seedLog.Infof("seeded platform admin FGA tuple for %s", seed.AdminEmail)
		}
	}

	// tenant owner tuple
	allowed, err = fga.Check(ctx, userID, "owner", "tenant", tenantID)
	if err != nil {
		seedLog.Warnf("seed FGA check tenant owner failed: %v", err)
	}
	if !allowed {
		if err := fga.WriteTuples(ctx, openfga.Tuple{
			User:     "user:" + userID,
			Relation: "owner",
			Object:   "tenant:" + tenantID,
		}); err != nil {
			seedLog.Warnf("seed tenant owner FGA tuple: %v", err)
		} else {
			seedLog.Infof("seeded tenant owner FGA tuple for %s", seed.AdminEmail)
		}
	}
}
