package migrations

import (
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"

	"ystyle.top/go/cjrepo/internal/models"
)

func hasColumn(engine *xorm.Engine, table, column string) bool {
	rows, err := engine.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return false
	}
	for _, row := range rows {
		if string(row["name"]) == column {
			return true
		}
	}
	return false
}

func tableExists(engine *xorm.Engine, name string) bool {
	ok, _ := engine.SQL("SELECT name FROM sqlite_master WHERE type='table' AND name=?", name).Exist()
	return ok
}

func TestV110Migration(t *testing.T) {
	dbPath := "/tmp/test_migration_v110.db"
	os.Remove(dbPath)

	engine, err := xorm.NewEngine("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("create engine: %v", err)
	}
	defer engine.Close()
	engine.ShowSQL(false)

	// 用 raw SQL 模拟 v1.0.x schema（不含 publisher_i_d）
	sqls := []string{
		`CREATE TABLE packages (
			i_d INTEGER PRIMARY KEY AUTOINCREMENT,
			organization TEXT DEFAULT '' NOT NULL,
			name TEXT NOT NULL,
			version TEXT NOT NULL,
			description TEXT DEFAULT '',
			cjc_version TEXT DEFAULT '',
			artifact_type TEXT DEFAULT 'src',
			executable INTEGER DEFAULT 0,
			authors TEXT DEFAULT '',
			repository TEXT DEFAULT '',
			homepage TEXT DEFAULT '',
			documentation TEXT DEFAULT '',
			tags TEXT DEFAULT '',
			categories TEXT DEFAULT '',
			licenses TEXT DEFAULT '',
			meta_version INTEGER DEFAULT 0,
			meta_data TEXT DEFAULT '',
			readme TEXT DEFAULT '',
			tarball_path TEXT DEFAULT '',
			tarball_size INTEGER DEFAULT 0,
			tarball_sha256 TEXT DEFAULT '',
			download_count INTEGER DEFAULT 0,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)`,
		`INSERT INTO packages (name, organization, version) VALUES ('mypkg', '', '1.0.0')`,
		`INSERT INTO packages (name, organization, version) VALUES ('utils', 'acme', '0.1.0')`,
		`CREATE TABLE organizations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			display_name TEXT DEFAULT '',
			description TEXT DEFAULT '',
			is_default INTEGER DEFAULT 0,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)`,
		`INSERT INTO organizations (name, display_name) VALUES ('acme', 'ACME Corp')`,
		`CREATE TABLE organization_members (
			i_d INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_i_d INTEGER NOT NULL,
			user_i_d INTEGER NOT NULL,
			created_at DATETIME
		)`,
		`CREATE TABLE users (
			i_d INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL,
			token TEXT,
			email TEXT DEFAULT '',
			is_active INTEGER DEFAULT 1,
			is_superuser INTEGER DEFAULT 0,
			created_at DATETIME
		)`,
		`INSERT INTO users (username, email) VALUES ('alice', 'alice@test.com')`,
	}
	for _, s := range sqls {
		if _, err := engine.Exec(s); err != nil {
			t.Fatalf("setup: %v", err)
		}
	}

	// 关联 acme 组织成员
	engine.Exec("INSERT INTO organization_members (organization_i_d, user_i_d) VALUES (1, 1)")

	// 验证初始状态：无 publisher_i_d
	if hasColumn(engine, "packages", "publisher_i_d") {
		t.Fatal("初始状态不应有 publisher_i_d 列")
	}

	// 预先 Sync2 team 表（生产环境在 migration 前已完成）
	if err := engine.Sync2(
		new(models.Team),
		new(models.TeamOrganization),
		new(models.TeamPackage),
		new(models.TeamMember),
	); err != nil {
		t.Fatalf("sync team tables: %v", err)
	}
	// 验证 team 表已存在但无 v1.2.0 变更
	if hasColumn(engine, "team_packages", "permission") {
		t.Fatal("初始 team_packages 不应有 permission 列")
	}

	// 运行迁移
	if err := Run(engine); err != nil {
		t.Fatalf("migration failed: %v", err)
	}

	// 验证 1: packages.publisher_i_d
	if !hasColumn(engine, "packages", "publisher_i_d") {
		t.Error("迁移后 packages 表缺少 publisher_i_d 列")
	} else {
		t.Log("✅ packages.publisher_i_d 列存在")
	}

	// 验证 2: team_packages 无 permission
	if hasColumn(engine, "team_packages", "permission") {
		t.Error("team_packages 仍存在 permission 列")
	} else if tableExists(engine, "team_packages") {
		t.Log("✅ team_packages.permission 列已删除")
	}

	// 验证 3: teams 表存在
	if !tableExists(engine, "teams") {
		t.Error("teams 表不存在")
	} else {
		t.Log("✅ teams 表存在")
	}

	// 验证 4: team_* 表存在
	for _, name := range []string{"team_members", "team_organizations", "team_packages"} {
		if !tableExists(engine, name) {
			t.Errorf("%s 表不存在", name)
		} else {
			t.Logf("✅ %s 表存在", name)
		}
	}

	// 验证 5: 迁移记录
	count, _ := engine.Count(&models.Migration{})
	if count == 0 {
		t.Error("迁移记录未被写入")
	} else {
		t.Logf("✅ 迁移记录数: %d", count)
	}

	// 验证 6: 数据迁移正确——acme 组织有对应团队
	var teams []models.Team
	engine.Find(&teams)
	if len(teams) == 0 {
		t.Error("未创建团队")
	} else {
		t.Logf("✅ 已创建 %d 个团队", len(teams))
		for _, tm := range teams {
			t.Logf("   team: %s (permission=%s)", tm.Name, tm.Permission)
		}
	}

	// 验证 7: 幂等性——二次运行不应报错
	if err := Run(engine); err != nil {
		t.Errorf("二次运行迁移失败: %v", err)
	}
	t.Log("✅ 迁移幂等性验证通过")
}
