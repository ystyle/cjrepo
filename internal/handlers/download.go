package handlers

import (
	"crypto/sha256"
	"fmt"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"xorm.io/xorm"

	"ystyle.top/go/cjrepo/internal/auth"
	"ystyle.top/go/cjrepo/internal/models"
)

// UpstreamSync 上游同步器接口
type UpstreamSync interface {
	GetEnabledUpstream() (*models.Upstream, error)
	FetchAndSavePackage(upstream *models.Upstream, name, version, org string) (*models.Package, error)
	FetchIndex(upstream *models.Upstream, name, org string) ([]byte, error)
}

// DownloadHandler handles package downloads
type DownloadHandler struct {
	engine       *xorm.Engine
	storageMgr   interface {
		FileExists(org, name, version string) bool
		GetFile(org, name, version string) (*interface{}, error)
	}
	upstreamSync UpstreamSync
	requireAuth  bool // 是否需要认证
}

// NewDownloadHandler creates a new download handler
func NewDownloadHandler(engine *xorm.Engine, upstreamSync UpstreamSync, requireAuth bool) *DownloadHandler {
	return &DownloadHandler{
		engine:       engine,
		upstreamSync: upstreamSync,
		requireAuth:  requireAuth,
	}
}

// HandleDownload handles GET /pkg/{name}/{version}?organization={org}
func (h *DownloadHandler) HandleDownload(c *fiber.Ctx) error {
	packageName := c.Params("name")
	version := c.Params("version")
	organization := c.Query("organization", "")

	log.Printf("[DEBUG] Download request: package=%s, version=%s, org=%s",
		packageName, version, organization)

	// 如果开启强制认证，验证 token 并检查权限
	if h.requireAuth {
		token := c.Get("Authorization")
		if token == "" {
			return c.Status(401).JSON(fiber.Map{
				"error": "authorization required",
			})
		}

		user, err := h.validateTokenAndGetUser(token)
		if err != nil {
			return c.Status(403).JSON(fiber.Map{
				"error": "invalid token",
			})
		}

		// Check permission using team-based system
		if !user.IsSuperuser {
			checker := auth.NewPermissionChecker(h.engine)
			if !checker.CheckPermission(user.ID, organization, packageName, "read") {
				return c.Status(403).JSON(fiber.Map{
					"error": "permission denied",
				})
			}
		}
	}

	// Query package from database
	var pkg models.Package
	var has bool
	var err error

	// Build query based on organization parameter
	if organization == "" {
		// When org is empty, query for packages with empty organization
		has, err = h.engine.Where("(organization = '' OR organization IS NULL) AND name = ? AND version = ?",
			packageName, version).Get(&pkg)
	} else {
		// When org is specified, query for that organization
		has, err = h.engine.Where("organization = ? AND name = ? AND version = ?",
			organization, packageName, version).Get(&pkg)
	}

	if err != nil {
		log.Printf("[ERROR] Database error: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "database error",
		})
	}

	if !has {
		log.Printf("[WARN] Package not found locally: %s/%s-%s", organization, packageName, version)

		// 尝试从上游同步（如果配置了上游）
		if h.upstreamSync != nil {
			upstream, err := h.upstreamSync.GetEnabledUpstream()
			if err == nil && upstream != nil && upstream.Enabled {
				log.Printf("[INFO] Trying to fetch from upstream: %s", upstream.Name)

				// 从上游获取包信息并保存到本地
				syncedPkg, err := h.upstreamSync.FetchAndSavePackage(upstream, packageName, version, organization)
				if err == nil {
					log.Printf("[INFO] Successfully synced from upstream: %s/%s", packageName, version)
					pkg = *syncedPkg
					has = true
				} else {
					log.Printf("[ERROR] Failed to sync from upstream: %v", err)
					// 继续返回404
				}
			}
		}

		// 如果仍然没有找到，返回404
		if !has {
			return c.Status(404).JSON(fiber.Map{
				"error": "package not found",
			})
		}
	}

	log.Printf("[INFO] Downloading package: %s/%s-%s from %s",
		organization, packageName, version, pkg.TarballPath)

	// Increment download count using xorm Incr with Exec
	result, err := h.engine.ID(pkg.ID).Incr("download_count", 1).Update(&models.Package{})
	if err != nil {
		log.Printf("[ERROR] Failed to increment download count: %v", err)
	} else {
		log.Printf("[INFO] Download count incremented: pkgID=%d, affected=%d", pkg.ID, result)
	}

	// Read entire file into memory (for reliable transmission)
	data, err := os.ReadFile(pkg.TarballPath)
	if err != nil {
		log.Printf("[ERROR] Failed to read file: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to read package file",
		})
	}

	log.Printf("[DEBUG] File size: %d bytes, SHA256: %x", len(data), sha256.Sum256(data))

	// Set headers
	c.Set("Content-Type", "application/x-gzip")
	c.Set("Content-Disposition", "attachment; filename=\""+packageName+"-"+version+".cjp\"")
	c.Set("Content-Length", fmt.Sprintf("%d", len(data)))
	c.Set("Connection", "close")

	// Send data directly
	return c.Send(data)
}

// validateTokenAndGetUser 验证 token 并返回用户
func (h *DownloadHandler) validateTokenAndGetUser(token string) (*models.User, error) {
	var user models.User
	has, err := h.engine.Where("token = ? AND is_active = ?", token, true).Get(&user)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, fmt.Errorf("token not found")
	}
	return &user, nil
}
