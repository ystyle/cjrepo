package handlers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"xorm.io/xorm"

	"ystyle.top/go/cjrepo/internal/models"
	"ystyle.top/go/cjrepo/internal/semverutil"
	"ystyle.top/go/cjrepo/internal/task"
	upstream2 "ystyle.top/go/cjrepo/internal/upstream"
)

// metaData 对应包元数据 JSON 结构（用于解析依赖）
type metaData struct {
	Dependencies        []depEntry `json:"dependencies"`
	TestDependencies    []depEntry `json:"test-dependencies"`
	ScriptDependencies  []depEntry `json:"script-dependencies"`
	// 某些版本的 cjpm 将依赖放在 index 字段内
	Index *struct {
		Dependencies       []depEntry `json:"dependencies"`
		TestDependencies   []depEntry `json:"test-dependencies"`
		ScriptDependencies []depEntry `json:"script-dependencies"`
	} `json:"index,omitempty"`
}

type depEntry struct {
	Name         string `json:"name"`
	Require      string `json:"require"`
	Organization string `json:"organization"`
	Target       string `json:"target"`
}

type analyzeResult struct {
	PackageID    int64    `json:"package_id"`
	Organization string   `json:"organization"`
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	SHA256       string   `json:"sha256"`
	RemoteSHA256 string   `json:"remote_sha256,omitempty"`
	Category     string   `json:"category"`
	Selected     bool     `json:"selected"`
	Requirement  string   `json:"requirement,omitempty"`
	LocalVersions []string `json:"local_versions,omitempty"`
}

// resolvePackages 递归解析依赖，返回拓扑排序结果
func (h *PublishPlanHandler) resolvePackages(packageIDs []int64, targetUpstream string) ([]analyzeResult, []int64, error) {
	if len(packageIDs) == 0 {
		return nil, nil, nil
	}

	var results []analyzeResult
	seen := make(map[int64]bool)
	visited := make(map[int64]bool)
	var order []int64

	var resolve func(pid int64, requirement string)
	resolve = func(pid int64, requirement string) {
		if visited[pid] {
			return
		}
		if seen[pid] {
			for i := range results {
				if results[i].PackageID == pid && requirement != "" {
					results[i].Requirement = requirement
				}
			}
			return
		}
		visited[pid] = true

		var pkg models.Package
		if ok, _ := h.engine.ID(pid).Get(&pkg); !ok {
			delete(visited, pid)
			return
		}

		seen[pid] = true
		results = append(results, analyzeResult{
			PackageID:    pkg.ID,
			Organization: pkg.Organization,
			Name:         pkg.Name,
			Version:      pkg.Version,
			SHA256:       pkg.TarballSHA256,
			Category:     "need_publish",
			Selected:     true,
			Requirement:  requirement,
		})

		deps := parseDependencies(pkg.MetaData)
		for _, dep := range deps {
			child, err := h.findPackage(dep)
			if err != nil {
				continue
			}
			resolve(child.ID, dep.Require)
		}

		delete(visited, pid)
		order = append(order, pid)
	}

	for _, pid := range packageIDs {
		resolve(pid, "")
	}

	return results, order, nil
}

func parseOrgName(raw string) (org, name string) {
	if strings.Contains(raw, "::") {
		parts := strings.SplitN(raw, "::", 2)
		return parts[0], parts[1]
	}
	return "", raw
}

type PublishPlanHandler struct {
	engine       *xorm.Engine
	tm           *task.TaskManager
	upstreamSync *upstream2.Sync
}

func NewPublishPlanHandler(engine *xorm.Engine, tm *task.TaskManager, upstreamSync *upstream2.Sync) *PublishPlanHandler {
	return &PublishPlanHandler{engine: engine, tm: tm, upstreamSync: upstreamSync}
}

