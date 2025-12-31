package handlers

import (
	"crypto/sha256"
	"fmt"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"xorm.io/xorm"

	"ystyle.top/go/cjrepo/internal/models"
)

// DownloadHandler handles package downloads
type DownloadHandler struct {
	engine     *xorm.Engine
	storageMgr interface {
		FileExists(org, name, version string) bool
		GetFile(org, name, version string) (*interface{}, error)
	}
}

// NewDownloadHandler creates a new download handler
func NewDownloadHandler(engine *xorm.Engine) *DownloadHandler {
	return &DownloadHandler{
		engine: engine,
	}
}

// HandleDownload handles GET /pkg/{name}/{version}?organization={org}
func (h *DownloadHandler) HandleDownload(c *fiber.Ctx) error {
	packageName := c.Params("name")
	version := c.Params("version")
	organization := c.Query("organization", "")

	log.Printf("[DEBUG] Download request: package=%s, version=%s, org=%s",
		packageName, version, organization)

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
		log.Printf("[WARN] Package not found: %s/%s-%s", organization, packageName, version)
		return c.Status(404).JSON(fiber.Map{
			"error": "package not found",
		})
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
