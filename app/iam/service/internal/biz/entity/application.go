package entity

import "time"

// Application represents an OAuth2/OIDC client registered in IAM.
// Type distinguishes usage: "web" | "native" | "m2m".
type Application struct {
	ID               string
	ClientID         string
	ClientSecretHash string
	Name             string
	RedirectURIs     []string
	Scopes           []string
	GrantTypes       []string
	ApplicationType  string
	AccessTokenType  string
	Type             string
	IDTokenLifetime  time.Duration
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
