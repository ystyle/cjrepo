package handlers

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
	"xorm.io/xorm"

	"ystyle.top/go/cjrepo/internal/auth"
	"ystyle.top/go/cjrepo/internal/models"
	"ystyle.top/go/cjrepo/internal/protocol"
	"ystyle.top/go/cjrepo/internal/storage"
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
	CjcVersion    string                 `json:"cjc-version"`
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
	organization := c.Query("organization", "")

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

	// Validate token and get user
	user, err := h.validateTokenAndGetUser(token)
	if err != nil {
		h.logPublish(organization, packageName, "", "failed", "invalid token: "+err.Error(), c)
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
	if metaData.Name != packageName || metaData.Organization != organization {
		errMsg := fmt.Sprintf("metadata mismatch: expected (name=%s, org=%s), got (name=%s, org=%s)",
			packageName, organization, metaData.Name, metaData.Organization)
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

	// 5.5 Check permission using team-based permission system
	if !user.IsSuperuser {
		checker := auth.NewPermissionChecker(h.engine)
		versionExists, _ := h.checkPackageExists(organization, packageName, metaData.Version)

		var requiredPerm string
		if versionExists {
			requiredPerm = "overwrite"
		} else {
			requiredPerm = "write"
		}

		if !checker.CheckPermission(user.ID, organization, packageName, requiredPerm) {
			if versionExists {
				h.logPublish(organization, packageName, metaData.Version, "failed", "need overwrite permission", c)
				return c.Status(403).JSON(fiber.Map{
					"error": "version already exists, need overwrite permission",
				})
			}
			h.logPublish(organization, packageName, metaData.Version, "failed", "permission denied", c)
			return c.Status(403).JSON(fiber.Map{
				"error": "permission denied",
			})
		}
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

	// 7. Handle existing version (overwrite case)
	versionExists, _ := h.checkPackageExists(organization, packageName, metaData.Version)
	if versionExists {
		log.Printf("[INFO] Overwriting existing version: %s/%s-%s", organization, packageName, metaData.Version)
		// Delete old package record
		_, err := h.engine.Where("organization = ? AND name = ? AND version = ? AND deleted_at IS NULL",
			organization, packageName, metaData.Version).Delete(&models.Package{})
		if err != nil {
			log.Printf("[ERROR] Failed to delete old package: %v", err)
			h.logPublish(organization, packageName, metaData.Version, "failed", "failed to delete old version", c)
			return c.Status(500).JSON(fiber.Map{
				"error": "failed to overwrite package",
			})
		}
	}

	// 8. Save file
	if err := h.storageMgr.SaveTarball(organization, packageName, metaData.Version, req.Tarball); err != nil {
		log.Printf("[ERROR] Failed to save tarball: %v", err)
		h.logPublish(organization, packageName, metaData.Version, "failed", "failed to save file", c)
		return c.Status(500).JSON(fiber.Map{
			"error": "failed to save package",
		})
	}

	// 8.5. Extract README from tarball
	readmeContent := extractReadmeFromTarball(req.Tarball)
	if readmeContent != "" {
		log.Printf("[INFO] README extracted successfully, size: %d", len(readmeContent))
	} else {
		log.Printf("[INFO] No README found in package")
	}

	// 9. Store in database
	pkg := &models.Package{
		Organization:  organization,
		Name:          packageName,
		Version:       metaData.Version,
		CjcVersion:    metaData.CjcVersion,
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
		Readme:        readmeContent,
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
	// 只检查未删除的包
	has, err := h.engine.Where("organization = ? AND name = ? AND version = ? AND deleted_at IS NULL",
		org, name, version).Exist(&models.Package{})
	return has, err
}

func (h *PublishHandler) validateTokenAndGetUser(token string) (*models.User, error) {
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

func (h *PublishHandler) checkOrganizationPermission(userID int64, organizationName string) (bool, error) {
	// 查找组织
	var org models.Organization
	has, err := h.engine.Where("name = ? AND deleted_at IS NULL", organizationName).Get(&org)
	if err != nil {
		return false, err
	}
	if !has {
		// 组织不存在，允许创建（第一个上传到该组织的人会自动创建它）
		return true, nil
	}

	// 检查用户是否是组织成员
	var member models.OrganizationMember
	has, err = h.engine.Where("organization_i_d = ? AND user_i_d = ?", org.ID, userID).Get(&member)
	if err != nil {
		return false, err
	}
	return has, nil
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

// extractReadmeFromTarball extracts README content from tarball
func extractReadmeFromTarball(tarballData []byte) string {
	// Create gzip reader
	gzReader, err := gzip.NewReader(bytes.NewReader(tarballData))
	if err != nil {
		log.Printf("[ERROR] Failed to create gzip reader: %v", err)
		return ""
	}
	defer gzReader.Close()

	// Create tar reader
	tarReader := tar.NewReader(gzReader)

	// README filenames to look for (in priority order)
	readmeFiles := []string{
		"README_zh.md",
		"README.md",
	}

	// Map to store found README contents
	readmeContents := make(map[string]string)

	// Track all files for debugging
	var allFiles []string

	// Iterate through tar archive
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("[ERROR] Failed to read tar header: %v", err)
			return ""
		}

		// Track all files
		filename := header.Name
		allFiles = append(allFiles, filename)

		// Check if file is a README
		for _, readmeFile := range readmeFiles {
			// 匹配：文件名等于 README，或者路径以 README 结尾（不区分大小写）
			if strings.EqualFold(filename, readmeFile) || strings.HasSuffix(strings.ToLower(filename), "/"+strings.ToLower(readmeFile)) {
				// Read file content
				content, err := io.ReadAll(tarReader)
				if err != nil {
					log.Printf("[ERROR] Failed to read README file: %v", err)
					continue
				}
				readmeContents[readmeFile] = string(content)
				log.Printf("[INFO] Found README: %s (size: %d)", filename, len(content))
			}
		}
	}

	// Print all files for debugging
	log.Printf("[DEBUG] Tarball contains %d files: %v", len(allFiles), allFiles)

	// Return README in priority order
	for _, readmeFile := range readmeFiles {
		if content, ok := readmeContents[readmeFile]; ok {
			log.Printf("[INFO] Using README: %s (length: %d)", readmeFile, len(content))
			return content
		}
	}

	log.Printf("[WARN] No README found in tarball")
	return ""
}