// List 计划列表
func (h *PublishPlanHandler) List(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	pageSize := c.QueryInt("pageSize", 20)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	total, _ := h.engine.Count(&models.PublishPlan{})

	var plans []models.PublishPlan
	h.engine.Desc("created_at").Limit(pageSize, (page-1)*pageSize).Find(&plans)
	if plans == nil {
		plans = []models.PublishPlan{}
	}

	return c.JSON(fiber.Map{
		"data":     plans,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// Get 计划详情
type itemWithPkg struct {
	models.PublishPlanItem
	PackageName          string `json:"package_name"`
	PackageOrganization string `json:"package_organization"`
	PackageVersion      string `json:"package_version"`
}

func (h *PublishPlanHandler) Get(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}

	var plan models.PublishPlan
	ok, err := h.engine.ID(id).Get(&plan)
	if err != nil || !ok {
		return c.Status(404).JSON(fiber.Map{"error": "plan not found"})
	}

	var items []models.PublishPlanItem
	h.engine.Where("plan_i_d = ?", id).OrderBy("\"order\"").Find(&items)

	itemsWithPkg := make([]itemWithPkg, len(items))
	for i, item := range items {
		itemsWithPkg[i] = itemWithPkg{PublishPlanItem: item}
		var pkg models.Package
		if ok, _ := h.engine.ID(item.PackageID).Get(&pkg); ok {
			itemsWithPkg[i].PackageName = pkg.Name
			itemsWithPkg[i].PackageOrganization = pkg.Organization
			itemsWithPkg[i].PackageVersion = pkg.Version
		}
	}

	return c.JSON(fiber.Map{
		"plan":  plan,
		"items": itemsWithPkg,
	})
}

// Create 创建计划
func (h *PublishPlanHandler) Create(c *fiber.Ctx) error {
	var req struct {
		Name           string  `json:"name"`
		TargetUpstream int64   `json:"target_upstream"`
		PackageIDs     []int64 `json:"package_ids"`
		PollInterval   int     `json:"poll_interval"`
		PollTimeout    int     `json:"poll_timeout"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if req.Name == "" || req.TargetUpstream == 0 || len(req.PackageIDs) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "name, target_upstream and package_ids required"})
	}

	plan := &models.PublishPlan{
		Name:           req.Name,
		TargetUpstream: req.TargetUpstream,
		Status:         "pending",
		TotalCount:     len(req.PackageIDs),
		PollInterval:   req.PollInterval,
		PollTimeout:    req.PollTimeout,
	}
	if plan.PollInterval <= 0 {
		plan.PollInterval = 60
	}
	if plan.PollTimeout <= 0 {
		plan.PollTimeout = 600
	}
	if _, err := h.engine.Insert(plan); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "insert failed"})
	}

	for i, pid := range req.PackageIDs {
		item := &models.PublishPlanItem{
			PlanID:    plan.ID,
			PackageID: pid,
			Order:     i + 1,
			Status:    "pending",
			Selected:  true,
		}
		if _, err := h.engine.Insert(item); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "insert item failed"})
		}
	}

	return c.JSON(plan)
}

// UpdateItems 更新计划项选择状态
func (h *PublishPlanHandler) UpdateItems(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}

	var req struct {
		SelectedIDs []int64 `json:"selected_ids"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}

	h.engine.Where("plan_i_d = ?", id).Cols("selected").Update(&models.PublishPlanItem{Selected: false})
	for _, pid := range req.SelectedIDs {
		h.engine.ID(pid).Cols("selected").Update(&models.PublishPlanItem{Selected: true})
	}

	return c.JSON(fiber.Map{"status": "ok"})
}

// Delete 删除计划
func (h *PublishPlanHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}

	h.tm.Pause(id)

	h.engine.Where("plan_i_d = ?", id).Delete(&models.PublishPlanItem{})
	h.engine.ID(id).Delete(&models.PublishPlan{})
	return c.JSON(fiber.Map{"status": "deleted"})
}

// Start 开始执行
func (h *PublishPlanHandler) Start(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}

	h.engine.Where("plan_i_d = ? AND status = ?", id, "failed").Cols("status", "error").
		Update(&models.PublishPlanItem{Status: "pending"})

	if err := h.tm.Start(id); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"status": "started"})
}

// Pause 暂停
func (h *PublishPlanHandler) Pause(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}
	if err := h.tm.Pause(id); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"status": "paused"})
}

// Resume 恢复
func (h *PublishPlanHandler) Resume(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}
	if err := h.tm.Resume(id); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"status": "resumed"})
}

// Events SSE 事件流
func (h *PublishPlanHandler) Events(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid id"})
	}

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Status(200)

	ch := h.tm.Subscribe(id)
	defer h.tm.Unsubscribe(id, ch)

	ctx := c.Context()
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		for {
			select {
			case evt, ok := <-ch:
				if !ok {
					return
				}
				data, _ := json.Marshal(evt)
				fmt.Fprintf(w, "data: %s\n\n", data)
				w.Flush()
			case <-ticker.C:
				fmt.Fprintf(w, ": keepalive\n\n")
				w.Flush()
			case <-ctx.Done():
				return
			}
		}
	})
	return nil
}

