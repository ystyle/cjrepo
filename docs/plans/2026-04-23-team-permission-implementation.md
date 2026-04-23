# 团队权限功能 实现计划

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 为 cjrepo 添加团队权限功能，实现精细化的包管理权限控制

**Architecture:** 新增 Team/TeamOrganization/TeamPackage/TeamMember 四张表，通过团队管理用户对组织/包的访问权限。权限递进：read → write → overwrite

**Tech Stack:** Go + XORM + Fiber + Vue 3 + Element Plus

---

## Task 1: 创建数据模型

**Files:**
- Create: `internal/models/team.go`

**Step 1: 创建 Team 数据模型文件**

```go
package models

import "time"

// Team 团队
type Team struct {
	ID          int64     `xorm:"pk autoincr 'i_d'" json:"id"`
	Name        string    `xorm:"VARCHAR(100) NOT NULL UNIQUE 'name'" json:"name"`
	DisplayName string    `xorm:"VARCHAR(255) 'display_name'" json:"display_name"`
	Description string    `xorm:"TEXT 'description'" json:"description"`
	Permission  string    `xorm:"VARCHAR(20) NOT NULL 'permission'" json:"permission"` // read/write/overwrite
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

// TeamPackage 团队对特定包的权限
type TeamPackage struct {
	ID           int64     `xorm:"pk autoincr 'i_d'" json:"id"`
	TeamID       int64     `xorm:"NOT NULL INDEX 'team_i_d'" json:"team_id"`
	Organization string    `xorm:"INDEX 'organization'" json:"organization"` // 空字符串表示无组织包
	PackageName  string    `xorm:"NOT NULL INDEX 'package_name'" json:"package_name"`
	Permission   string    `xorm:"VARCHAR(20) NOT NULL 'permission'" json:"permission"` // read/write/overwrite
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
```

**Step 2: 在 main.go 注册新表**

修改 `main.go` 的 `initDatabase` 函数，在 Sync2 中添加新表：

```go
if err := engine.Sync2(
    new(models.Package),
    new(models.User),
    new(models.PublishLog),
    new(models.AdminLog),
    new(models.Upstream),
    new(models.Organization),
    new(models.OrganizationMember),
    new(models.Team),            // 新增
    new(models.TeamOrganization), // 新增
    new(models.TeamPackage),      // 新增
    new(models.TeamMember),       // 新增
); err != nil {
    return nil, fmt.Errorf("failed to sync database: %w", err)
}
```

**Step 3: 验证编译**

```bash
go build -o cjrepo .
```

**Step 4: Commit**

```bash
git add internal/models/team.go main.go
git commit -m "feat: 添加团队权限数据模型"
```

---

## Task 2: 创建团队 Handler

**Files:**
- Create: `internal/handlers/team.go`

**Step 1: 创建 TeamHandler**

创建文件 `internal/handlers/team.go`：

```go
package handlers

import (
	"log"
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

// ListTeams 获取团队列表
func (h *TeamHandler) ListTeams(c *fiber.Ctx) error {
	var teams []models.Team
	err := h.engine.Find(&teams)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "database error"})
	}

	// 为每个团队统计成员数和组织数
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

	return c.JSON(result)
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

	// 检查名称是否已存在
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
	if !has {
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

	// 删除关联记录
	h.engine.Where("team_i_d = ?", id).Delete(&models.TeamMember{})
	h.engine.Where("team_i_d = ?", id).Delete(&models.TeamOrganization{})
	h.engine.Where("team_i_d = ?", id).Delete(&models.TeamPackage{})

	// 删除团队
	_, err = h.engine.ID(id).Delete(&models.Team{})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to delete team"})
	}

	return c.JSON(fiber.Map{"message": "team deleted"})
}
```

**Step 2: 验证编译**

```bash
go build -o cjrepo .
```

**Step 3: Commit**

```bash
git add internal/handlers/team.go
git commit -m "feat: 添加团队 Handler 基础 CRUD"
```

---

## Task 3: 创建团队关联 Handler

**Files:**
- Modify: `internal/handlers/team.go`

**Step 1: 添加团队-组织关联 API**

在 `team.go` 中添加：

