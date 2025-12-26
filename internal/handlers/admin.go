package handlers

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"xorm.io/xorm"

	"ystyle.top/go/cjrepo/internal/auth"
	"ystyle.top/go/cjrepo/internal/models"
)

// AdminHandler 管理后台处理器
type AdminHandler struct {
	engine       *xorm.Engine
	authService  *auth.AuthService
}

// NewAdminHandler 创建管理后台处理器
func NewAdminHandler(engine *xorm.Engine, authService *auth.AuthService) *AdminHandler {
	return &AdminHandler{
		engine:      engine,
		authService: authService,
	}
}

// LoginRequest 登录请求
type LoginRequest struct {
	AdminKey string `json:"adminKey"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expiresAt"`
	Message   string `json:"message"`
}

// Login 管理员登录（使用管理密钥换取 JWT token）
func (h *AdminHandler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// 验证管理密钥（已经是 MD5 加密后的）
	if !h.authService.VerifyAdminKey(req.AdminKey) {
		return c.Status(401).JSON(fiber.Map{
			"error": "Invalid admin key",
		})
	}

	// 生成 JWT token（30分钟有效）
	token, err := h.authService.GenerateToken()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to generate token",
		})
	}

	// 记录登录日志
	h.logAdminAction(c, "admin_login", "-", nil)

	return c.JSON(LoginResponse{
		Token:     token,
		ExpiresAt: time.Now().Add(30 * time.Minute).Unix(),
		Message:   "Login successful",
	})
}

// DashboardStats Dashboard 统计数据
type DashboardStats struct {
	Packages       int64 `json:"packages"`
	Versions       int64 `json:"versions"`
	Users          int64 `json:"users"`
	ActiveUsers    int64 `json:"activeUsers"`
	StorageSize    int64 `json:"storageSize"`
	PublishSuccess int64 `json:"publishSuccess"`
	PublishFailed  int64 `json:"publishFailed"`
}

// GetDashboardStats 获取 Dashboard 统计数据
func (h *AdminHandler) GetDashboardStats(c *fiber.Ctx) error {
	var stats DashboardStats

	// 包统计
	stats.Packages, _ = h.engine.Table("packages").Where("deleted_at IS NULL").Count()
	stats.Versions, _ = h.engine.Table("packages").Count()

	// 用户统计
	stats.Users, _ = h.engine.Table("users").Count()
	stats.ActiveUsers, _ = h.engine.Table("users").Where("is_active = ?", true).Count()

	// 存储大小
	storageSizeFloat, _ := h.engine.Table("packages").Where("deleted_at IS NULL").Sum(&models.Package{}, "tarball_size")
	stats.StorageSize = int64(storageSizeFloat)

	// 发布日志统计
	stats.PublishSuccess, _ = h.engine.Table("publish_logs").Where("status = ?", "success").Count()
	stats.PublishFailed, _ = h.engine.Table("publish_logs").Where("status = ?", "failed").Count()

	return c.JSON(stats)
}

// ListPackagesResponse 包列表响应
type ListPackagesResponse struct {
	Data     []models.Package `json:"data"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"pageSize"`
}

