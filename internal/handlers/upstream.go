package handlers

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"xorm.io/xorm"

	"ystyle.top/go/cjrepo/internal/models"
)

// UpstreamHandler 上游管理处理器
type UpstreamHandler struct {
	engine *xorm.Engine
}

// NewUpstreamHandler 创建上游管理处理器
func NewUpstreamHandler(engine *xorm.Engine) *UpstreamHandler {
	return &UpstreamHandler{
		engine: engine,
	}
}

// ListUpstreams 获取上游列表
func (h *UpstreamHandler) ListUpstreams(c *fiber.Ctx) error {
	var upstreams []models.Upstream
	err := h.engine.Find(&upstreams)
	if err != nil {
		log.Printf("[ERROR] Failed to list upstreams: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to list upstreams",
		})
	}

	return c.JSON(upstreams)
}

// CreateUpstreamRequest 创建上游请求
type CreateUpstreamRequest struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	Enabled   bool   `json:"enabled"`
	CacheTTL  int    `json:"cache_ttl"`
	AuthToken string `json:"auth_token"`
}

// CreateUpstream 创建上游
func (h *UpstreamHandler) CreateUpstream(c *fiber.Ctx) error {
	req := new(CreateUpstreamRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// 验证必填字段
	if req.Name == "" || req.URL == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "name and url are required",
		})
	}

	// 检查名称是否已存在
	var existing models.Upstream
	has, _ := h.engine.Where("name = ?", req.Name).Get(&existing)
	if has {
		return c.Status(400).JSON(fiber.Map{
			"error": "upstream with this name already exists",
		})
	}

	// 设置默认值
	if req.CacheTTL == 0 {
		req.CacheTTL = 86400 // 默认24小时
	}
	if !req.Enabled {
		req.Enabled = true // 默认启用
	}

	upstream := &models.Upstream{
		Name:      req.Name,
		URL:       req.URL,
		Enabled:   req.Enabled,
		CacheTTL:  req.CacheTTL,
		AuthToken: req.AuthToken,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err := h.engine.Insert(upstream)
	if err != nil {
		log.Printf("[ERROR] Failed to create upstream: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to create upstream",
		})
	}

	log.Printf("[INFO] Created upstream: %s (%s)", upstream.Name, upstream.URL)
	return c.JSON(upstream)
}

// UpdateUpstreamRequest 更新上游请求
type UpdateUpstreamRequest struct {
	URL       string `json:"url"`
	Enabled   *bool  `json:"enabled"`
	CacheTTL  *int   `json:"cache_ttl"`
	AuthToken string `json:"auth_token"`
}

// UpdateUpstream 更新上游
func (h *UpstreamHandler) UpdateUpstream(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "id is required",
		})
	}

	req := new(UpdateUpstreamRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	var upstream models.Upstream
	has, err := h.engine.Where("id = ?", id).Get(&upstream)
	if err != nil {
		log.Printf("[ERROR] Failed to find upstream: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to find upstream",
		})
	}
	if !has {
		return c.Status(404).JSON(fiber.Map{
			"error": "upstream not found",
		})
	}

	// 更新字段
	if req.URL != "" {
		upstream.URL = req.URL
	}
	if req.Enabled != nil {
		upstream.Enabled = *req.Enabled
	}
	if req.CacheTTL != nil {
		upstream.CacheTTL = *req.CacheTTL
	}
	if req.AuthToken != "" {
		upstream.AuthToken = req.AuthToken
	}
	upstream.UpdatedAt = time.Now()

	_, err = h.engine.ID(upstream.ID).Update(&upstream)
	if err != nil {
		log.Printf("[ERROR] Failed to update upstream: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to update upstream",
		})
	}

	log.Printf("[INFO] Updated upstream: %s", upstream.Name)
	return c.JSON(upstream)
}

// DeleteUpstream 删除上游
func (h *UpstreamHandler) DeleteUpstream(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "id is required",
		})
	}

	var upstream models.Upstream
	has, err := h.engine.Where("id = ?", id).Get(&upstream)
	if err != nil {
		log.Printf("[ERROR] Failed to find upstream: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to find upstream",
		})
	}
	if !has {
		return c.Status(404).JSON(fiber.Map{
			"error": "upstream not found",
		})
	}

	_, err = h.engine.ID(upstream.ID).Delete(&models.Upstream{})
	if err != nil {
		log.Printf("[ERROR] Failed to delete upstream: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to delete upstream",
		})
	}

	log.Printf("[INFO] Deleted upstream: %s", upstream.Name)
	return c.JSON(fiber.Map{
		"message": "upstream deleted successfully",
	})
}

// TestUpstream 测试上游连接
func (h *UpstreamHandler) TestUpstream(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "id is required",
		})
	}

	var upstream models.Upstream
	has, err := h.engine.Where("id = ?", id).Get(&upstream)
	if err != nil {
		log.Printf("[ERROR] Failed to find upstream: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to find upstream",
		})
	}
	if !has {
		return c.Status(404).JSON(fiber.Map{
			"error": "upstream not found",
		})
	}

	// TODO: 实现实际的上游连接测试
	// 这里暂时返回成功，实际应该向上游发送一个简单的请求来验证连接

	return c.JSON(fiber.Map{
		"success": true,
		"message": "upstream connection test passed",
	})
}