```go
// UpdateTeamOrganizations 替换团队关联组织（Replace 模式）
func (h *TeamHandler) UpdateTeamOrganizations(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}

	var req struct {
		OrganizationIDs []int64 `json:"organization_ids"` // null 表示无组织
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	// 删除旧关联
	h.engine.Where("team_i_d = ?", id).Delete(&models.TeamOrganization{})

	// 插入新关联
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

	// 返回更新后的列表
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
		var org models.Organization
		if to.OrganizationID != nil {
			h.engine.ID(to.OrganizationID).Get(&org)
			result[i] = map[string]interface{}{
				"id":             to.ID,
				"organization_id": to.OrganizationID,
				"organization_name": org.Name,
				"is_null_org":    false,
			}
		} else {
			result[i] = map[string]interface{}{
				"id":             to.ID,
				"organization_id": nil,
				"organization_name": "",
				"is_null_org":    true, // 无组织包
			}
		}
	}

	return c.JSON(result)
}
```

**Step 2: 添加团队-包关联 API**

```go
// UpdateTeamPackages 替换团队包权限（Replace 模式）
func (h *TeamHandler) UpdateTeamPackages(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}

	var req struct {
		Packages []struct {
			Organization string `json:"organization"`
			PackageName  string `json:"package_name"`
			Permission   string `json:"permission"`
		} `json:"packages"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	// 验证权限值
	for _, pkg := range req.Packages {
		if pkg.Permission != "read" && pkg.Permission != "write" && pkg.Permission != "overwrite" {
			return c.Status(400).JSON(fiber.Map{"error": "invalid permission"})
		}
	}

	// 删除旧关联
	h.engine.Where("team_i_d = ?", id).Delete(&models.TeamPackage{})

	// 插入新关联
	for _, pkg := range req.Packages {
		teamPkg := &models.TeamPackage{
			TeamID:       id,
			Organization: pkg.Organization,
			PackageName:  pkg.PackageName,
			Permission:   pkg.Permission,
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
```

**Step 3: 添加团队成员 API**

```go
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

	// 删除旧成员
	h.engine.Where("team_i_d = ?", id).Delete(&models.TeamMember{})

	// 插入新成员
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

	// 获取用户详情
	result := make([]map[string]interface{}, len(members))
	for i, m := range members {
		var user models.User
		h.engine.ID(m.UserID).Get(&user)
		result[i] = map[string]interface{}{
			"id":       m.ID,
			"user_id":  m.UserID,
			"username": user.Username,
			"email":    user.Email,
			"is_active": user.IsActive,
		}
	}

	return c.JSON(result)
}
```

**Step 4: 验证编译**

```bash
go build -o cjrepo .
```

**Step 5: Commit**

```bash
git add internal/handlers/team.go
git commit -m "feat: 添加团队关联 API（组织/包/成员）"
```

---

## Task 4: 注册团队路由

**Files:**
- Modify: `main.go`

**Step 1: 在 setupRoutes 中注册团队路由**

在 `admin` 路由组中添加团队路由：

```go
// Team management
teamHandler := handlers.NewTeamHandler(engine)
admin.Get("/teams", teamHandler.ListTeams)
admin.Post("/teams", teamHandler.CreateTeam)
admin.Put("/teams/:id", teamHandler.UpdateTeam)
admin.Delete("/teams/:id", teamHandler.DeleteTeam)
admin.Put("/teams/:id/organizations", teamHandler.UpdateTeamOrganizations)
admin.Get("/teams/:id/organizations", teamHandler.ListTeamOrganizations)
admin.Put("/teams/:id/packages", teamHandler.UpdateTeamPackages)
admin.Get("/teams/:id/packages", teamHandler.ListTeamPackages)
admin.Put("/teams/:id/members", teamHandler.UpdateTeamMembers)
admin.Get("/teams/:id/members", teamHandler.ListTeamMembers)
```

**Step 2: 验证编译**

```bash
go build -o cjrepo .
```

**Step 3: Commit**

```bash
git add main.go
git commit -m "feat: 注册团队管理 API 路由"
```

---

## Task 5: 创建权限检查模块

**Files:**
- Create: `internal/auth/permission.go`

**Step 1: 创建权限检查模块**

```go
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

// PermissionLevel 权限级别
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

	// 1. 查用户所属团队
	var members []models.TeamMember
	p.engine.Where("user_i_d = ?", userID).Find(&members)

	if len(members) == 0 {
		return false
	}

	// 2. 遍历团队检查权限
	for _, member := range members {
		teamID := member.TeamID

		// 先查 TeamPackage（包级权限）
		var teamPkg models.TeamPackage
		has, _ := p.engine.Where("team_i_d = ? AND organization = ? AND package_name = ?", teamID, org, pkgName).Get(&teamPkg)
		if has && PermissionLevel(teamPkg.Permission) >= requiredLevel {
			return true
		}

		// 无包级权限，查 TeamOrganization（组织级权限）
		var teamOrg models.TeamOrganization
		// 查询条件：organization_i_d = org的ID 或 organization_i_d IS NULL（无组织包）
		orgID := p.getOrgID(org)
		if orgID != nil {
			has, _ = p.engine.Where("team_i_d = ? AND organization_i_d = ?", teamID, orgID).Get(&teamOrg)
		} else {
			has, _ = p.engine.Where("team_i_d = ? AND organization_i_d IS NULL", teamID).Get(&teamOrg)
		}

		if has {
			// 获取团队的默认权限
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
	p.engine.In("i_d", teamIDs).Find(&teams)
	return teams
}
```

**Step 2: 验证编译**

```bash
go build -o cjrepo .
```

**Step 3: Commit**

```bash
git add internal/auth/permission.go
git commit -m "feat: 创建权限检查模块"
```

---

## Task 6: 添加权限中间件

**Files:**
- Modify: `internal/middleware/auth.go`

**Step 1: 添加权限检查中间件**

```go
// RequirePermission 权限检查中间件
func RequirePermission(engine *xorm.Engine, requiredPerm string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 获取用户ID（已通过 JWT 验证）
		userID := c.Locals("userID").(int64)

		// 查询用户是否为超级管理员
		var user models.User
		has, _ := engine.ID(userID).Get(&user)
		if has && user.IsSuperuser {
			return c.Next()
		}

		// 获取目标组织和包名
		org := c.Query("organization", "")
		pkgName := c.Params("name")

		// 检查权限
		checker := auth.NewPermissionChecker(engine)
		if !checker.CheckPermission(userID, org, pkgName, requiredPerm) {
			return c.Status(403).JSON(fiber.Map{
				"error": "permission denied",
			})
		}

		return c.Next()
	}
}
```

**Step 2: Commit**

```bash
git add internal/middleware/auth.go
git commit -m "feat: 添加权限检查中间件"
```

---

## Task 7: 应用权限到 Publish Handler

**Files:**
- Modify: `internal/handlers/publish.go`

**Step 1: 在 HandlePublish 中添加权限检查**

在发布逻辑开始处添加：

```go
func (h *PublishHandler) HandlePublish(c *fiber.Ctx) error {
	// ... 解析请求参数 ...

	// 获取用户ID
	userID := c.Locals("userID").(int64)

	// 查询用户是否为超级管理员
	var user models.User
	has, _ := h.engine.ID(userID).Get(&user)
	isSuperuser := has && user.IsSuperuser

	// 检查版本是否已存在
	versionExists := h.checkVersionExists(organization, name, version)

	if !isSuperuser {
		checker := auth.NewPermissionChecker(h.engine)
		var requiredPerm string
		if versionExists {
			requiredPerm = "overwrite"
		} else {
			requiredPerm = "write"
		}

		if !checker.CheckPermission(userID, organization, name, requiredPerm) {
			if versionExists {
				return c.Status(403).JSON(fiber.Map{
					"error": "version already exists, need overwrite permission",
				})
			}
			return c.Status(403).JSON(fiber.Map{
				"error": "permission denied",
			})
		}
	}

	// ... 继续发布逻辑 ...
}

func (h *PublishHandler) checkVersionExists(org, name, version string) bool {
	var pkg models.Package
	if org == "" {
		has, _ := h.engine.Where("(organization = '' OR organization IS NULL) AND name = ? AND version = ? AND deleted_at IS NULL", name, version).Get(&pkg)
		return has
	}
	has, _ := h.engine.Where("organization = ? AND name = ? AND version = ? AND deleted_at IS NULL", org, name, version).Get(&pkg)
	return has
}
```

**Step 2: 验证编译**

```bash
go build -o cjrepo .
```

**Step 3: Commit**

```bash
git add internal/handlers/publish.go
git commit -m "feat: 发布时添加权限检查"
```

---

## Task 8: 应用权限到 Download 和 Index Handler

**Files:**
- Modify: `internal/handlers/download.go`
- Modify: `internal/handlers/index.go`

**Step 1: 在 HandleDownload 中添加权限检查**

```go
func (h *DownloadHandler) HandleDownload(c *fiber.Ctx) error {
	// ... 解析参数 ...

	// 权限检查（需要 read 权限）
	userID := c.Locals("userID").(int64)
	var user models.User
	has, _ := h.engine.ID(userID).Get(&user)

	if !has || !user.IsSuperuser {
		checker := auth.NewPermissionChecker(h.engine)
		if !checker.CheckPermission(userID, organization, name, "read") {
			return c.Status(403).JSON(fiber.Map{"error": "permission denied"})
		}
	}

	// ... 继续下载逻辑 ...
}
```

**Step 2: 在 HandleIndex 中添加权限检查**

```go
func (h *IndexHandler) HandleIndex(c *fiber.Ctx) error {
	// ... 解析参数 ...

	// 权限检查（需要 read 权限）
	userID := c.Locals("userID").(int64)
	var user models.User
	has, _ := h.engine.ID(userID).Get(&user)

	if !has || !user.IsSuperuser {
		checker := auth.NewPermissionChecker(h.engine)
		if !checker.CheckPermission(userID, organization, fullName, "read") {
			return c.Status(403).JSON(fiber.Map{"error": "permission denied"})
		}
	}

	// ... 继续索引逻辑 ...
}
```

**Step 3: 验证编译**

```bash
go build -o cjrepo .
```

**Step 4: Commit**

```bash
git add internal/handlers/download.go internal/handlers/index.go
git commit -m "feat: 下载和索引添加权限检查"
```

---

## Task 9: 前端 API 类型定义

**Files:**
- Create: `frontend/src/api/team.ts`

**Step 1: 创建团队 API 类型**

```typescript
import request from './index'

export interface Team {
  id: number
  name: string
  display_name: string
  description: string
  permission: string // read/write/overwrite
  member_count: number
  org_count: number
  package_count: number
  created_at: string
}

export interface TeamOrganization {
  id: number
  organization_id: number | null
  organization_name: string
  is_null_org: boolean // 无组织包
}

export interface TeamPackage {
  id: number
  team_id: number
  organization: string
  package_name: string
  permission: string
}

export interface TeamMember {
  id: number
  user_id: number
  username: string
  email: string
  is_active: boolean
}

// 团队 CRUD
export const getTeams = () => {
  return request<Team[]>({
    url: '/admin/teams',
    method: 'get',
  })
}

export const createTeam = (data: {
  name: string
  display_name?: string
  description?: string
  permission: string
}) => {
  return request<Team>({
    url: '/admin/teams',
    method: 'post',
    data,
  })
}

export const updateTeam = (id: number, data: {
  name?: string
  display_name?: string
  description?: string
  permission?: string
}) => {
  return request<Team>({
    url: `/admin/teams/${id}`,
    method: 'put',
    data,
  })
}

export const deleteTeam = (id: number) => {
  return request({
    url: `/admin/teams/${id}`,
    method: 'delete',
  })
}

// 团队-组织关联（Replace 模式）
export const getTeamOrganizations = (id: number) => {
  return request<TeamOrganization[]>({
    url: `/admin/teams/${id}/organizations`,
    method: 'get',
  })
}

export const updateTeamOrganizations = (id: number, organization_ids: (number | null)[]) => {
  return request<TeamOrganization[]>({
    url: `/admin/teams/${id}/organizations`,
    method: 'put',
    data: { organization_ids },
  })
}

// 团队-包关联（Replace 模式）
export const getTeamPackages = (id: number) => {
  return request<TeamPackage[]>({
    url: `/admin/teams/${id}/packages`,
    method: 'get',
  })
}

export const updateTeamPackages = (id: number, packages: {
  organization: string
  package_name: string
  permission: string
}[]) => {
  return request<TeamPackage[]>({
    url: `/admin/teams/${id}/packages`,
    method: 'put',
    data: { packages },
  })
}

// 团队成员（Replace 模式）
export const getTeamMembers = (id: number) => {
  return request<TeamMember[]>({
    url: `/admin/teams/${id}/members`,
    method: 'get',
  })
}

export const updateTeamMembers = (id: number, user_ids: number[]) => {
  return request<TeamMember[]>({
    url: `/admin/teams/${id}/members`,
    method: 'put',
    data: { user_ids },
  })
}
```

**Step 2: Commit**

```bash
git add frontend/src/api/team.ts
git commit -m "feat: 前端团队 API 类型定义"
```

---

## Task 10: 创建 Teams.vue 页面

**Files:**
- Create: `frontend/src/views/admin/Teams.vue`

**Step 1: 创建团队列表页面**

（完整 Vue 组件代码，包含列表、操作按钮、调用弹窗）

**Step 2: Commit**

```bash
git add frontend/src/views/admin/Teams.vue
git commit -m "feat: 创建团队列表页面"
```

---

## Task 11: 创建团队弹窗组件

**Files:**
- Create: `frontend/src/components/admin/TeamFormDialog.vue`
- Create: `frontend/src/components/admin/TeamOrganizationsDialog.vue`
- Create: `frontend/src/components/admin/TeamPackagesDialog.vue`
- Create: `frontend/src/components/admin/TeamMembersDialog.vue`

**Step 1-4: 分别创建四个弹窗组件**

**Step 5: Commit**

```bash
git add frontend/src/components/admin/
git commit -m "feat: 创建团队管理弹窗组件"
```

---

## Task 12: 注册前端路由

**Files:**
- Modify: `frontend/src/router/index.ts`
- Modify: `frontend/src/layouts/AdminLayout.vue`

**Step 1: 添加路由**

```typescript
{
  path: '/admin/teams',
  name: 'Teams',
  component: () => import('../views/admin/Teams.vue'),
  meta: { title: '团队管理' },
}
```

**Step 2: 添加侧边栏菜单**

```vue
<el-menu-item index="/admin/teams">
  <el-icon><UserFilled /></el-icon>
  <span>团队管理</span>
</el-menu-item>
```

**Step 3: Commit**

```bash
git add frontend/src/router/index.ts frontend/src/layouts/AdminLayout.vue
git commit -m "feat: 注册团队管理前端路由和菜单"
```

---

## Task 13: 删除 OrganizationMember 相关代码

**Files:**
- Modify: `internal/models/organization.go`
- Modify: `internal/handlers/organization.go`
- Modify: `main.go`

**Step 1: 删除 OrganizationMember 模型**

从 `organization.go` 中删除 `OrganizationMember` 结构体和 TableName 方法

**Step 2: 删除 OrganizationMember 路由和处理逻辑**

从 `organization.go` 和 `main.go` 中移除相关代码

**Step 3: Commit**

```bash
git add internal/models/organization.go internal/handlers/organization.go main.go
git commit -m "refactor: 移除 OrganizationMember，改用 TeamMember"
```

---

## Task 14: 构建测试

**Step 1: 构建前端**

```bash
cd frontend && npm run build-only && cd ..
```

**Step 2: 构建后端**

```bash
go build -o cjrepo .
```

**Step 3: Commit**

```bash
git add -A
git commit -m "feat: 团队权限功能完成"
```

---

## Task 15: Docker 构建测试

**Step 1: 构建 Docker 镜像**

```bash
docker build -t cjrepo:test .
```

**Step 2: 运行容器测试**

```bash
docker run --rm -d -e CJREPO_ADMIN_KEY=test-key -p 18060:8060 --name cjrepo-test cjrepo:test
```

**Step 3: 测试 API**

```bash
curl http://localhost:18060/api/admin/teams
```

**Step 4: Commit**

```bash
git tag -a v1.1.0 -m "v1.1.0 - 团队权限功能"
git push origin v1.1.0
```