// Analyze 分析依赖
func (h *PublishPlanHandler) Analyze(c *fiber.Ctx) error {
	var req struct {
		PackageIDs     []int64 `json:"package_ids"`
		TargetUpstream int64   `json:"target_upstream"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request"})
	}
	if len(req.PackageIDs) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "package_ids required"})
	}

	results, order, err := h.resolvePackages(req.PackageIDs, "")
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// 收集本地版本信息
	for i, item := range results {
		var allVersions []models.Package
		h.engine.Where("name = ? AND (organization = ? OR organization = '' OR organization IS NULL)", item.Name, item.Organization).Find(&allVersions)
		var versions []string
		for _, v := range allVersions {
			versions = append(versions, v.Version)
		}
		semverutil.Sort(versions)
		results[i].LocalVersions = versions

		if item.Requirement != "" {
			match, err := semverutil.Check(item.Version, item.Requirement)
			if err == nil && !match {
				best, err := semverutil.BestMatch(versions, item.Requirement)
				if err == nil && best != item.Version {
					results[i].Category = "version_optional"
				}
			}
		}
	}

	// 获取上游索引用于版本对比
	remoteIndex := make(map[string]map[string]string) // org::name -> version -> sha256
	if req.TargetUpstream > 0 {
		var up models.Upstream
		if ok, _ := h.engine.ID(req.TargetUpstream).Get(&up); ok {
			for _, item := range results {
				data, err := h.upstreamSync.FetchIndex(&up, item.Name, item.Organization)
				if err != nil || len(data) == 0 {
					continue
				}
				key := item.Name
				if item.Organization != "" {
					key = item.Organization + "::" + item.Name
				}
				for _, line := range strings.Split(string(data), "\n") {
					line = strings.TrimSpace(line)
					if line == "" {
						continue
					}
					var entry struct {
						Name     string `json:"name"`
						Version  string `json:"version"`
						SHA256   string `json:"sha256sum"`
					}
					if json.Unmarshal([]byte(line), &entry) != nil {
						continue
					}
					if remoteIndex[key] == nil {
						remoteIndex[key] = make(map[string]string)
					}
					remoteIndex[key][entry.Version] = entry.SHA256
				}
			}
		}
	}

	// 通过上游索引更新分类
	for i, item := range results {
		fullName := item.Name
		if item.Organization != "" {
			fullName = item.Organization + "::" + item.Name
		}
		ri := remoteIndex[fullName]

		if ri != nil {
			if remoteSHA, ok := ri[item.Version]; ok {
				results[i].RemoteSHA256 = remoteSHA
				if remoteSHA != "" && remoteSHA == item.SHA256 {
					results[i].Category = "already_exists"
					results[i].Selected = false
				} else if remoteSHA != "" && remoteSHA != item.SHA256 {
					results[i].Category = "conflict"
					results[i].Selected = false
				}
			}
			// 检查是否有远程版本比本地版本更新的推荐
			if results[i].Category == "need_publish" || results[i].Category == "version_optional" {
				bestRemote := ""
				for rv := range ri {
					if bestRemote == "" {
						bestRemote = rv
					} else {
						cmp, _ := semverutil.Compare(rv, bestRemote)
						if cmp > 0 {
							bestRemote = rv
						}
					}
				}
				if bestRemote != "" {
					cmp, err := semverutil.Compare(item.Version, bestRemote)
					if err == nil && cmp < 0 {
						results[i].Category = "version_optional"
					}
				}
			}
		}
	}

	return c.JSON(fiber.Map{
		"packages":     results,
		"publish_order": order,
		"total":        len(results),
	})
}

// collectDeps 收集依赖列表（去重）
func collectDeps(primary, test, script []depEntry) []depEntry {
	seen := make(map[string]bool)
	var deps []depEntry
	for _, list := range [][]depEntry{primary, test, script} {
		for _, d := range list {
			key := d.Name + "@" + d.Require
			if seen[key] {
				continue
			}
			seen[key] = true
			deps = append(deps, d)
		}
	}
	return deps
}

// parseDependencies 从 meta_data JSON 中解析所有依赖
// 元数据中依赖可能在顶层（老格式）或在 index 字段内（新格式）
func parseDependencies(metaDataStr string) []depEntry {
	if metaDataStr == "" {
		return nil
	}
	var md metaData
	if err := json.Unmarshal([]byte(metaDataStr), &md); err != nil {
		return nil
	}

	if len(md.Dependencies) > 0 || len(md.TestDependencies) > 0 || len(md.ScriptDependencies) > 0 {
		return collectDeps(md.Dependencies, md.TestDependencies, md.ScriptDependencies)
	}

	// 检查 index 字段内
	if md.Index != nil {
		if len(md.Index.Dependencies) > 0 || len(md.Index.TestDependencies) > 0 || len(md.Index.ScriptDependencies) > 0 {
			return collectDeps(md.Index.Dependencies, md.Index.TestDependencies, md.Index.ScriptDependencies)
		}
	}

	return nil
}

// findPackage 根据依赖项查找本地已发布的包
func (h *PublishPlanHandler) findPackage(dep depEntry) (*models.Package, error) {
	org, name := parseOrgName(dep.Name)
	if dep.Organization != "" {
		org = dep.Organization
	}

	// 查找本地所有版本
	var versions []models.Package
	query := h.engine.Where("name = ? AND deleted_at IS NULL", name)
	if org != "" {
		query = query.Where("organization = ?", org)
	} else {
		query = query.Where("(organization = ? OR organization = '' OR organization IS NULL)", org)
	}
	if err := query.Find(&versions); err != nil || len(versions) == 0 {
		return nil, fmt.Errorf("package not found: %s/%s", org, name)
	}

	// 按版本排序，找到最佳匹配
	var versionStrs []string
	verMap := make(map[string]*models.Package)
	for i := range versions {
		v := &versions[i]
		versionStrs = append(versionStrs, v.Version)
		verMap[v.Version] = v
	}
	semverutil.Sort(versionStrs)

	best, err := semverutil.BestMatch(versionStrs, dep.Require)
	if err != nil {
		// 没找到匹配，使用最新版本
		return verMap[versionStrs[len(versionStrs)-1]], nil
	}
	return verMap[best], nil
}
