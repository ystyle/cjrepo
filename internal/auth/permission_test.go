package auth

import (
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"xorm.io/xorm"

	"ystyle.top/go/cjrepo/internal/models"
)

func setupTestDB(t *testing.T) *xorm.Engine {
	engine, err := xorm.NewEngine("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	if err := engine.Sync2(
		new(models.Package),
		new(models.User),
		new(models.Team),
		new(models.TeamOrganization),
		new(models.TeamPackage),
		new(models.TeamMember),
	); err != nil {
		t.Fatalf("failed to sync tables: %v", err)
	}

	return engine
}

func seedData(engine *xorm.Engine) {
	// 用户
	engine.Insert(&models.User{ID: 1, Username: "publisher", Token: "tok1"})
	engine.Insert(&models.User{ID: 2, Username: "team_member", Token: "tok2"})
	engine.Insert(&models.User{ID: 3, Username: "nobody", Token: "tok3"})

	// 包
	engine.Insert(&models.Package{ID: 1, Organization: "", Name: "pub-pkg", PublisherID: 1})
	engine.Insert(&models.Package{ID: 2, Organization: "acme", Name: "org-pkg", PublisherID: 1})

	// 团队
	engine.Insert(&models.Team{ID: 1, Name: "dev-team", Permission: "write"})
	engine.Insert(&models.Team{ID: 2, Name: "readonly-team", Permission: "read"})

	// 团队成员
	engine.Insert(&models.TeamMember{ID: 1, TeamID: 1, UserID: 2})
	engine.Insert(&models.TeamMember{ID: 2, TeamID: 2, UserID: 2})

	// 团队-包关联
	engine.Insert(&models.TeamPackage{ID: 1, TeamID: 1, Organization: "", PackageName: "team-pkg"})
	engine.Insert(&models.TeamPackage{ID: 2, TeamID: 2, Organization: "", PackageName: "readonly-pkg"})
	engine.Insert(&models.TeamPackage{ID: 3, TeamID: 1, Organization: "acme", PackageName: "org-pkg"})

	// 组织
	engine.Insert(&models.Organization{ID: 1, Name: "acme", DisplayName: "Acme Corp"})

	// 团队-组织关联
	engine.Insert(&models.TeamOrganization{ID: 1, TeamID: 1, OrganizationID: int64Ptr(1)})
}

func int64Ptr(v int64) *int64 { return &v }

// ==== 路径1：个人发布（publisher）====

func Test_Publisher_OrgLess_Write(t *testing.T) {
	engine := setupTestDB(t)
	defer engine.Close()
	seedData(engine)

	checker := NewPermissionChecker(engine)
	// publisher 发布自己的无组织包新版本 → write ✅
	if !checker.CheckPermission(1, "", "pub-pkg", "write") {
		t.Error("publisher should have write on own org-less package")
	}
}

func Test_Publisher_OrgLess_Overwrite(t *testing.T) {
	engine := setupTestDB(t)
	defer engine.Close()
	seedData(engine)

	checker := NewPermissionChecker(engine)
	// publisher 覆盖自己的无组织包版本 → overwrite ❌（publisher 只有 write）
	if checker.CheckPermission(1, "", "pub-pkg", "overwrite") {
		t.Error("publisher should NOT have overwrite on own org-less package")
	}
}

func Test_Publisher_OrgLess_Read(t *testing.T) {
	engine := setupTestDB(t)
	defer engine.Close()
	seedData(engine)

	checker := NewPermissionChecker(engine)
	// publisher 下载 → read ✅（publisher 的 write >= read）
	if !checker.CheckPermission(1, "", "pub-pkg", "read") {
		t.Error("publisher should have read on own org-less package")
	}
}

func Test_NonPublisher_OrgLess_Write(t *testing.T) {
	engine := setupTestDB(t)
	defer engine.Close()
	seedData(engine)

	checker := NewPermissionChecker(engine)
	// 非 publisher 请求 write → ❌（nobody 无团队关联）
	if checker.CheckPermission(3, "", "pub-pkg", "write") {
		t.Error("non-publisher should NOT have write on someone else's package")
	}
}

func Test_Publisher_OrgPackage_Write(t *testing.T) {
	engine := setupTestDB(t)
	defer engine.Close()
	seedData(engine)

	checker := NewPermissionChecker(engine)
	// publisher 对有组织的包请求 write → ❌（publisher 仅对无组织包有效）
	if checker.CheckPermission(1, "acme", "org-pkg", "write") {
		t.Error("publisher should NOT auto-have write on org-associated package")
	}
}

// ==== 路径2：团队管理 ====

func Test_Team_Package_Write_Success(t *testing.T) {
	engine := setupTestDB(t)
	defer engine.Close()
	seedData(engine)

	checker := NewPermissionChecker(engine)
	// dev-team（write）关联了 team-pkg → write ✅
	if !checker.CheckPermission(2, "", "team-pkg", "write") {
		t.Error("team member should have write on team-associated package")
	}
}

func Test_Team_Package_Overwrite_Fail(t *testing.T) {
	engine := setupTestDB(t)
	defer engine.Close()
	seedData(engine)

	checker := NewPermissionChecker(engine)
	// dev-team 只有 write，请求 overwrite → ❌
	if checker.CheckPermission(2, "", "team-pkg", "overwrite") {
		t.Error("team with write should NOT have overwrite on package")
	}
}

func Test_Team_Package_Readonly(t *testing.T) {
	engine := setupTestDB(t)
	defer engine.Close()
	seedData(engine)

	checker := NewPermissionChecker(engine)
	// readonly-team 只有 read，请求 write → ❌
	if checker.CheckPermission(2, "", "readonly-pkg", "write") {
		t.Error("team with read should NOT have write on package")
	}
	// readonly-team read → ✅
	if !checker.CheckPermission(2, "", "readonly-pkg", "read") {
		t.Error("team with read should have read on package")
	}
}

func Test_Team_Org_Write_Success(t *testing.T) {
	engine := setupTestDB(t)
	defer engine.Close()
	seedData(engine)

	checker := NewPermissionChecker(engine)
	// dev-team 关联了 acme 组织 → 对该组织的包有 write
	if !checker.CheckPermission(2, "acme", "org-pkg", "write") {
		t.Error("team member should have write on org-associated package via team-org link")
	}
}

func Test_Team_Org_Write_NoOrgLink(t *testing.T) {
	engine := setupTestDB(t)
	defer engine.Close()
	seedData(engine)

	checker := NewPermissionChecker(engine)
	// 没有团队关联 other-org → ❌
	if checker.CheckPermission(2, "other-org", "any-pkg", "write") {
		t.Error("team member should NOT have write on unrelated org")
	}
}

func Test_NoTeam_User(t *testing.T) {
	engine := setupTestDB(t)
	defer engine.Close()
	seedData(engine)

	checker := NewPermissionChecker(engine)
	// nobody 没有团队也不是 publisher → ❌
	if checker.CheckPermission(3, "", "pub-pkg", "write") {
		t.Error("user with no team should NOT have any permission")
	}
}
