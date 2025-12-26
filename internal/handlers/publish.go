package handlers

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
	"xorm.io/xorm"

	"ystyle.top/go/cjrepo/internal/models"
	"ystyle.top/go/cjrepo/internal/storage"
	"ystyle.top/go/cjrepo/internal/protocol"
)

// PublishHandler handles package publishing
type PublishHandler struct {
	engine      *xorm.Engine
	storageMgr  *storage.Manager
}

// MetaData represents the meta-data.json structure
type MetaData struct {
	Organization  string                 `json:"organization"`
	Name          string                 `json:"name"`
	Version       string                 `json:"version"`
	Description   string                 `json:"description"`
	ArtifactType  string                 `json:"artifact-type"`
	Executable    bool                   `json:"executable"`
	Authors       []string               `json:"authors"`
	Repository    string                 `json:"repository"`
	Homepage      string                 `json:"homepage"`
	Documentation string                 `json:"documentation"`
	Tag           []string               `json:"tag"`
	Category      []string               `json:"category"`
	License       []string               `json:"license"`
	Index         map[string]interface{} `json:"index"`
	MetaVersion   int                    `json:"meta-version"`
}

// ArtifactIndex represents the index field in meta-data
type ArtifactIndex struct {
	SHA256Sum string `json:"sha256sum"`
}

// NewPublishHandler creates a new publish handler
func NewPublishHandler(engine *xorm.Engine, storageMgr *storage.Manager) *PublishHandler {
	return &PublishHandler{
		engine:     engine,
		storageMgr: storageMgr,
	}
}

