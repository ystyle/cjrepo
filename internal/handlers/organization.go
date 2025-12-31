package handlers

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"xorm.io/xorm"

	"ystyle.top/go/cjrepo/internal/models"
)

// OrganizationHandler 组织管理处理器
type OrganizationHandler struct {
	engine *xorm.Engine
}

// NewOrganizationHandler 创建组织管理处理器
func NewOrganizationHandler(engine *xorm.Engine) *OrganizationHandler {
	return &OrganizationHandler{
		engine: engine,
	}
}

// ListOrganizations 获取组织列表
func (h *OrganizationHandler) ListOrganizations(c *fiber.Ctx) error {
	var organizations []models.Organization
	err := h.engine.Where("deleted_at IS NULL").OrderBy("is_default DESC, created_at DESC").Find(&organizations)
	if err != nil {
		log.Printf("[ERROR] Failed to list organizations: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to list organizations",
		})
	}

	// 为每个组织添加成员数量和包数量
	result := make([]fiber.Map, 0, len(organizations))
	for _, org := range organizations {
		// 统计成员数量
		memberCount, err := h.engine.Where("organization_i_d = ?", org.ID).Count(&models.OrganizationMember{})
		if err != nil {
			memberCount = 0
		}

		// 统计包数量
		packageCount, err := h.engine.Where("organization = ?", org.Name).Count(&models.Package{})
		if err != nil {
			packageCount = 0
		}

		result = append(result, fiber.Map{
			"id":           org.ID,
			"name":         org.Name,
			"display_name": org.DisplayName,
			"description":  org.Description,
			"is_default":   org.IsDefault,
			"member_count": memberCount,
			"package_count": packageCount,
			"created_at":   org.CreatedAt,
			"updated_at":   org.UpdatedAt,
		})
	}

	return c.JSON(result)
}

// CreateOrganizationRequest 创建组织请求
type CreateOrganizationRequest struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
}

// CreateOrganization 创建组织
func (h *OrganizationHandler) CreateOrganization(c *fiber.Ctx) error {
	req := new(CreateOrganizationRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// 验证必填字段
	if req.Name == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "name is required",
		})
	}

	// 检查名称是否已存在
	var existing models.Organization
	has, _ := h.engine.Where("name = ?", req.Name).Get(&existing)
	if has {
		return c.Status(400).JSON(fiber.Map{
			"error": "organization with this name already exists",
		})
	}

	org := &models.Organization{
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Description: req.Description,
		IsDefault:   false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, err := h.engine.Insert(org)
	if err != nil {
		log.Printf("[ERROR] Failed to create organization: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to create organization",
		})
	}

	log.Printf("[INFO] Created organization: %s", org.Name)
	return c.JSON(org)
}

// UpdateOrganizationRequest 更新组织请求
type UpdateOrganizationRequest struct {
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	IsDefault   *bool  `json:"is_default"`
}

// UpdateOrganization 更新组织
func (h *OrganizationHandler) UpdateOrganization(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "id is required",
		})
	}

	req := new(UpdateOrganizationRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	var org models.Organization
	has, err := h.engine.Where("id = ? AND deleted_at IS NULL", id).Get(&org)
	if err != nil {
		log.Printf("[ERROR] Failed to find organization: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to find organization",
		})
	}
	if !has {
		return c.Status(404).JSON(fiber.Map{
			"error": "organization not found",
		})
	}

	// 如果设置为默认组织，需要取消其他组织的默认状态
	if req.IsDefault != nil && *req.IsDefault {
		_, err := h.engine.Exec("UPDATE organizations SET is_default = FALSE WHERE is_default = TRUE")
		if err != nil {
			log.Printf("[ERROR] Failed to reset default organizations: %v", err)
		}
	}

	// 更新字段
	if req.DisplayName != "" {
		org.DisplayName = req.DisplayName
	}
	if req.Description != "" {
		org.Description = req.Description
	}
	if req.IsDefault != nil {
		org.IsDefault = *req.IsDefault
	}
	org.UpdatedAt = time.Now()

	_, err = h.engine.ID(org.ID).Update(&org)
	if err != nil {
		log.Printf("[ERROR] Failed to update organization: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to update organization",
		})
	}

	log.Printf("[INFO] Updated organization: %s", org.Name)
	return c.JSON(org)
}

