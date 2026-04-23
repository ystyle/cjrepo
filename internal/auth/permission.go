package auth

import (
	"xorm.io/xorm"

	"ystyle.top/go/cjrepo/internal/models"
)

type PermissionChecker struct {
	engine *xorm.Engine
}

func NewPermissionChecker(engine *xorm.Engine) *PermissionChecker {
	return &PermissionChecker{engine: engine}
}

// PermissionLevel 权限级别数值
func PermissionLevel(perm string) int {
	switch perm {
	case "read":
		return 1
	case "write":
		return 2
	case "overwrite":
		return 3
	default:
		return 0
	}
}

// CheckPermission 检查用户对包的权限
func (p *PermissionChecker) CheckPermission(userID int64, org, pkgName, requiredPerm string) bool {
	requiredLevel := PermissionLevel(requiredPerm)

	var members []models.TeamMember
	p.engine.Where("user_i_d = ?", userID).Find(&members)

	if len(members) == 0 {
		return false
	}

	for _, member := range members {
		teamID := member.TeamID

		var teamPkg models.TeamPackage
		has, _ := p.engine.Where("team_i_d = ? AND organization = ? AND package_name = ?", teamID, org, pkgName).Get(&teamPkg)
		if has && PermissionLevel(teamPkg.Permission) >= requiredLevel {
			return true
		}

		var teamOrg models.TeamOrganization
		orgID := p.getOrgID(org)
		if orgID != nil {
			has, _ = p.engine.Where("team_i_d = ? AND organization_i_d = ?", teamID, *orgID).Get(&teamOrg)
		} else {
			has, _ = p.engine.Where("team_i_d = ? AND organization_i_d IS NULL", teamID).Get(&teamOrg)
		}

		if has {
			var team models.Team
			p.engine.ID(teamID).Get(&team)
			if PermissionLevel(team.Permission) >= requiredLevel {
				return true
			}
		}
	}

	return false
}

// getOrgID 根据组织名获取组织ID
func (p *PermissionChecker) getOrgID(orgName string) *int64 {
	if orgName == "" {
		return nil
	}
	var org models.Organization
	has, _ := p.engine.Where("name = ?", orgName).Get(&org)
	if !has {
		return nil
	}
	return &org.ID
}

// GetUserTeams 获取用户所属团队
func (p *PermissionChecker) GetUserTeams(userID int64) []models.Team {
	var members []models.TeamMember
	p.engine.Where("user_i_d = ?", userID).Find(&members)

	teamIDs := make([]int64, len(members))
	for i, m := range members {
		teamIDs[i] = m.TeamID
	}

	var teams []models.Team
	if len(teamIDs) > 0 {
		p.engine.In("i_d", teamIDs).Find(&teams)
	}
	return teams
}