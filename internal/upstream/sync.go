package upstream

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"xorm.io/xorm"

	"ystyle.top/go/cjrepo/internal/models"
)

// Sync 上游同步器
type Sync struct {
	engine     *xorm.Engine
	httpClient *http.Client
}

// NewSync 创建上游同步器
func NewSync(engine *xorm.Engine) *Sync {
	return &Sync{
		engine: engine,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetEnabledUpstream 获取启用的上游（优先级最高的）
func (s *Sync) GetEnabledUpstream() (*models.Upstream, error) {
	var upstream models.Upstream
	has, err := s.engine.Where("enabled = ?", true).OrderBy("id DESC").Get(&upstream)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, nil
	}
	return &upstream, nil
}

// FetchPackageFromUpstream 从上游获取包信息
func (s *Sync) FetchPackageFromUpstream(upstream *models.Upstream, name, version, org string) (*models.Package, []byte, error) {
	// 先尝试从索引获取元数据
	indexData, err := s.fetchIndexFromUpstream(upstream, name, org)
	if err != nil {
		return nil, nil, fmt.Errorf("fetch index failed: %w", err)
	}

	// 解析索引，找到指定版本
	pkg, err := s.parseIndexForVersion(indexData, name, version, org)
	if err != nil {
		return nil, nil, fmt.Errorf("parse index failed: %w", err)
	}

	// 下载tarball
	tarballData, err := s.DownloadFromUpstream(upstream, pkg)
	if err != nil {
		return nil, nil, fmt.Errorf("download tarball failed: %w", err)
	}

	return pkg, tarballData, nil
}

// buildPackageURL 构建包的上游URL
func (s *Sync) buildPackageURL(baseURL, name, version, org string) string {
	// 仓颉中央库格式: https://pkg.cangjie-lang.cn/cjpm/pkg/{name}/{version}?organization={org}
	baseURL = strings.TrimSuffix(baseURL, "/")

	url := fmt.Sprintf("%s/pkg/%s/%s", baseURL, name, version)
	if org != "" {
		url += "?organization=" + org
	}
	return url
}

// buildIndexURL 构建索引URL
// 格式: /index/${mo}/${du}/${name}?organization=${org}
func (s *Sync) buildIndexURL(baseURL, name, org string) string {
	baseURL = strings.TrimSuffix(baseURL, "/")

	// 计算索引路径
	if len(name) < 2 {
		name = name + "__" // 补齐长度
	}
	mo := name[0:2]
	du := string(name[1])

	url := fmt.Sprintf("%s/index/%s/%s/%s", baseURL, mo, du, name)
	if org != "" {
		url += "?organization=" + org
	}
	return url
}

// fetchIndexFromUpstream 从上游获取索引
func (s *Sync) fetchIndexFromUpstream(upstream *models.Upstream, name, org string) ([]byte, error) {
	url := s.buildIndexURL(upstream.URL, name, org)

	log.Printf("[INFO] Fetching index from upstream: %s", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create index request failed: %w", err)
	}

	if upstream.AuthToken != "" {
		req.Header.Set("Authorization", upstream.AuthToken)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch index failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upstream index returned status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// FetchIndex 公开的索引获取方法
func (s *Sync) FetchIndex(upstream *models.Upstream, name, org string) ([]byte, error) {
	return s.fetchIndexFromUpstream(upstream, name, org)
}

// parseIndexForVersion 从索引数据中解析指定版本的包
func (s *Sync) parseIndexForVersion(indexData []byte, name, version, org string) (*models.Package, error) {
	// 索引文件每行是一个JSON条目
	lines := strings.Split(string(indexData), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 解析JSON索引条目
		var indexEntry struct {
			Organization string `json:"organization"`
			Name         string `json:"name"`
			Version      string `json:"version"`
			SHA256       string `json:"sha256sum"`
		}

		if err := json.Unmarshal([]byte(line), &indexEntry); err != nil {
			log.Printf("[WARN] Failed to parse index line: %s, error: %v", line, err)
			continue
		}

		// 查找匹配的版本
		if indexEntry.Name == name && indexEntry.Version == version {
			// 检查组织是否匹配
			if org == "" && indexEntry.Organization != "" {
				continue
			}
			if org != "" && indexEntry.Organization != org {
				continue
			}

			// 找到了，创建包对象
			pkg := &models.Package{
				Name:         name,
				Version:      version,
				Organization: org,
				TarballSHA256: indexEntry.SHA256,
			}

			return pkg, nil
		}
	}

	return nil, fmt.Errorf("version %s not found in index", version)
}

// FetchAndSavePackage 从上游拉取并保存包到本地
func (s *Sync) FetchAndSavePackage(upstream *models.Upstream, name, version, org string) (*models.Package, error) {
	log.Printf("[INFO] Fetching package from upstream: %s/%s (org: %s)", name, version, org)

	// 检查本地是否已存在
	var existingPkg models.Package
	has, err := s.engine.Where("name = ? AND version = ? AND (organization = ? OR organization IS NULL OR organization = '')", name, version, org).Get(&existingPkg)
	if err != nil {
		return nil, fmt.Errorf("check existing package failed: %w", err)
	}

	// 如果已存在且缓存未过期，直接返回
	if has {
		if !s.IsCacheExpired(&existingPkg, upstream.CacheTTL) {
			log.Printf("[INFO] Package cache hit: %s/%s", name, version)
			return &existingPkg, nil
		}
		log.Printf("[INFO] Package cache expired: %s/%s, fetching from upstream", name, version)
	}

	// 从上游获取包和tarball
	pkg, tarballData, err := s.FetchPackageFromUpstream(upstream, name, version, org)
	if err != nil {
		return nil, fmt.Errorf("fetch package from upstream failed: %w", err)
	}

	// 计算存储路径
	storagePath := s.getStoragePath(org, name, version)

	// 保存tarball到本地
	if err := s.saveTarball(storagePath, tarballData); err != nil {
		return nil, fmt.Errorf("save tarball failed: %w", err)
	}

	// 更新包信息
	pkg.TarballPath = storagePath
	pkg.TarballSize = int64(len(tarballData))
	pkg.TarballSHA256 = fmt.Sprintf("%x", sha256.Sum256(tarballData))
	pkg.UpstreamID = upstream.ID
	pkg.UpstreamName = upstream.Name

	// 如果包已存在，更新；否则插入
	if has {
		pkg.ID = existingPkg.ID
		pkg.CreatedAt = existingPkg.CreatedAt
		pkg.UpdatedAt = time.Now()
		pkg.DownloadCount = existingPkg.DownloadCount

		_, err = s.engine.ID(pkg.ID).Update(pkg)
		if err != nil {
			return nil, fmt.Errorf("update package failed: %w", err)
		}
	} else {
		pkg.CreatedAt = time.Now()
		pkg.UpdatedAt = time.Now()

		_, err = s.engine.Insert(pkg)
		if err != nil {
			return nil, fmt.Errorf("insert package failed: %w", err)
		}
	}

	// 更新上游最后同步时间
	upstream.LastSyncAt = time.Now()
	s.engine.ID(upstream.ID).Update(upstream)

	log.Printf("[INFO] Successfully synced package from upstream: %s/%s", name, version)
	return pkg, nil
}

// getStoragePath 获取包的存储路径
func (s *Sync) getStoragePath(org, name, version string) string {
	// 格式: storage/{org}/{name}/{version}.cjp 或 storage/{name}/{version}.cjp
	if org != "" {
		return fmt.Sprintf("storage/%s/%s/%s.cjp", org, name, version)
	}
	return fmt.Sprintf("storage/%s/%s.cjp", name, version)
}

// saveTarball 保存tarball到本地
func (s *Sync) saveTarball(path string, data []byte) error {
	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory failed: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write file failed: %w", err)
	}

	return nil
}

// IsCacheExpired 检查缓存是否过期
func (s *Sync) IsCacheExpired(pkg *models.Package, ttl int) bool {
	if ttl <= 0 {
		return false // 缓存永不过期
	}
	return time.Since(pkg.UpdatedAt) > time.Duration(ttl)*time.Second
}

// DownloadFromUpstream 从上游下载tarball
func (s *Sync) DownloadFromUpstream(upstream *models.Upstream, pkg *models.Package) ([]byte, error) {
	// 构建下载URL
	// 假设下载接口格式: {baseURL}/pkg/{name}/{version}/download?organization={org}
	downloadURL := s.buildDownloadURL(upstream.URL, pkg.Name, pkg.Version, pkg.Organization)

	log.Printf("[INFO] Downloading tarball from upstream: %s", downloadURL)

	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create download request failed: %w", err)
	}

	if upstream.AuthToken != "" {
		req.Header.Set("Authorization", upstream.AuthToken)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download from upstream failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upstream download returned status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// buildDownloadURL 构建下载URL
func (s *Sync) buildDownloadURL(baseURL, name, version, org string) string {
	baseURL = strings.TrimSuffix(baseURL, "/")
	url := fmt.Sprintf("%s/pkg/%s/%s", baseURL, name, version)
	if org != "" {
		url += "?organization=" + org
	}
	return url
}

// GetPackageMetadata 从上游获取包的元数据
func (s *Sync) GetPackageMetadata(upstream *models.Upstream, name string) (map[string]interface{}, error) {
	// 构建元数据查询URL
	// 假设格式: {baseURL}/packages/{name}
	metadataURL := strings.TrimSuffix(upstream.URL, "/") + "/packages/" + name

	log.Printf("[INFO] Fetching package metadata from upstream: %s", metadataURL)

	req, err := http.NewRequest("GET", metadataURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create metadata request failed: %w", err)
	}

	if upstream.AuthToken != "" {
		req.Header.Set("Authorization", upstream.AuthToken)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch metadata from upstream failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upstream metadata returned status %d", resp.StatusCode)
	}

	var metadata map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return nil, fmt.Errorf("decode metadata failed: %w", err)
	}

	return metadata, nil
}

// ProxyRequest 代理请求到上游
func (s *Sync) ProxyRequest(upstream *models.Upstream, path string) (*http.Response, error) {
	// 构建完整的上游URL
	upstreamURL := strings.TrimSuffix(upstream.URL, "/") + "/" + strings.TrimPrefix(path, "/")

	log.Printf("[INFO] Proxying request to upstream: %s", upstreamURL)

	req, err := http.NewRequest("GET", upstreamURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create proxy request failed: %w", err)
	}

	if upstream.AuthToken != "" {
		req.Header.Set("Authorization", upstream.AuthToken)
	}

	// 复制原始请求的头部
	// req.Header.Set("User-Agent", "cjrepo/1.0")

	return s.httpClient.Do(req)
}

// ProxyDownload 代理下载请求到上游并缓存
func (s *Sync) ProxyDownload(upstream *models.Upstream, pkg *models.Package, writer io.Writer) error {
	// 从上游下载
	data, err := s.DownloadFromUpstream(upstream, pkg)
	if err != nil {
		return err
	}

	// 写入响应
	_, err = writer.Write(data)
	return err
}
