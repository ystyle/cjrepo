package models

import "time"

// Team 团队
type Team struct {
	ID          int64     `xorm:"pk autoincr 'i_d'" json:"id"`
	Name        string    `xorm:"VARCHAR(100) NOT NULL UNIQUE 'name'" json:"name"`
	DisplayName string    `xorm:"VARCHAR(255) 'display_name'" json:"display_name"`
	Description string    `xorm:"TEXT 'description'" json:"description"`
	Permission  string    `xorm:"VARCHAR(20) NOT NULL 'permission'" json:"permission"` // read/write/overwrite
	DeletedAt   time.Time `xorm:"deleted 'deleted_at'" json:"deleted_at"`
	CreatedAt   time.Time `xorm:"created 'created_at'" json:"created_at"`
	UpdatedAt   time.Time `xorm:"updated 'updated_at'" json:"updated_at"`
}

// TeamOrganization 团队可操作的组织
type TeamOrganization struct {
	ID             int64     `xorm:"pk autoincr 'i_d'" json:"id"`
	TeamID         int64     `xorm:"NOT NULL INDEX 'team_i_d'" json:"team_id"`
	OrganizationID *int64    `xorm:"INDEX 'organization_i_d'" json:"organization_id"` // NULL 表示无组织包
	CreatedAt      time.Time `xorm:"created 'created_at'" json:"created_at"`
}

// TeamPackage 团队关联的特定包（权限走 team.permission）
type TeamPackage struct {
	ID           int64     `xorm:"pk autoincr 'i_d'" json:"id"`
	TeamID       int64     `xorm:"NOT NULL INDEX 'team_i_d'" json:"team_id"`
	Organization string    `xorm:"INDEX 'organization'" json:"organization"` // 空字符串表示无组织包
	PackageName  string    `xorm:"NOT NULL INDEX 'package_name'" json:"package_name"`
	CreatedAt    time.Time `xorm:"created 'created_at'" json:"created_at"`
}

// TeamMember 团队成员
type TeamMember struct {
	ID        int64     `xorm:"pk autoincr 'i_d'" json:"id"`
	TeamID    int64     `xorm:"NOT NULL INDEX 'team_i_d'" json:"team_id"`
	UserID    int64     `xorm:"NOT NULL INDEX 'user_i_d'" json:"user_id"`
	CreatedAt time.Time `xorm:"created 'created_at'" json:"created_at"`
}

func (Team) TableName() string {
	return "teams"
}

func (TeamOrganization) TableName() string {
	return "team_organizations"
}

func (TeamPackage) TableName() string {
	return "team_packages"
}

func (TeamMember) TableName() string {
	return "team_members"
}