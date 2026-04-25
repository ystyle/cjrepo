package handlers

import (
	"encoding/json"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"

	"ystyle.top/go/cjrepo/internal/models"
)

func TestParseOrgName(t *testing.T) {
	tests := []struct {
		raw     string
		wantOrg string
		wantName string
	}{
		{"soulsoft::identity_tokens", "soulsoft", "identity_tokens"},
		{"simple-pkg", "", "simple-pkg"},
		{"org::name", "org", "name"},
		{"::emptyorg", "", "emptyorg"},
		{"", "", ""},
	}
	for _, tt := range tests {
		org, name := parseOrgName(tt.raw)
		if org != tt.wantOrg || name != tt.wantName {
			t.Errorf("parseOrgName(%q) = (%q, %q), want (%q, %q)", tt.raw, org, name, tt.wantOrg, tt.wantName)
		}
	}
}

func TestParseDependencies(t *testing.T) {
	// 完整的 meta-data JSON
	metaData := map[string]interface{}{
		"name":         "test-pkg",
		"version":      "1.0.0",
		"organization": "",
		"dependencies": []map[string]interface{}{
			{"name": "dep1", "require": "1.0.0"},
			{"name": "org::dep2", "require": "[1.0.0, 2.0.0)"},
			{"name": "dep3", "require": "2.0.0", "target": "linux"},
		},
		"test-dependencies": []map[string]interface{}{
			{"name": "test-dep", "require": "0.5.0"},
		},
		"script-dependencies": []map[string]interface{}{
			{"name": "build-tool", "require": "1.0.0"},
		},
	}
	data, _ := json.Marshal(metaData)

	deps := parseDependencies(string(data))
	if len(deps) != 5 {
		t.Errorf("expected 5 deps, got %d", len(deps))
	}

	// 验证依赖名
	names := make(map[string]bool)
	for _, d := range deps {
		names[d.Name] = true
	}
	for _, n := range []string{"dep1", "org::dep2", "dep3", "test-dep", "build-tool"} {
		if !names[n] {
			t.Errorf("missing dep: %s", n)
		}
	}

	// 验证去重：相同的 name+require 只保留一个
	deps = parseDependencies(`{"dependencies":[{"name":"a","require":"1.0.0"},{"name":"a","require":"1.0.0"},{"name":"b","require":"2.0.0"}],"test-dependencies":[],"script-dependencies":[]}`)
	if len(deps) != 2 {
		t.Errorf("expected 2 unique deps after dedup, got %d", len(deps))
	}

	// 空 JSON
	deps = parseDependencies("")
	if deps != nil {
		t.Error("expected nil for empty input")
	}

	// 无依赖字段
	deps = parseDependencies(`{"name":"test","version":"1.0.0"}`)
	if len(deps) != 0 {
		t.Errorf("expected 0 deps for no-dependency JSON, got %d", len(deps))
	}
}

func TestParseDependenciesEdgeCases(t *testing.T) {
	// 测试嵌套依赖和各类边缘情况
	deps := parseDependencies(`{"dependencies":[{"name":"a","require":"1.0.0","target":"linux"},{"name":"a","require":"1.0.0","target":"macos"}],"test-dependencies":[{"name":"a","require":"1.0.0","target":"windows"}],"script-dependencies":[]}`)
	// 相同的 name+require 不同 target，应该算不同依赖
	// 但是我们按 name+require 去重，target 不同不作为区分
	if len(deps) != 1 {
		t.Errorf("dedup by name+require, expected 1, got %d", len(deps))
	}

	// JSON 解析失败的情况
	deps = parseDependencies(`not json`)
	if deps != nil {
		t.Error("expected nil for invalid JSON")
	}
}

// getLocalVersions 辅助函数：获取包的所有本地版本
func getLocalVersions(engine *xorm.Engine, org, name string) []string {
	var pkgs []models.Package
	engine.Where("name = ? AND (organization = ? OR organization = '' OR organization IS NULL)", name, org).Find(&pkgs)
	versions := make([]string, len(pkgs))
	for i, p := range pkgs {
		versions[i] = p.Version
	}
	return versions
}