// ListPackages 获取包列表（包含已删除）
func (h *AdminHandler) ListPackages(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	pageSize := c.QueryInt("pageSize", 20)
	search := c.Query("search", "")
	org := c.Query("org", "")
	artifactType := c.Query("artifactType", "")
	showDeleted := c.QueryBool("deleted", false)

	// 构建基础查询
	baseQuery := h.engine.Table("packages")
	if !showDeleted {
		baseQuery = baseQuery.Where("deleted_at IS NULL")
	}
	if search != "" {
		baseQuery = baseQuery.Where("name LIKE ? OR description LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if org != "" {
		baseQuery = baseQuery.Where("organization = ?", org)
	}
	if artifactType == "executable" {
		baseQuery = baseQuery.Where("executable = ?", true)
	} else if artifactType == "src" || artifactType == "bin" {
		baseQuery = baseQuery.Where("artifact_type = ?", artifactType)
	}

	// 统计总数
	total, _ := baseQuery.Count()

	// 查询数据（重新构建查询）
	query := h.engine.Table("packages")
	if !showDeleted {
		query = query.Where("deleted_at IS NULL")
	}
	if search != "" {
		query = query.Where("name LIKE ? OR description LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if org != "" {
		query = query.Where("organization = ?", org)
	}
	if artifactType == "executable" {
		query = query.Where("executable = ?", true)
	} else if artifactType == "src" || artifactType == "bin" {
		query = query.Where("artifact_type = ?", artifactType)
	}

	var packages []models.Package
	query.OrderBy("created_at DESC").Limit(pageSize, (page-1)*pageSize).Find(&packages)

	// 确保返回空数组而不是 null
	if packages == nil {
		packages = []models.Package{}
	}

	return c.JSON(ListPackagesResponse{
		Data:     packages,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// DeletePackage 软删除包
func (h *AdminHandler) DeletePackage(c *fiber.Ctx) error {
	id := c.Params("id")

	// 软删除
	_, err := h.engine.ID(id).Cols("deleted_at").Update(&models.Package{
		DeletedAt: time.Now(),
	})

	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to delete package",
		})
	}

	// 记录操作日志
	h.logAdminAction(c, "delete_package", id, nil)

	return c.JSON(fiber.Map{
		"message": "Package deleted successfully",
	})
}

// GetPackageVersions 获取某个包的所有版本
func (h *AdminHandler) GetPackageVersions(c *fiber.Ctx) error {
	name := c.Params("name")
	org := c.Query("org", "")

	// 构建查询
	query := h.engine.Where("name = ?", name)
	if org != "" {
		query = query.Where("organization = ?", org)
	}

	// 查询所有版本
	var packages []models.Package
	err := query.OrderBy("created_at DESC").Find(&packages)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to fetch package versions",
		})
	}

	return c.JSON(packages)
}

// RestorePackage 恢复已删除的包
func (h *AdminHandler) RestorePackage(c *fiber.Ctx) error {
	id := c.Params("id")

	// 恢复
	_, err := h.engine.ID(id).Cols("deleted_at").Update(&models.Package{
		DeletedAt: time.Time{}, // 零值表示未删除
	})

	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to restore package",
		})
	}

	// 记录操作日志
	h.logAdminAction(c, "restore_package", id, nil)

	return c.JSON(fiber.Map{
		"message": "Package restored successfully",
	})
}

// HardDeletePackage 硬删除包（不可恢复）
func (h *AdminHandler) HardDeletePackage(c *fiber.Ctx) error {
	id := c.Params("id")

	// 先查询包信息
	var pkg models.Package
	has, err := h.engine.ID(id).Get(&pkg)
	if err != nil || !has {
		return c.Status(404).JSON(fiber.Map{
			"error": "Package not found",
		})
	}

	// 硬删除
	_, err = h.engine.ID(id).Delete(&models.Package{})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to hard delete package",
		})
	}

	// 记录操作日志
	details := map[string]interface{}{
		"name":      pkg.Name,
		"version":   pkg.Version,
		"org":       pkg.Organization,
		"tarball":   pkg.TarballPath,
	}
	h.logAdminAction(c, "hard_delete_package", id, details)

	return c.JSON(fiber.Map{
		"message": "Package permanently deleted",
	})
}

// ListUsers 获取用户列表
func (h *AdminHandler) ListUsers(c *fiber.Ctx) error {
	var users []models.User
	err := h.engine.OrderBy("created_at DESC").Find(&users)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to fetch users",
		})
	}

	return c.JSON(users)
}

// CreateUserRequest 创建用户请求
type CreateUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

// CreateUser 创建用户
func (h *AdminHandler) CreateUser(c *fiber.Ctx) error {
	var req CreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// 检查用户名是否已存在
	exists, _ := h.engine.Where("username = ?", req.Username).Exist(&models.User{})
	if exists {
		return c.Status(409).JSON(fiber.Map{
			"error": "Username already exists",
		})
	}

	// 生成 token
	token := fmt.Sprintf("token-%s-%d", req.Username, time.Now().Unix())

	// 创建用户
	user := &models.User{
		Username: req.Username,
		Email:    req.Email,
		Token:    token,
		IsActive: true,
	}

	_, err := h.engine.Insert(user)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to create user",
		})
	}

	// 记录操作日志
	h.logAdminAction(c, "create_user", fmt.Sprintf("%d", user.ID), map[string]interface{}{
		"username": user.Username,
		"email":    user.Email,
	})

	return c.Status(201).JSON(user)
}