// HandlePublish handles POST /pkg/{name}?organization={org}
func (h *PublishHandler) HandlePublish(c *fiber.Ctx) error {
	// 1. Extract parameters
	packageName := c.Params("name")
	organization := c.Query("organization", "default")

	log.Printf("[DEBUG] Received publish request: package=%s, org=%s", packageName, organization)

	if packageName == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "package name is required",
		})
	}

	// 2. Validate Authorization
	token := c.Get("Authorization")
	log.Printf("[DEBUG] Authorization token: %s", token)

	if token == "" {
		return c.Status(401).JSON(fiber.Map{
			"error": "authorization required",
		})
	}

	// Validate token
	if !h.validateToken(token) {
		h.logPublish(organization, packageName, "", "failed", "invalid token", c)
		return c.Status(403).JSON(fiber.Map{
			"error": "invalid token",
		})
	}

	// 3. Parse binary data
	body := c.BodyRaw()
	log.Printf("[DEBUG] Request body size: %d bytes", len(body))

	req, err := protocol.ParsePublishData(body)
	if err != nil {
		log.Printf("[ERROR] Failed to parse publish data: %v", err)
		h.logPublish(organization, packageName, "", "failed", err.Error(), c)
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid data format",
			"detail": err.Error(),
		})
	}

	log.Printf("[DEBUG] Parsed metadata size: %d, tarball size: %d", len(req.MetaData), len(req.Tarball))

	// 4. Parse meta-data.json
	var metaData MetaData
	if err := json.Unmarshal(req.MetaData, &metaData); err != nil {
		log.Printf("[ERROR] Failed to parse meta-data.json: %v", err)
		log.Printf("[DEBUG] meta-data.json content: %s", string(req.MetaData))
		h.logPublish(organization, packageName, "", "failed", "invalid meta-data.json: "+err.Error(), c)
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid meta-data.json",
			"detail": err.Error(),
		})
	}

	log.Printf("[DEBUG] Parsed metadata: name=%s, org=%s, version=%s", metaData.Name, metaData.Organization, metaData.Version)

	// 5. Validate package name and version match
	// Handle empty organization by treating it as "default"
	metaOrg := metaData.Organization
	if metaOrg == "" {
		metaOrg = "default"
	}

	if metaData.Name != packageName || metaOrg != organization {
		errMsg := fmt.Sprintf("metadata mismatch: expected (name=%s, org=%s), got (name=%s, org=%s)",
			packageName, organization, metaData.Name, metaOrg)
		log.Printf("[ERROR] %s", errMsg)
		h.logPublish(organization, packageName, metaData.Version, "failed", errMsg, c)
		return c.Status(400).JSON(fiber.Map{
			"error": "metadata mismatch",
			"expected": fiber.Map{
				"name":         packageName,
				"organization": organization,
			},
			"actual": fiber.Map{
				"name":         metaData.Name,
				"organization": metaData.Organization,
			},
		})
	}

	// 6. Validate SHA256
	expectedSHA256, ok := metaData.Index["sha256sum"].(string)
	if !ok {
		log.Printf("[ERROR] Missing sha256sum in metadata index")
		h.logPublish(organization, packageName, metaData.Version, "failed", "missing sha256sum", c)
		return c.Status(400).JSON(fiber.Map{
			"error": "missing sha256sum in metadata",
		})
	}

	actualSHA256 := protocol.CalculateSHA256(req.Tarball)
	log.Printf("[DEBUG] SHA256 - expected: %s, actual: %s", expectedSHA256, actualSHA256)

	if !protocol.ValidateTarballSHA256(req.Tarball, expectedSHA256) {
		h.logPublish(organization, packageName, metaData.Version, "failed", "checksum mismatch", c)
		return c.Status(400).JSON(fiber.Map{
			"error": "tarball checksum mismatch",
		})
	}

	// 7. Check if package already exists
	exists, err := h.checkPackageExists(organization, packageName, metaData.Version)
	if err != nil {
		log.Printf("[ERROR] Failed to check package existence: %v", err)
		h.logPublish(organization, packageName, metaData.Version, "failed", "database error", c)
		return c.Status(500).JSON(fiber.Map{
			"error": "database error",
		})
	}
	if exists {
		h.logPublish(organization, packageName, metaData.Version, "failed", "package version already exists", c)
		return c.Status(409).JSON(fiber.Map{
			"error": "package version already exists",
		})
	}

	// 8. Save file
	if err := h.storageMgr.SaveTarball(organization, packageName, metaData.Version, req.Tarball); err != nil {
		log.Printf("[ERROR] Failed to save tarball: %v", err)
		h.logPublish(organization, packageName, metaData.Version, "failed", "failed to save file", c)
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to save package",
		})
	}

	// 9. Store in database
	pkg := &models.Package{
		Organization:  organization,
		Name:          packageName,
		Version:       metaData.Version,
		Description:   metaData.Description,
		ArtifactType:  metaData.ArtifactType,
		Executable:    metaData.Executable,
		Authors:       toJSONString(metaData.Authors),
		Repository:    metaData.Repository,
		Homepage:      metaData.Homepage,
		Documentation: metaData.Documentation,
		Tags:          toJSONString(metaData.Tag),
		Categories:    toJSONString(metaData.Category),
		Licenses:      toJSONString(metaData.License),
		MetaVersion:   metaData.MetaVersion,
		MetaData:      string(req.MetaData),
		TarballPath:   h.storageMgr.GetTarballPath(organization, packageName, metaData.Version),
		TarballSize:   int64(len(req.Tarball)),
		TarballSHA256: expectedSHA256,
	}

	if _, err := h.engine.Insert(pkg); err != nil {
		log.Printf("[ERROR] Failed to insert package: %v", err)
		// Cleanup saved file
		h.storageMgr.DeleteTarball(organization, packageName, metaData.Version)
		h.logPublish(organization, packageName, metaData.Version, "failed", "database error: "+err.Error(), c)
		return c.Status(500).JSON(fiber.Map{
			"error": "database error",
		})
	}

	log.Printf("[INFO] Package published successfully: %s/%s-%s", organization, packageName, metaData.Version)

	// 10. Log success
	h.logPublish(organization, packageName, metaData.Version, "success", "", c)

	return c.Status(200).JSON(fiber.Map{
		"message": "package published successfully",
		"package": fiber.Map{
			"organization": organization,
			"name":        packageName,
			"version":     metaData.Version,
		},
	})
}

func (h *PublishHandler) checkPackageExists(org, name, version string) (bool, error) {
	has, err := h.engine.Where("organization = ? AND name = ? AND version = ?",
		org, name, version).Exist(&models.Package{})
	return has, err
}

func (h *PublishHandler) validateToken(token string) bool {
	has, _ := h.engine.Where("token = ? AND is_active = ?", token, true).Exist(&models.User{})
	return has
}

func (h *PublishHandler) logPublish(org, name, version, status, errMsg string, c *fiber.Ctx) {
	log := &models.PublishLog{
		Organization: org,
		PackageName:  name,
		Version:      version,
		Status:       status,
		ErrorMessage: errMsg,
		IPAddr:       c.IP(),
		UserAgent:    c.Get("User-Agent"),
	}
	h.engine.Insert(log)
}

// toJSONString converts a string slice to JSON string
func toJSONString(arr []string) string {
	if arr == nil {
		return "[]"
	}
	data, _ := json.Marshal(arr)
	return string(data)
}