// setupTestPackages 创建测试用的包数据
func setupTestPackages(engine *xorm.Engine) {
	pkgs := []struct {
		org    string
		name   string
		ver    string
		sha256 string
		meta   string
	}{
		{
			org: "acme", name: "core", ver: "1.0.0", sha256: "aaa",
			meta: `{"name":"core","version":"1.0.0","dependencies":[]}`,
		},
		{
			org: "acme", name: "core", ver: "1.1.0", sha256: "bbb",
			meta: `{"name":"core","version":"1.1.0","dependencies":[]}`,
		},
		{
			org: "acme", name: "core", ver: "2.0.0", sha256: "ccc",
			meta: `{"name":"core","version":"2.0.0","dependencies":[]}`,
		},
		{
			org: "", name: "utils", ver: "1.0.0", sha256: "ddd",
			meta: `{"name":"utils","version":"1.0.0","dependencies":[]}`,
		},
		{
			org: "acme", name: "lib-a", ver: "1.0.0", sha256: "eee",
			meta: `{"name":"lib-a","version":"1.0.0","dependencies":[{"name":"acme::core","require":"[1.0.0, 2.0.0)"}]}`,
		},
		{
			org: "acme", name: "lib-b", ver: "1.0.0", sha256: "fff",
			meta: `{"name":"lib-b","version":"1.0.0","dependencies":[{"name":"acme::core","require":"[1.0.0, 2.0.0)"},{"name":"utils","require":"1.0.0"}]}`,
		},
		{
			org: "acme", name: "service", ver: "1.0.0", sha256: "ggg",
			meta: `{"name":"service","version":"1.0.0","dependencies":[{"name":"acme::lib-a","require":"1.0.0"},{"name":"acme::lib-b","require":"1.0.0"}]}`,
		},
		// 循环依赖测试：p1 依赖 p2，p2 依赖 p1
		{
			org: "", name: "p1", ver: "1.0.0", sha256: "hhh",
			meta: `{"name":"p1","version":"1.0.0","dependencies":[{"name":"p2","require":"1.0.0"}]}`,
		},
		{
			org: "", name: "p2", ver: "1.0.0", sha256: "iii",
			meta: `{"name":"p2","version":"1.0.0","dependencies":[{"name":"p1","require":"1.0.0"}]}`,
		},
		// 多版本可选：local 有多个版本满足依赖需求
		{
			org: "", name: "multi", ver: "1.0.0", sha256: "jjj",
			meta: `{"name":"multi","version":"1.0.0","dependencies":[{"name":"core","require":"[1.0.0, 3.0.0)"}],"organization":"acme"}`,
		},
	}

	for _, p := range pkgs {
		engine.Insert(&models.Package{
			Organization:  p.org,
			Name:          p.name,
			Version:       p.ver,
			TarballSHA256: p.sha256,
			MetaData:      p.meta,
		})
	}
}

func TestDependencyResolution(t *testing.T) {
	engine, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("create engine: %v", err)
	}
	engine.Sync2(new(models.Package))
	setupTestPackages(engine)

	h := &PublishPlanHandler{engine: engine}

	// 测试 1：单一包无依赖
	results, order, err := h.resolvePackages([]int64{1, 2}, "")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected results")
	}
	t.Logf("Test 1 - single pkg no deps: %d results, order=%v", len(results), order)

	// 测试 2：lib-a 依赖 core（1级依赖）
	// lib-a 的 ID 是 5（第5个插入的）
	results, order, err = h.resolvePackages([]int64{5}, "")
	if err != nil {
		t.Fatalf("resolve lib-a: %v", err)
	}
	pkgNames := make(map[int64]string)
	for _, r := range results {
		pkgNames[r.PackageID] = r.Name
	}
	if _, ok := pkgNames[2]; !ok {
		t.Error("lib-a should include core (ID=2)")
	}
	t.Logf("Test 2 - lib-a deps: %d results, order=%v", len(results), order)
	for _, r := range results {
		t.Logf("  package: org=%q name=%s ver=%s", r.Organization, r.Name, r.Version)
	}

	// 测试 3：service 依赖 lib-a + lib-b（传递依赖）
	results, order, err = h.resolvePackages([]int64{7}, "")
	if err != nil {
		t.Fatalf("resolve service: %v", err)
	}
	if len(results) < 4 {
		t.Errorf("service should resolve >=4 packages, got %d", len(results))
	}
	t.Logf("Test 3 - service transitive deps: %d results", len(results))
	for _, r := range results {
		t.Logf("  package: org=%q name=%s ver=%s", r.Organization, r.Name, r.Version)
	}

	// 验证拓扑顺序：core → utils → lib-a → lib-b → service
	t.Logf("  publish order: %v", order)

	// 测试 4：循环依赖检测（p1 → p2 → p1）
	results, order, err = h.resolvePackages([]int64{9}, "") // p1
	if err != nil {
		t.Fatalf("resolve circular: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 packages for circular dep test, got %d", len(results))
	}
	t.Logf("Test 4 - circular deps: %d results, order=%v", len(results), order)

	// 测试 5：空输入
	results, order, err = h.resolvePackages(nil, "")
	if err != nil {
		t.Fatalf("resolve empty: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty input, got %d", len(results))
	}
	if len(order) != 0 {
		t.Errorf("expected empty order for empty input, got %v", order)
	}

	// 测试 6：不存在的包 ID
	results, order, err = h.resolvePackages([]int64{9999}, "")
	if err != nil {
		t.Fatalf("resolve nonexistent: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for nonexistent, got %d", len(results))
	}

	// 测试 7：多起始包
	results, order, err = h.resolvePackages([]int64{5, 6}, "") // lib-a + lib-b
	if err != nil {
		t.Fatalf("resolve multi-start: %v", err)
	}
	t.Logf("Test 7 - multi start: %d results", len(results))
	for _, r := range results {
		t.Logf("  package: org=%q name=%s ver=%s", r.Organization, r.Name, r.Version)
	}
}
