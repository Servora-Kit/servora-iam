package biz

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/golang-jwt/jwt/v5"

	authpb "github.com/Servora-Kit/servora/api/gen/go/auth/service/v1"
	"github.com/Servora-Kit/servora/api/gen/go/conf/v1"
	"github.com/Servora-Kit/servora/app/iam/service/internal/biz/entity"
	dataent "github.com/Servora-Kit/servora/app/iam/service/internal/data/ent"
	"github.com/Servora-Kit/servora/pkg/helpers"
	"github.com/Servora-Kit/servora/pkg/jwks"
	"github.com/Servora-Kit/servora/pkg/logger"
)

type AuthUsecase struct {
	repo    AuthRepo
	log     *logger.Helper
	cfg     *conf.App
	keyManager *jwks.KeyManager
	orgUC   *OrganizationUsecase
	projUC  *ProjectUsecase
}

func NewAuthUsecase(repo AuthRepo, l logger.Logger, cfg *conf.App, km *jwks.KeyManager, orgUC *OrganizationUsecase, projUC *ProjectUsecase) *AuthUsecase {
	return &AuthUsecase{
		repo:       repo,
		log:        logger.NewHelper(l, logger.WithModule("auth/biz/iam-service")),
		cfg:        cfg,
		keyManager: km,
		orgUC:      orgUC,
		projUC:     projUC,
	}
}

