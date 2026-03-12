package middleware

import (
	"context"
	"fmt"
	"strings"

	gojwt "github.com/golang-jwt/jwt/v5"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"

	"github.com/Servora-Kit/servora/pkg/actor"
	jwtpkg "github.com/Servora-Kit/servora/pkg/jwt"
)

// tokenKey 用于在 context 中存储原始 Bearer token，便于下游传播。
type tokenKey struct{}

// TokenFromContext 从 context 中获取由 authn 中间件存储的原始 Bearer token。
func TokenFromContext(ctx context.Context) (string, bool) {
	t, ok := ctx.Value(tokenKey{}).(string)
	return t, ok
}

// UserClaimsMapper 用于将解析后的 JWT MapClaims 转换为 actor.Actor。
type UserClaimsMapper func(claims gojwt.MapClaims) (actor.Actor, error)

// AuthnOption 用于配置 authn 中间件的选项。
type AuthnOption func(*authnConfig)

type authnConfig struct {
	verifier     *jwtpkg.Verifier
	claimsMapper UserClaimsMapper
	errorHandler func(ctx context.Context, err error) error
}

// 配置 JWT 验证器
func WithVerifier(v *jwtpkg.Verifier) AuthnOption {
	return func(c *authnConfig) { c.verifier = v }
}

// 配置自定义声明转换函数
func WithClaimsMapper(m UserClaimsMapper) AuthnOption {
	return func(c *authnConfig) { c.claimsMapper = m }
}

// 配置认证错误处理函数
func WithAuthnErrorHandler(h func(ctx context.Context, err error) error) AuthnOption {
	return func(c *authnConfig) { c.errorHandler = h }
}

// 默认的声明转换函数，将 MapClaims 转换成 UserActor
func defaultClaimsMapper(claims gojwt.MapClaims) (actor.Actor, error) {
	id := claimString(claims, "sub")
	if id == "" {
		id = claimString(claims, "id")
	}
	name := claimString(claims, "name")
	email := claimString(claims, "email")

	metadata := make(map[string]string)
	if role := claimString(claims, "role"); role != "" {
		metadata["role"] = role
	}

	return actor.NewUserActor(id, name, email, metadata), nil
}

// 工具函数：从 claims 里取出字符串字段
func claimString(claims gojwt.MapClaims, key string) string {
	v, ok := claims[key]
	if !ok {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		return fmt.Sprintf("%.0f", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// Authn 用于创建一个 Kratos 的中间件，进行 JWT Token 验证并注入 actor.Actor 到请求上下文中。
//
// 如果没有携带 token，则注入匿名用户（AnonymousActor）。
// 可以与 selector.Server + WhiteList 配合实现开放路由。
func Authn(opts ...AuthnOption) middleware.Middleware {
	cfg := &authnConfig{
		claimsMapper: defaultClaimsMapper,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				ctx = actor.NewContext(ctx, actor.NewAnonymousActor())
				return handler(ctx, req)
			}

			tokenString := extractBearerToken(tr.RequestHeader().Get("Authorization"))
			if tokenString == "" {
				ctx = actor.NewContext(ctx, actor.NewAnonymousActor())
				return handler(ctx, req)
			}

			if cfg.verifier == nil {
				ctx = actor.NewContext(ctx, actor.NewAnonymousActor())
				ctx = context.WithValue(ctx, tokenKey{}, tokenString)
				return handler(ctx, req)
			}

			claims := gojwt.MapClaims{}
			if err := cfg.verifier.Verify(tokenString, claims); err != nil {
				if cfg.errorHandler != nil {
					return nil, cfg.errorHandler(ctx, err)
				}
				return nil, err
			}

			a, err := cfg.claimsMapper(claims)
			if err != nil {
				if cfg.errorHandler != nil {
					return nil, cfg.errorHandler(ctx, err)
				}
				return nil, err
			}

			ctx = actor.NewContext(ctx, a)
			ctx = context.WithValue(ctx, tokenKey{}, tokenString)
			return handler(ctx, req)
		}
	}
}

// 工具函数：从 Authorization 头部抽取 Bearer Token
func extractBearerToken(header string) string {
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}
	return parts[1]
}
