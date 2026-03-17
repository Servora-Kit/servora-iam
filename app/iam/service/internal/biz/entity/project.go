package entity

import "time"

type Project struct {
	ID             string
	OrganizationID string
	Name           string
	Slug           string
	Description    string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type ProjectMember struct {
	ID        string
	ProjectID string
	UserID    string
	UserName  string
	UserEmail string
	Role      string
	Status    string // "active" | "invited"
	CreatedAt time.Time
}
