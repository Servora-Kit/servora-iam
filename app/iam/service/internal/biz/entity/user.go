package entity

import "time"

// UserProfile stores OIDC standard profile claims.
type UserProfile struct {
	Name       string
	GivenName  string
	FamilyName string
	Nickname   string
	Picture    string
	Gender     string
	Birthdate  string
	Zoneinfo   string
	Locale     string
}

type User struct {
	ID              string
	Username        string
	Email           string
	Password        string
	Phone           string
	PhoneVerified   bool
	Role            string
	Status          string
	EmailVerified   bool
	EmailVerifiedAt *time.Time
	Profile         UserProfile
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
