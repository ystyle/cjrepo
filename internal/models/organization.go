package models

import "time"

// Organization represents an organization
type Organization struct {
	ID          int64     `xorm:"pk autoincr 'id'" json:"id"`
	Name        string    `xorm:"VARCHAR(100) NOT NULL UNIQUE 'name'" json:"name"`
	DisplayName string    `xorm:"VARCHAR(255) 'display_name'" json:"display_name"`
	Description string    `xorm:"text 'description'" json:"description"`
	IsDefault   bool      `xorm:"DEFAULT false 'is_default'" json:"is_default"`
	CreatedAt   time.Time `xorm:"created 'created_at'" json:"created_at"`
	UpdatedAt   time.Time `xorm:"updated 'updated_at'" json:"updated_at"`
	DeletedAt   time.Time `xorm:"deleted 'deleted_at'" json:"deleted_at"`
}

// OrganizationMember represents a user's membership in an organization
type OrganizationMember struct {
	ID             int64     `xorm:"pk autoincr 'i_d'" json:"id"`
	OrganizationID int64     `xorm:"'organization_i_d' index" json:"organization_id"`
	UserID         int64     `xorm:"'user_i_d' index" json:"user_id"`
	CreatedAt      time.Time `xorm:"created 'created_at'" json:"created_at"`
}

// TableName returns the table name for Organization
func (Organization) TableName() string {
	return "organizations"
}

// TableName returns the table name for OrganizationMember
func (OrganizationMember) TableName() string {
	return "organization_members"
}
