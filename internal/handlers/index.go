package handlers

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
	"xorm.io/xorm"

	"ystyle.top/go/cjrepo/internal/auth"
	"ystyle.top/go/cjrepo/internal/models"
)

// IndexHandler handles package index queries
type IndexHandler struct {
	engine       *xorm.Engine
	upstreamSync UpstreamSync
	requireAuth  bool // 是否需要认证
}

// NewIndexHandler creates a new index handler
func NewIndexHandler(engine *xorm.Engine, upstreamSync UpstreamSync, requireAuth bool) *IndexHandler {
	return &IndexHandler{
		engine:       engine,
		upstreamSync: upstreamSync,
		requireAuth:  requireAuth,
	}
}

// HandleIndex handles GET /index/{first}/{second}/{name}?organization={org}
// Path structure: /index/{first}/{second}/{full_package_name}
// - first: package name[0:2] - used for directory organization
// - second: package name[2:4] - used for subdirectory organization
// - name: full package name
// Example: "defer" -> /index/de/fe/defer
func (h *IndexHandler) HandleIndex(c *fiber.Ctx) error {
	first := c.Params("first")
	second := c.Params("second")
	fullName := c.Params("name")  // This is the complete package name
	organization := c.Query("organization", "")

	if len(fullName) < 4 {
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid package name",
		})
	}

	log.Printf("[DEBUG] Index query: first=%s, second=%s, fullName=%s, org=%s",
		first, second, fullName, organization)

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
			if !checker.CheckPermission(user.ID, organization, fullName, "read") {
				return c.Status(403).JSON(fiber.Map{
					"error": "permission denied",
				})
			}
		}
	}

	// Query matching packages by exact name
	var packages []models.Package
	var err error

	// Build query based on organization parameter
	if organization == "" {
		// When org is empty, query for packages with empty organization
		err = h.engine.Where("(organization = '' OR organization IS NULL) AND name = ?", fullName).
			OrderBy("created_at DESC").
			Find(&packages)
	} else {
		// When org is specified, query for that organization
		err = h.engine.Where("organization = ? AND name = ?", organization, fullName).
			OrderBy("created_at DESC").
			Find(&packages)
	}

	if err != nil {
		log.Printf("Database error: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "database error",
		})
	}

	log.Printf("[DEBUG] Found %d packages for %s", len(packages), fullName)

	// 如果本地没有包，尝试从上游获取索引
	if len(packages) == 0 && h.upstreamSync != nil {
		log.Printf("[INFO] No local packages found, trying upstream")

		upstream, err := h.upstreamSync.GetEnabledUpstream()
		if err == nil && upstream != nil && upstream.Enabled {
			// 从上游获取索引
			indexData, err := h.fetchIndexFromUpstream(upstream, fullName, organization)
			if err == nil && len(indexData) > 0 {
				log.Printf("[INFO] Successfully fetched index from upstream")
				c.Set("Content-Type", "application/x-ndjson")
				c.Write(indexData)
				return nil
			}
			log.Printf("[WARN] Failed to fetch index from upstream: %v", err)
		}
	}

	// Generate JSON Lines format (ArtifactIndex structure)
	c.Set("Content-Type", "application/x-ndjson")
	hasIndex := false
	for _, pkg := range packages {
		// Parse metadata
		var metaData map[string]interface{}
		if pkg.MetaData != "" && pkg.MetaData != "null" {
			json.Unmarshal([]byte(pkg.MetaData), &metaData)
		}

		// Extract index field
		indexField := metaData["index"]
		if indexField == nil {
			log.Printf("[WARN] No index field in metadata for %s, generating basic index", pkg.Name)
		}

		// Build ArtifactIndex structure
		// 优先使用 index 字段，如果没有则使用包的基本信息
		artifactIndex := fiber.Map{
			"organization":       pkg.Organization,
			"name":               pkg.Name,
			"version":            pkg.Version,
			"dependencies":       []interface{}{},
			"test-dependencies":  []interface{}{},
			"script-dependencies": []interface{}{},
		}

		// 如果有 index 字段，使用其中的信息
		if indexField != nil {
			if idxMap, ok := indexField.(map[string]interface{}); ok {
				if sha256, ok := idxMap["sha256sum"].(string); ok {
					artifactIndex["sha256sum"] = sha256
				} else {
					artifactIndex["sha256sum"] = pkg.TarballSHA256
				}
				if cjcVersion, ok := idxMap["cjc-version"].(string); ok {
					artifactIndex["cjc-version"] = cjcVersion
				}
				// 提取依赖信息
				if deps, ok := idxMap["dependencies"].([]interface{}); ok {
					artifactIndex["dependencies"] = deps
				}
				if testDeps, ok := idxMap["test-dependencies"].([]interface{}); ok {
					artifactIndex["test-dependencies"] = testDeps
				}
				if scriptDeps, ok := idxMap["script-dependencies"].([]interface{}); ok {
					artifactIndex["script-dependencies"] = scriptDeps
				}
				// 提取 yanked 和 index-version
				if yanked, ok := idxMap["yanked"].(bool); ok {
					artifactIndex["yanked"] = yanked
				}
				if indexVersion, ok := idxMap["index-version"].(float64); ok {
					artifactIndex["index-version"] = int(indexVersion)
				}
			}
		} else {
			// 使用数据库中的 SHA256
			artifactIndex["sha256sum"] = pkg.TarballSHA256
		}

		artifactIndex["yanked"] = false
		artifactIndex["index-version"] = 1

		line, _ := json.Marshal(artifactIndex)
		c.Write(line)
		c.Write([]byte("\n"))
		hasIndex = true
	}

	if !hasIndex {
		log.Printf("[WARN] No valid index generated for %s, returning 404", fullName)
		return c.Status(404).JSON(fiber.Map{
			"error": "package not found",
		})
	}

	return nil
}

// fetchIndexFromUpstream 从上游获取索引数据
func (h *IndexHandler) fetchIndexFromUpstream(upstream *models.Upstream, name, org string) ([]byte, error) {
	return h.upstreamSync.FetchIndex(upstream, name, org)
}

// validateTokenAndGetUser 验证 token 并返回用户
func (h *IndexHandler) validateTokenAndGetUser(token string) (*models.User, error) {
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