// DeleteUser 删除用户
func (h *AdminHandler) DeleteUser(c *fiber.Ctx) error {
	id := c.Params("id")

	_, err := h.engine.ID(id).Delete(&models.User{})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to delete user",
		})
	}

	// 记录操作日志
	h.logAdminAction(c, "delete_user", id, nil)

	return c.JSON(fiber.Map{
		"message": "User deleted successfully",
	})
}

// ToggleUser 启用/禁用用户
func (h *AdminHandler) ToggleUser(c *fiber.Ctx) error {
	id := c.Params("id")

	// 查询用户
	var user models.User
	has, err := h.engine.ID(id).Get(&user)
	if err != nil || !has {
		return c.Status(404).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// 切换状态
	user.IsActive = !user.IsActive
	_, err = h.engine.ID(id).Cols("is_active").Update(&user)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to toggle user",
		})
	}

	// 记录操作日志
	h.logAdminAction(c, "toggle_user", id, map[string]interface{}{
		"isActive": user.IsActive,
	})

	return c.JSON(user)
}

// ResetUserTokenResponse 重置 token 响应
type ResetUserTokenResponse struct {
	Message string `json:"message"`
	Token   string `json:"token"`
}

// ResetUserToken 重置用户 token
func (h *AdminHandler) ResetUserToken(c *fiber.Ctx) error {
	id := c.Params("id")

	// 查询用户
	var user models.User
	has, err := h.engine.ID(id).Get(&user)
	if err != nil || !has {
		return c.Status(404).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// 生成新 token
	newToken := fmt.Sprintf("token-%s-%d", user.Username, time.Now().Unix())

	// 更新 token
	user.Token = newToken
	_, err = h.engine.ID(id).Cols("token").Update(&user)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to reset token",
		})
	}

	// 记录操作日志
	h.logAdminAction(c, "reset_token", id, nil)

	return c.JSON(ResetUserTokenResponse{
		Message: "Token reset successfully",
		Token:   newToken,
	})
}

// ListPublishLogsResponse 发布日志响应
type ListPublishLogsResponse struct {
	Data     []models.PublishLog `json:"data"`
	Total    int64               `json:"total"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"pageSize"`
}

// GetPublishLogs 获取发布日志
func (h *AdminHandler) GetPublishLogs(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	pageSize := c.QueryInt("pageSize", 20)
	status := c.Query("status", "")

	// 构建查询
	query := h.engine.Table("publish_logs")
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// 统计总数
	total, _ := query.Count()

	// 查询数据
	var logs []models.PublishLog
	query.OrderBy("created_at DESC").Limit(pageSize, (page-1)*pageSize).Find(&logs)

	return c.JSON(ListPublishLogsResponse{
		Data:     logs,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// ListAdminLogsResponse 管理员操作日志响应
type ListAdminLogsResponse struct {
	Data     []models.AdminLog `json:"data"`
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"pageSize"`
}

// GetAdminLogs 获取管理员操作日志
func (h *AdminHandler) GetAdminLogs(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	pageSize := c.QueryInt("pageSize", 20)
	action := c.Query("action", "")

	// 构建查询
	query := h.engine.Table("admin_logs")
	if action != "" {
		query = query.Where("action = ?", action)
	}

	// 统计总数
	total, _ := query.Count()

	// 查询数据
	var logs []models.AdminLog
	query.OrderBy("created_at DESC").Limit(pageSize, (page-1)*pageSize).Find(&logs)

	return c.JSON(ListAdminLogsResponse{
		Data:     logs,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// logAdminAction 记录管理员操作日志
func (h *AdminHandler) logAdminAction(c *fiber.Ctx, action, target string, details interface{}) {
	var detailsStr string
	if details != nil {
		bytes, err := json.Marshal(details)
		if err == nil {
			detailsStr = string(bytes)
		}
	}

	log := &models.AdminLog{
		Action:    action,
		Target:    target,
		Details:   detailsStr,
		IPAddr:    c.IP(),
		UserAgent: c.Get("User-Agent"),
	}

	h.engine.Insert(log)
}
