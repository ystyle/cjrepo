package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"xorm.io/xorm"

	"ystyle.top/go/cjrepo/internal/models"
)

type TeamHandler struct {
	engine *xorm.Engine
}

func NewTeamHandler(engine *xorm.Engine) *TeamHandler {
	return &TeamHandler{engine: engine}
}

// ListTeamsResponse 团队列表响应
type ListTeamsResponse struct {
	Data     []map[string]interface{} `json:"data"`
	Total    int64                    `json:"total"`
	Page     int                      `json:"page"`
	PageSize int                      `json:"pageSize"`
}

// ListTeams 获取团队列表（分页）
func (h *TeamHandler) ListTeams(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	pageSize := c.QueryInt("pageSize", 20)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 统计总数
	total, _ := h.engine.Count(&models.Team{})

	// 分页查询
	var teams []models.Team
	h.engine.Limit(pageSize, (page-1)*pageSize).Find(&teams)

	result := make([]map[string]interface{}, len(teams))
	for i, team := range teams {
		memberCount, _ := h.engine.Where("team_i_d = ?", team.ID).Count(&models.TeamMember{})
		orgCount, _ := h.engine.Where("team_i_d = ?", team.ID).Count(&models.TeamOrganization{})
		pkgCount, _ := h.engine.Where("team_i_d = ?", team.ID).Count(&models.TeamPackage{})

		result[i] = map[string]interface{}{
			"id":            team.ID,
			"name":          team.Name,
			"display_name":  team.DisplayName,
			"description":   team.Description,
			"permission":    team.Permission,
			"member_count":  memberCount,
			"org_count":     orgCount,
			"package_count": pkgCount,
			"created_at":    team.CreatedAt,
		}
	}

	return c.JSON(ListTeamsResponse{
		Data:     result,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// CreateTeam 创建团队
func (h *TeamHandler) CreateTeam(c *fiber.Ctx) error {
	var req struct {
		Name        string `json:"name"`
		DisplayName string `json:"display_name"`
		Description string `json:"description"`
		Permission  string `json:"permission"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	if req.Name == "" {
		return c.Status(400).JSON(fiber.Map{"error": "name is required"})
	}

	if req.Permission != "read" && req.Permission != "write" && req.Permission != "overwrite" {
		return c.Status(400).JSON(fiber.Map{"error": "invalid permission, must be read/write/overwrite"})
	}

	exists, _ := h.engine.Where("name = ?", req.Name).Exist(&models.Team{})
	if exists {
		return c.Status(409).JSON(fiber.Map{"error": "team name already exists"})
	}

	team := &models.Team{
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Description: req.Description,
		Permission:  req.Permission,
	}

	_, err := h.engine.Insert(team)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to create team"})
	}

	return c.JSON(team)
}

// UpdateTeam 更新团队基本信息
func (h *TeamHandler) UpdateTeam(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}

	var team models.Team
	has, err := h.engine.ID(id).Get(&team)
	if err != nil || !has {
		return c.Status(404).JSON(fiber.Map{"error": "team not found"})
	}

	var req struct {
		Name        string `json:"name"`
		DisplayName string `json:"display_name"`
		Description string `json:"description"`
		Permission  string `json:"permission"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	if req.Permission != "" && req.Permission != "read" && req.Permission != "write" && req.Permission != "overwrite" {
		return c.Status(400).JSON(fiber.Map{"error": "invalid permission"})
	}

	if req.Name != "" {
		team.Name = req.Name
	}
	if req.DisplayName != "" {
		team.DisplayName = req.DisplayName
	}
	if req.Description != "" {
		team.Description = req.Description
	}
	if req.Permission != "" {
		team.Permission = req.Permission
	}

	_, err = h.engine.ID(id).Update(&team)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to update team"})
	}

	return c.JSON(team)
}

// DeleteTeam 删除团队
func (h *TeamHandler) DeleteTeam(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}

	h.engine.Where("team_i_d = ?", id).Delete(&models.TeamMember{})
	h.engine.Where("team_i_d = ?", id).Delete(&models.TeamOrganization{})
	h.engine.Where("team_i_d = ?", id).Delete(&models.TeamPackage{})
	h.engine.ID(id).Delete(&models.Team{})

	return c.JSON(fiber.Map{"message": "team deleted"})
}

// UpdateTeamOrganizations 替换团队关联组织（Replace 模式）
func (h *TeamHandler) UpdateTeamOrganizations(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}

	var req struct {
		OrganizationIDs []int64 `json:"organization_ids"` // null 或 0 表示无组织
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	h.engine.Where("team_i_d = ?", id).Delete(&models.TeamOrganization{})

	for _, orgID := range req.OrganizationIDs {
		var orgIDPtr *int64
		if orgID != 0 {
			orgIDPtr = &orgID
		}
		teamOrg := &models.TeamOrganization{
			TeamID:         id,
			OrganizationID: orgIDPtr,
		}
		h.engine.Insert(teamOrg)
	}

	return h.ListTeamOrganizations(c)
}

// ListTeamOrganizations 获取团队关联组织
func (h *TeamHandler) ListTeamOrganizations(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}

	var teamOrgs []models.TeamOrganization
	h.engine.Where("team_i_d = ?", id).Find(&teamOrgs)

	result := make([]map[string]interface{}, len(teamOrgs))
	for i, to := range teamOrgs {
		if to.OrganizationID != nil {
			var org models.Organization
			h.engine.ID(to.OrganizationID).Get(&org)
			result[i] = map[string]interface{}{
				"id":              to.ID,
				"organization_id": to.OrganizationID,
				"organization_name": org.Name,
				"is_null_org":     false,
			}
		} else {
			result[i] = map[string]interface{}{
				"id":              to.ID,
				"organization_id": nil,
				"organization_name": "",
				"is_null_org":     true,
			}
		}
	}

	return c.JSON(result)
}

// UpdateTeamPackages 替换团队关联的包（Replace 模式）
func (h *TeamHandler) UpdateTeamPackages(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}

	var req struct {
		Packages []struct {
			Organization string `json:"organization"`
			PackageName  string `json:"package_name"`
		} `json:"packages"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	h.engine.Where("team_i_d = ?", id).Delete(&models.TeamPackage{})

	for _, pkg := range req.Packages {
		teamPkg := &models.TeamPackage{
			TeamID:       id,
			Organization: pkg.Organization,
			PackageName:  pkg.PackageName,
		}
		h.engine.Insert(teamPkg)
	}

	return h.ListTeamPackages(c)
}
// ListTeamPackages 获取团队包权限列表
func (h *TeamHandler) ListTeamPackages(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}

	var teamPkgs []models.TeamPackage
	h.engine.Where("team_i_d = ?", id).Find(&teamPkgs)

	return c.JSON(teamPkgs)
}

// UpdateTeamMembers 替换团队成员（Replace 模式）
func (h *TeamHandler) UpdateTeamMembers(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}

	var req struct {
		UserIDs []int64 `json:"user_ids"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	h.engine.Where("team_i_d = ?", id).Delete(&models.TeamMember{})

	for _, userID := range req.UserIDs {
		member := &models.TeamMember{
			TeamID: id,
			UserID: userID,
		}
		h.engine.Insert(member)
	}

	return h.ListTeamMembers(c)
}

// ListTeamMembers 获取团队成员列表
func (h *TeamHandler) ListTeamMembers(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}

	var members []models.TeamMember
	h.engine.Where("team_i_d = ?", id).Find(&members)

	result := make([]map[string]interface{}, len(members))
	for i, m := range members {
		var user models.User
		h.engine.ID(m.UserID).Get(&user)
		result[i] = map[string]interface{}{
			"id":        m.ID,
			"user_id":   m.UserID,
			"username":  user.Username,
			"email":     user.Email,
			"is_active": user.IsActive,
		}
	}

	return c.JSON(result)
}