// DeleteOrganization 删除组织（软删除）
func (h *OrganizationHandler) DeleteOrganization(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "id is required",
		})
	}

	var org models.Organization
	has, err := h.engine.Where("id = ? AND deleted_at IS NULL", id).Get(&org)
	if err != nil {
		log.Printf("[ERROR] Failed to find organization: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to find organization",
		})
	}
	if !has {
		return c.Status(404).JSON(fiber.Map{
			"error": "organization not found",
		})
	}

	// 软删除（xorm 会自动将 deleted 字段设置为当前时间）
	_, err = h.engine.ID(org.ID).Delete(&models.Organization{})
	if err != nil {
		log.Printf("[ERROR] Failed to delete organization: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to delete organization",
		})
	}

	log.Printf("[INFO] Deleted organization: %s", org.Name)
	return c.JSON(fiber.Map{
		"message": "organization deleted successfully",
	})
}

// GetOrganizationMembers 获取组织成员列表
func (h *OrganizationHandler) GetOrganizationMembers(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "id is required",
		})
	}

	// 检查组织是否存在
	var org models.Organization
	has, err := h.engine.Where("id = ? AND deleted_at IS NULL", id).Get(&org)
	if err != nil {
		log.Printf("[ERROR] Failed to find organization: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to find organization",
		})
	}
	if !has {
		return c.Status(404).JSON(fiber.Map{
			"error": "organization not found",
		})
	}

	// 查询组织成员
	var members []models.OrganizationMember
	err = h.engine.Where("organization_i_d = ?", id).Find(&members)
	if err != nil {
		log.Printf("[ERROR] Failed to query members: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to query members",
		})
	}

	// 获取用户详细信息
	result := make([]fiber.Map, 0, len(members))
	for _, member := range members {
		var user models.User
		has, err := h.engine.ID(member.UserID).Get(&user)
		if err != nil || !has {
			continue
		}

		result = append(result, fiber.Map{
			"user_id":    user.ID,
			"username":   user.Username,
			"email":      user.Email,
			"is_active":  user.IsActive,
			"created_at": member.CreatedAt,
		})
	}

	return c.JSON(result)
}

// AddMemberRequest 添加成员请求
type AddMemberRequest struct {
	Username string `json:"username"`
}

// AddMember 添加组织成员
func (h *OrganizationHandler) AddMember(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "id is required",
		})
	}

	req := new(AddMemberRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.Username == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "username is required",
		})
	}

	// 检查组织是否存在
	var org models.Organization
	has, err := h.engine.Where("id = ? AND deleted_at IS NULL", id).Get(&org)
	if err != nil {
		log.Printf("[ERROR] Failed to find organization: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to find organization",
		})
	}
	if !has {
		return c.Status(404).JSON(fiber.Map{
			"error": "organization not found",
		})
	}

	// 查找用户
	var user models.User
	has, err = h.engine.Where("username = ?", req.Username).Get(&user)
	if err != nil {
		log.Printf("[ERROR] Failed to find user: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to find user",
		})
	}
	if !has {
		return c.Status(404).JSON(fiber.Map{
			"error": "user not found",
		})
	}

	// 检查是否已经是成员
	var existingMember models.OrganizationMember
	has, err = h.engine.Where("organization_i_d = ? AND user_i_d = ?", org.ID, user.ID).Get(&existingMember)
	if err != nil {
		log.Printf("[ERROR] Failed to check membership: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to check membership",
		})
	}
	if has {
		return c.Status(400).JSON(fiber.Map{
			"error": "user is already a member of this organization",
		})
	}

	// 添加成员
	member := &models.OrganizationMember{
		OrganizationID: org.ID,
		UserID:         user.ID,
		CreatedAt:      time.Now(),
	}

	_, err = h.engine.Insert(member)
	if err != nil {
		log.Printf("[ERROR] Failed to add member: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to add member",
		})
	}

	log.Printf("[INFO] Added user %s to organization %s", user.Username, org.Name)
	return c.JSON(fiber.Map{
		"message": "member added successfully",
		"user": fiber.Map{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
		},
	})
}

// RemoveMember 移除组织成员
func (h *OrganizationHandler) RemoveMember(c *fiber.Ctx) error {
	id := c.Params("id")
	userID := c.Params("user_id")

	if id == "" || userID == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "id and user_id are required",
		})
	}

	// 检查组织是否存在
	var org models.Organization
	has, err := h.engine.Where("id = ? AND deleted_at IS NULL", id).Get(&org)
	if err != nil {
		log.Printf("[ERROR] Failed to find organization: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to find organization",
		})
	}
	if !has {
		return c.Status(404).JSON(fiber.Map{
			"error": "organization not found",
		})
	}

	// 删除成员关系
	_, err = h.engine.Where("organization_i_d = ? AND user_i_d = ?", id, userID).Delete(&models.OrganizationMember{})
	if err != nil {
		log.Printf("[ERROR] Failed to remove member: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to remove member",
		})
	}

	log.Printf("[INFO] Removed user %s from organization %s", userID, org.Name)
	return c.JSON(fiber.Map{
		"message": "member removed successfully",
	})
}
