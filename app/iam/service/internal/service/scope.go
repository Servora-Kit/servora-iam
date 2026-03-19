package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/errors"

	"github.com/Servora-Kit/servora/app/iam/service/internal/biz"
	"github.com/Servora-Kit/servora/pkg/actor"
)

// requireAuthenticatedUser extracts the authenticated user ID from context.
func requireAuthenticatedUser(ctx context.Context) (userID string, err error) {
	a, ok := actor.FromContext(ctx)
	if !ok || a.Type() != actor.TypeUser {
		return "", errors.Unauthorized("UNAUTHORIZED", "unauthorized")
	}
	return a.ID(), nil
}

// errForbidden returns a standardized Forbidden error with the given reason.
func errForbidden(reason string) error {
	return errors.Forbidden("FORBIDDEN", reason)
}

// checkPlatformAdmin returns an error if the authenticated user is not a platform admin.
func checkPlatformAdmin(ctx context.Context, userUC *biz.UserUsecase) error {
	callerID, err := requireAuthenticatedUser(ctx)
	if err != nil {
		return err
	}
	user, err := userUC.CurrentUserInfo(ctx, callerID)
	if err != nil {
		return err
	}
	if user.Role != "admin" {
		return errors.Forbidden("FORBIDDEN", "only platform admin can perform this action")
	}
	return nil
}
