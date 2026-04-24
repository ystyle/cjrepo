package handlers

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"xorm.io/xorm"

	"ystyle.top/go/cjrepo/internal/models"
)

type OrganizationHandler struct {
	engine *xorm.Engine
}

func NewOrganizationHandler(engine *xorm.Engine) *OrganizationHandler {
	return &OrganizationHandler{
		engine: engine,
	}
}

func (h *OrganizationHandler) ListOrganizations(c *fiber.Ctx) error {
	search := c.Query("search", "")

	query := h.engine.Where("deleted_at IS NULL")
	if search != "" {
		query = query.Where("name LIKE ? OR display_name LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	var organizations []models.Organization
	err := query.OrderBy("is_default DESC, created_at DESC").Find(&organizations)
	if err != nil {
		log.Printf("[ERROR] Failed to list organizations: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to list organizations",
		})
	}

	result := make([]fiber.Map, 0, len(organizations))
	for _, org := range organizations {
		packageCount, err := h.engine.Where("organization = ?", org.Name).Count(&models.Package{})
		if err != nil {
			packageCount = 0
		}

		result = append(result, fiber.Map{
			"id":            org.ID,
			"name":          org.Name,
			"display_name":  org.DisplayName,
			"description":   org.Description,
			"is_default":    org.IsDefault,
			"member_count":  0,
			"package_count": packageCount,
			"created_at":    org.CreatedAt,
			"updated_at":    org.UpdatedAt,
		})
	}

	return c.JSON(result)
}

type CreateOrganizationRequest struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
}

func (h *OrganizationHandler) CreateOrganization(c *fiber.Ctx) error {
	req := new(CreateOrganizationRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.Name == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "name is required",
		})
	}

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

type UpdateOrganizationRequest struct {
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	IsDefault   *bool  `json:"is_default"`
}

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

	if req.IsDefault != nil && *req.IsDefault {
		_, err := h.engine.Exec("UPDATE organizations SET is_default = FALSE WHERE is_default = TRUE")
		if err != nil {
			log.Printf("[ERROR] Failed to reset default organizations: %v", err)
		}
	}

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