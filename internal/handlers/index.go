package handlers

import (
	"encoding/json"
	"log"

	"github.com/gofiber/fiber/v2"
	"xorm.io/xorm"

	"ystyle.top/go/cjrepo/internal/models"
)

// IndexHandler handles package index queries
type IndexHandler struct {
	engine *xorm.Engine
}

// NewIndexHandler creates a new index handler
func NewIndexHandler(engine *xorm.Engine) *IndexHandler {
	return &IndexHandler{
		engine: engine,
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

	// Generate JSON Lines format (ArtifactIndex structure)
	c.Set("Content-Type", "application/x-ndjson")
	for _, pkg := range packages {
		// Parse metadata
		var metaData map[string]interface{}
		json.Unmarshal([]byte(pkg.MetaData), &metaData)

		// Extract index field
		indexField := metaData["index"]
		if indexField == nil {
			log.Printf("[WARN] No index field in metadata for %s", pkg.Name)
			continue
		}

		// Build ArtifactIndex structure
		artifactIndex := fiber.Map{
			"organization":      pkg.Organization,
			"name":              pkg.Name,
			"version":           pkg.Version,
			"dependencies":      []interface{}{},
			"testDependencies":  []interface{}{},
			"scriptDependencies": []interface{}{},
			"sha256sum":         indexField.(map[string]interface{})["sha256sum"],
			"yanked":           false,
			"cjc-version":       indexField.(map[string]interface{})["cjc-version"],
			"index-version":     1,
		}

		line, _ := json.Marshal(artifactIndex)
		c.Write(line)
		c.Write([]byte("\n"))
	}

	return nil
}
