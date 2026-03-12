package entity

import "time"

type Organization struct {
	ID          string
	PlatformID  string
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
	CreatedAt      time.Time
}
