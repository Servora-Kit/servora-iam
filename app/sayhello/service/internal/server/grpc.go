package server

import (
	kgrpc "github.com/go-kratos/kratos/v2/transport/grpc"

	sayhellov1 "github.com/Servora-Kit/servora-iam/api/gen/go/servora/sayhello/service/v1"
	"github.com/Servora-Kit/servora-iam/app/sayhello/service/internal/service"
	"github.com/Servora-Kit/servora/api/gen/go/servora/conf/v1"
	"github.com/Servora-Kit/servora/obs/audit"
	logger "github.com/Servora-Kit/servora/obs/logging"
	"github.com/Servora-Kit/servora/obs/telemetry"
	svrgrpc "github.com/Servora-Kit/servora/transport/server/grpc"
	"github.com/Servora-Kit/servora/transport/server/middleware"
)

func NewGRPCServer(c *conf.Server, trace *conf.Trace, mtc *telemetry.Metrics, recorder *audit.Recorder, l logger.Logger, sayhello *service.SayHelloService) *kgrpc.Server {
	grpcLogger := logger.With(l, "grpc/server/sayhello")

	mw := middleware.NewChainBuilder(grpcLogger).
		WithTrace(trace).
		WithMetrics(mtc).
		WithoutRateLimit().
		Build()

	// Audit rules are generated from proto annotations in sayhello.proto.
	auditMw := audit.Audit(
		audit.WithRecorder(recorder),
		audit.WithRulesFunc(sayhellov1.AuditRules),
	)
	mw = append(mw, auditMw)

	builder := svrgrpc.NewBuilder().
		WithLogger(grpcLogger).
		WithMiddleware(mw...).
		WithServices(
			func(s *kgrpc.Server) { sayhellov1.RegisterSayHelloServiceServer(s, sayhello) },
		)
	if c != nil && c.Grpc != nil {
		builder.WithConfig(c.Grpc)
	}

	return builder.MustBuild()
}
