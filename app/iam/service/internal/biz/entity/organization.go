package entity

import "time"

type Organization struct {
	ID       string
	TenantID string
	Name        string
	Slug        string
	DisplayName string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type OrganizationMember struct {
	ID             string
	OrganizationID string
	UserID         string
	UserName       string
	UserEmail      string
	Role           string
	Status         string // "active" | "invited"
	CreatedAt      time.Time
}
