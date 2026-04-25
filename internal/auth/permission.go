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
// 双路径：1. 个人发布（publisher）2. 团队管理（team）
func (p *PermissionChecker) CheckPermission(userID int64, org, pkgName, requiredPerm string) bool {
	requiredLevel := PermissionLevel(requiredPerm)

	// 路径 1：个人发布——无组织包时检查 publisher_id
	if org == "" {
		var pkg models.Package
		has, _ := p.engine.Where("name = ? AND organization = ? AND publisher_i_d = ?",
			pkgName, "", userID).Get(&pkg)
		if has && PermissionLevel("write") >= requiredLevel {
			return true
		}
	}

	// 路径 2：团队管理
	var members []models.TeamMember
	p.engine.Where("user_i_d = ?", userID).Find(&members)

	for _, member := range members {
		teamID := member.TeamID

		// 检查团队是否关联了该包（team_packages 存在性匹配，权限走 team.permission）
		has, _ := p.engine.Where("team_i_d = ? AND organization = ? AND package_name = ?",
			teamID, org, pkgName).Exist(&models.TeamPackage{})
		if has {
			var team models.Team
			p.engine.ID(teamID).Get(&team)
			if PermissionLevel(team.Permission) >= requiredLevel {
				return true
			}
		}

		// 检查团队是否关联了该组织
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
