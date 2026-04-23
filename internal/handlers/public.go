package handlers

import (
	"log"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"xorm.io/xorm"

	"ystyle.top/go/cjrepo/internal/models"
	"ystyle.top/go/cjrepo/internal/version"
)

// PublicHandler 公开 API 处理器
type PublicHandler struct {
	engine *xorm.Engine
}

// NewPublicHandler 创建公开 API 处理器
func NewPublicHandler(engine *xorm.Engine) *PublicHandler {
	return &PublicHandler{
		engine: engine,
	}
}

// StatsResponse 统计信息响应
type StatsResponse struct {
	Packages    int64  `json:"packages"`
	Users       int64  `json:"users"`
	Versions    int64  `json:"versions"`
	Downloads   int64  `json:"downloads"`
	SiteName    string `json:"siteName"`
	BuildDate   string `json:"buildDate"`
	GitCommit   string `json:"gitCommit"`
	GitVersion  string `json:"gitVersion"`
}

// GetStats 获取公开统计信息
func (h *PublicHandler) GetStats(c *fiber.Ctx) error {
	// 从环境变量获取站点名称，默认为"仓颉包仓库"
	siteName := os.Getenv("CJREPO_SITE_NAME")
	if siteName == "" {
		siteName = "仓颉包仓库"
	}

	// 统计未删除的包数量
	packageCount, _ := h.engine.Table("packages").Where("deleted_at IS NULL").Count()

	// 统计激活的用户数量
	userCount, _ := h.engine.Table("users").Where("is_active = ?", true).Count()

	// 统计所有版本数量
	versionCount, _ := h.engine.Table("packages").Count()

	// 统计总下载量
	downloadCountFloat, _ := h.engine.Table("packages").Sum(&models.Package{}, "download_count")
	downloadCount := int64(downloadCountFloat)

	buildDate, gitCommit, gitVersion := version.GetBuildInfo()

	return c.JSON(StatsResponse{
		Packages:   packageCount,
		Users:      userCount,
		Versions:   versionCount,
		Downloads:  downloadCount,
		SiteName:   siteName,
		BuildDate:  buildDate,
		GitCommit:  gitCommit,
		GitVersion: gitVersion,
	})
}

// ListPackagesResponse 包列表响应
type PublicListPackagesResponse struct {
	Data     []models.Package `json:"data"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"pageSize"`
}