type UserClaims struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Role  string `json:"role"`
	Nonce string `json:"nonce"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

type TokenStore interface {
	SaveRefreshToken(ctx context.Context, userID string, token string, expiration time.Duration) error
	GetRefreshToken(ctx context.Context, token string) (string, error)
	DeleteRefreshToken(ctx context.Context, token string) error
	DeleteUserRefreshTokens(ctx context.Context, userID string) error
}

type AuthRepo interface {
	SaveUser(context.Context, *entity.User) (*entity.User, error)
	GetUserByEmail(context.Context, string) (*entity.User, error)
	GetUserByUserName(context.Context, string) (*entity.User, error)
	GetUserByID(context.Context, string) (*entity.User, error)
	TokenStore
}

func (uc *AuthUsecase) SignupByEmail(ctx context.Context, user *entity.User) (*entity.User, error) {
	existingUser, err := uc.repo.GetUserByUserName(ctx, user.Name)
	if err != nil && !dataent.IsNotFound(err) {
		return nil, authpb.ErrorUserNotFound("failed to check username: %v", err)
	}
	if existingUser != nil {
		return nil, authpb.ErrorUserAlreadyExists("username already exists")
	}

	existingEmail, err := uc.repo.GetUserByEmail(ctx, user.Email)
	if err != nil && !dataent.IsNotFound(err) {
		return nil, authpb.ErrorUserNotFound("failed to check email: %v", err)
	}
	if existingEmail != nil {
		return nil, authpb.ErrorUserAlreadyExists("email already exists")
	}

	user.Role = "user"
	createdUser, err := uc.repo.SaveUser(ctx, user)
	if err != nil {
		return nil, err
	}

	slug := helpers.Slugify(createdUser.Name)
	org, err := uc.orgUC.CreateDefault(ctx, createdUser.ID, createdUser.Name+"'s Organization", slug+"-org")
	if err != nil {
		uc.log.Warnf("auto-create default org failed for user %s: %v", createdUser.ID, err)
	} else {
		if _, err := uc.projUC.CreateDefault(ctx, createdUser.ID, org.ID, "Default Project", "default"); err != nil {
			uc.log.Warnf("auto-create default project failed for user %s: %v", createdUser.ID, err)
		}
	}

	return createdUser, nil
}

func (uc *AuthUsecase) generateAccessToken(claims *UserClaims) (string, error) {
	return uc.keyManager.Signer().Sign(claims)
}

func (uc *AuthUsecase) generateOpaqueToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (uc *AuthUsecase) LoginByEmailPassword(ctx context.Context, user *entity.User) (*TokenPair, error) {
	foundUser, err := uc.repo.GetUserByEmail(ctx, user.Email)
	if err != nil {
		return nil, authpb.ErrorUserNotFound("failed to get user: %v", err)
	}
	if foundUser == nil {
		uc.log.Warnf("user %s does not exist", user.Email)
		return nil, authpb.ErrorUserNotFound("user %s does not exist", user.Email)
	}
	if !helpers.BcryptCheck(user.Password, foundUser.Password) {
		return nil, authpb.ErrorIncorrectPassword("incorrect password for user: %s", user.Email)
	}

	nonce, err := uc.generateOpaqueToken()
	if err != nil {
		return nil, authpb.ErrorTokenGenerationFailed("failed to generate nonce: %v", err)
	}

	accessClaims := &UserClaims{
		ID:    foundUser.ID,
		Name:  foundUser.Name,
		Role:  foundUser.Role,
		Nonce: nonce,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   foundUser.ID,
			Audience:  jwt.ClaimStrings{uc.cfg.Jwt.Audience},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(uc.cfg.Jwt.AccessExpire) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    uc.cfg.Jwt.Issuer,
		},
	}

	accessToken, err := uc.generateAccessToken(accessClaims)
	if err != nil {
		return nil, authpb.ErrorTokenGenerationFailed("failed to generate access token: %v", err)
	}

	refreshToken, err := uc.generateOpaqueToken()
	if err != nil {
		return nil, authpb.ErrorTokenGenerationFailed("failed to generate refresh token: %v", err)
	}

	refreshExpirationTime := time.Duration(uc.cfg.Jwt.RefreshExpire) * time.Second
	if err := uc.repo.SaveRefreshToken(ctx, foundUser.ID, refreshToken, refreshExpirationTime); err != nil {
		return nil, authpb.ErrorTokenGenerationFailed("failed to save refresh token: %v", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(uc.cfg.Jwt.AccessExpire),
	}, nil
}

func (uc *AuthUsecase) RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
	userID, err := uc.repo.GetRefreshToken(ctx, refreshToken)
	if err != nil {
		uc.log.Warnf("Invalid refresh token: %v", err)
		return nil, authpb.ErrorInvalidRefreshToken("invalid or expired refresh token")
	}

	user, err := uc.repo.GetUserByID(ctx, userID)
	if err != nil {
		uc.log.Errorf("Failed to get user by ID: %v", err)
		return nil, authpb.ErrorUserNotFound("user not found: %v", err)
	}

	accessExpirationTime := time.Duration(uc.cfg.Jwt.AccessExpire) * time.Second
	nonce, err := uc.generateOpaqueToken()
	if err != nil {
		return nil, authpb.ErrorTokenGenerationFailed("failed to generate nonce: %v", err)
	}

	accessClaims := &UserClaims{
		ID:    user.ID,
		Name:  user.Name,
		Role:  user.Role,
		Nonce: nonce,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			Audience:  jwt.ClaimStrings{uc.cfg.Jwt.Audience},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(accessExpirationTime)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    uc.cfg.Jwt.Issuer,
		},
	}

	accessToken, err := uc.generateAccessToken(accessClaims)
	if err != nil {
		return nil, authpb.ErrorTokenGenerationFailed("failed to generate access token: %v", err)
	}

	newRefreshToken, err := uc.generateOpaqueToken()
	if err != nil {
		return nil, authpb.ErrorTokenGenerationFailed("failed to generate refresh token: %v", err)
	}

	if err := uc.repo.DeleteRefreshToken(ctx, refreshToken); err != nil {
		uc.log.Warnf("Failed to delete old refresh token: %v", err)
	}

	refreshExpirationTime := time.Duration(uc.cfg.Jwt.RefreshExpire) * time.Second
	if err := uc.repo.SaveRefreshToken(ctx, user.ID, newRefreshToken, refreshExpirationTime); err != nil {
		return nil, authpb.ErrorTokenGenerationFailed("failed to save refresh token: %v", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int64(uc.cfg.Jwt.AccessExpire),
	}, nil
}

func (uc *AuthUsecase) Logout(ctx context.Context, refreshToken string) error {
	if err := uc.repo.DeleteRefreshToken(ctx, refreshToken); err != nil {
		uc.log.Warnf("Failed to delete refresh token during logout: %v", err)
	}
	return nil
}
