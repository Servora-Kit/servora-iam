package server

import (
	"fmt"

	"github.com/Servora-Kit/servora/api/gen/go/conf/v1"
	"github.com/Servora-Kit/servora/pkg/governance/registry"
	"github.com/Servora-Kit/servora/pkg/governance/telemetry"
	"github.com/Servora-Kit/servora/pkg/jwks"
	"github.com/Servora-Kit/servora/pkg/logger"
	"github.com/Servora-Kit/servora/pkg/openfga"
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(registry.NewRegistrar, telemetry.NewMetrics, NewKeyManager, NewOpenFGAClient, NewGRPCMiddleware, NewGRPCServer, NewHTTPMiddleware, NewHealthHandler, NewHTTPServer)

func NewKeyManager(cfg *conf.App) (*jwks.KeyManager, error) {
	if cfg.Jwt == nil {
		return nil, fmt.Errorf("jwt configuration is required")
	}
	var opts []jwks.Option
	if cfg.Jwt.PrivateKeyPath != "" {
		opts = append(opts, jwks.WithPrivateKeyPath(cfg.Jwt.PrivateKeyPath))
	} else if cfg.Jwt.PrivateKeyPem != "" {
		opts = append(opts, jwks.WithPrivateKeyPEM([]byte(cfg.Jwt.PrivateKeyPem)))
	} else {
		return nil, fmt.Errorf("jwt: no private key configured (set private_key_path or private_key_pem)")
	}
	return jwks.NewKeyManager(opts...)
}

func NewOpenFGAClient(cfg *conf.App, l logger.Logger) *openfga.Client {
	if cfg.Openfga == nil || cfg.Openfga.ApiUrl == "" || cfg.Openfga.StoreId == "" {
		logger.NewHelper(l, logger.WithModule("openfga/server/iam-service")).
			Info("OpenFGA not configured, authorization checks disabled")
		return nil
	}
	c, err := openfga.NewClient(cfg.Openfga)
	if err != nil {
		logger.NewHelper(l, logger.WithModule("openfga/server/iam-service")).
			Warnf("failed to create OpenFGA client: %v", err)
		return nil
	}
	return c
}