// ListPackages 获取包列表（公开）
func (h *PublicHandler) ListPackages(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	pageSize := c.QueryInt("pageSize", 20)
	search := c.Query("search", "")
	categories := c.Query("categories", "")

	log.Printf("[DEBUG] Query params: page=%d, pageSize=%d, search=%s, categories=%s",
		page, pageSize, search, categories)

	// 构建查询
	query := h.engine.Table("packages").Where("deleted_at IS NULL")

	// 解析搜索词，支持 org::keyword 格式
	searchOrg := ""
	searchKeyword := search
	if search != "" && strings.Contains(search, "::") {
		parts := strings.SplitN(search, "::", 2)
		if len(parts) == 2 {
			searchOrg = strings.TrimSpace(parts[0])
			searchKeyword = strings.TrimSpace(parts[1])
		}
		log.Printf("[DEBUG] Parsed search: org=%s, keyword=%s", searchOrg, searchKeyword)
	}

	// 如果有组织部分，添加组织筛选
	if searchOrg != "" {
		query = query.Where("organization = ?", searchOrg)
	}

	// 如果有关键词部分，搜索名称和描述
	if searchKeyword != "" {
		query = query.Where("name LIKE ? OR description LIKE ?", "%"+searchKeyword+"%", "%"+searchKeyword+"%")
	}
	if categories != "" {
		// 支持多个分类筛选，用逗号分隔
		categoryList := strings.Split(categories, ",")
		var conditions []string
		var args []interface{}

		for _, category := range categoryList {
			category = strings.TrimSpace(category)
			if category != "" {
				// categories 是 JSON 数组字符串，如 ["Network", "UI"]
				// 使用 JSON 查询包含指定分类的包
				conditions = append(conditions, "categories LIKE ?")
				args = append(args, "%\""+category+"\"%")
			}
		}

		if len(conditions) > 0 {
			// 多个分类之间使用 OR 关系，外层用括号包裹
			whereSQL := "(" + strings.Join(conditions, " OR ") + ")"
			log.Printf("[DEBUG] Category filter SQL: %s, args: %v", whereSQL, args)
			query = query.Where(whereSQL, args...)
		}
	}

	// 统计总数
	var total int64
	query.Count(&total)

	log.Printf("[DEBUG] Total packages found: %d", total)

	// 查询数据
	var packages []models.Package
	query.OrderBy("created_at DESC").Limit(pageSize, (page-1)*pageSize).Find(&packages)

	// 确保 packages 不为 nil
	if packages == nil {
		packages = []models.Package{}
	}

	return c.JSON(PublicListPackagesResponse{
		Data:     packages,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// PackageDetailResponse 包详情响应（包含所有版本）
type PackageDetailResponse struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Repository   string            `json:"repository"`
	Homepage     string            `json:"homepage"`
	Versions     []models.Package `json:"versions"`
}

// GetPackage 获取包详情（返回所有版本）
func (h *PublicHandler) GetPackage(c *fiber.Ctx) error {
	nameParam := c.Params("name")

	// 解析 org::name 格式
	var org, name string
	if strings.Contains(nameParam, "::") {
		parts := strings.SplitN(nameParam, "::", 2)
		org = parts[0]
		name = parts[1]
	} else {
		name = nameParam
	}

	// 获取所有版本
	var versions []models.Package
	var err error

	// Build query based on organization parameter
	if org == "" {
		// When org is empty, query for packages with empty organization
		err = h.engine.Where("(organization = '' OR organization IS NULL) AND name = ? AND deleted_at IS NULL",
			name).OrderBy("created_at DESC").Find(&versions)
	} else {
		// When org is specified, query for that organization
		err = h.engine.Where("organization = ? AND name = ? AND deleted_at IS NULL",
			org, name).OrderBy("created_at DESC").Find(&versions)
	}

	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to fetch package",
		})
	}

	if len(versions) == 0 {
		return c.Status(404).JSON(fiber.Map{
			"error": "Package not found",
		})
	}

	// 返回包详情
	response := PackageDetailResponse{
		Name:        versions[0].Name,
		Description: versions[0].Description,
		Repository:   versions[0].Repository,
		Homepage:     versions[0].Homepage,
		Versions:     versions,
	}

	return c.JSON(response)
}

// GetPackageVersion 获取特定版本详情
func (h *PublicHandler) GetPackageVersion(c *fiber.Ctx) error {
	name := c.Params("name")
	version := c.Params("version")
	org := c.Query("organization", "")

	var pkg models.Package
	var has bool
	var err error

	// Build query based on organization parameter
	if org == "" {
		// When org is empty, query for packages with empty organization
		has, err = h.engine.Where("(organization = '' OR organization IS NULL) AND name = ? AND version = ? AND deleted_at IS NULL",
			name, version).Get(&pkg)
	} else {
		// When org is specified, query for that organization
		has, err = h.engine.Where("organization = ? AND name = ? AND version = ? AND deleted_at IS NULL",
			org, name, version).Get(&pkg)
	}

	if err != nil || !has {
		return c.Status(404).JSON(fiber.Map{
			"error": "Package version not found",
		})
	}

	return c.JSON(pkg)
}

// GetOrganizations 获取组织列表
func (h *PublicHandler) GetOrganizations(c *fiber.Ctx) error {
	type OrgResponse struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}

	var orgs []OrgResponse
	err := h.engine.SQL("SELECT DISTINCT organization as name FROM packages WHERE deleted_at IS NULL ORDER BY organization ASC").Find(&orgs)

	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to fetch organizations",
		})
	}

	// 添加 ID
	for i := range orgs {
		orgs[i].ID = int64(i + 1)
	}

	return c.JSON(orgs)
}
