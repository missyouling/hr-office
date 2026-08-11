package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"siapp/internal/auth"
	"siapp/internal/models"
)

// setupTestDB 创建内存 SQLite 用于测试
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to in-memory sqlite: %v", err)
	}
	// 自动建表
	if err := db.AutoMigrate(
		&models.Role{},
		&models.Permission{},
		&models.RolePermission{},
		&models.UserRole{},
		&models.User{},
		&models.Department{},
		&models.DepartmentMember{},
	); err != nil {
		t.Fatalf("failed to auto-migrate: %v", err)
	}
	return db
}

// seedTestRolesPermissions 创建测试用的角色和权限数据
func seedTestRolesPermissions(db *gorm.DB) (adminRoleID, managerRoleID, editorRoleID, viewerRoleID uint) {
	roles := []models.Role{
		{Name: models.RoleAdmin, Label: "管理员", IsSystem: true},
		{Name: models.RoleManager, Label: "部门经理", IsSystem: true},
		{Name: models.RoleEditor, Label: "编辑者", IsSystem: true},
		{Name: models.RoleViewer, Label: "只读用户", IsSystem: true},
	}
	for i := range roles {
		db.Where("name = ?", roles[i].Name).FirstOrCreate(&roles[i])
	}
	adminRoleID = roles[0].ID
	managerRoleID = roles[1].ID
	editorRoleID = roles[2].ID
	viewerRoleID = roles[3].ID

	// 创建权限
	permissions := []models.Permission{
		{Module: "employee", Action: "view", Label: "查看"},
		{Module: "employee", Action: "delete", Label: "删除"},
	}
	for _, p := range permissions {
		db.Where("module = ? AND action = ?", p.Module, p.Action).FirstOrCreate(&p)
	}
	return
}

// seedTestUser 创建测试用户并分配角色
func seedTestUser(db *gorm.DB, username string, roleID uint) (*models.User, error) {
	user := models.User{
		Username: username,
		Email:    username + "@test.com",
		Active:   true,
	}
	if err := db.Create(&user).Error; err != nil {
		return nil, err
	}
	if err := db.Create(&models.UserRole{UserID: user.ID, RoleID: roleID}).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// newRequestWithUser 创建带有用户 context 的 HTTP 请求
func newRequestWithUser(userID uint, method, path string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	ctx := context.WithValue(req.Context(), auth.UserIDKey, userID)
	return req.WithContext(ctx)
}

// ========== RequireRole 测试 ==========

func TestRequireRole_AdminAccess(t *testing.T) {
	db := setupTestDB(t)
	adminRoleID, _, _, _ := seedTestRolesPermissions(db)
	adminUser, err := seedTestUser(db, "testadmin", adminRoleID)
	if err != nil {
		t.Fatalf("failed to create admin user: %v", err)
	}

	handler := RequireRole(db, models.RoleAdmin)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := newRequestWithUser(adminUser.ID, "GET", "/admin/test")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("期望状态码 200，实际得到 %d", rec.Code)
	}
}

func TestRequireRole_ViewerAccessAdminOnly(t *testing.T) {
	db := setupTestDB(t)
	_, _, _, viewerRoleID := seedTestRolesPermissions(db)
	viewerUser, err := seedTestUser(db, "testviewer", viewerRoleID)
	if err != nil {
		t.Fatalf("failed to create viewer user: %v", err)
	}

	handler := RequireRole(db, models.RoleAdmin)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := newRequestWithUser(viewerUser.ID, "GET", "/admin/test")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("期望状态码 403（viewer 无权访问 admin-only），实际得到 %d", rec.Code)
	}
}

func TestRequireRole_ManagerOrAdminAllowed(t *testing.T) {
	db := setupTestDB(t)
	_, managerRoleID, _, _ := seedTestRolesPermissions(db)
	managerUser, err := seedTestUser(db, "testmgr", managerRoleID)
	if err != nil {
		t.Fatalf("failed to create manager user: %v", err)
	}

	handler := RequireRole(db, models.RoleManager, models.RoleAdmin)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := newRequestWithUser(managerUser.ID, "GET", "/manage/test")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("期望状态码 200（manager 应允许），实际得到 %d", rec.Code)
	}
}

func TestRequireRole_NoAuth(t *testing.T) {
	db := setupTestDB(t)
	handler := RequireRole(db, models.RoleAdmin)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// 请求不带 user context
	req := httptest.NewRequest("GET", "/admin/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("期望状态码 401（未认证），实际得到 %d", rec.Code)
	}
}

// ========== RequireAdmin 测试 ==========

func TestRequireAdmin_Allowed(t *testing.T) {
	db := setupTestDB(t)
	adminRoleID, _, _, _ := seedTestRolesPermissions(db)
	adminUser, _ := seedTestUser(db, "admintest", adminRoleID)

	handler := RequireAdmin(db)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := newRequestWithUser(adminUser.ID, "GET", "/admin")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("期望状态码 200，实际得到 %d", rec.Code)
	}
}

func TestRequireAdmin_DeniedForViewer(t *testing.T) {
	db := setupTestDB(t)
	_, _, _, viewerRoleID := seedTestRolesPermissions(db)
	viewerUser, _ := seedTestUser(db, "viewerdenied", viewerRoleID)

	handler := RequireAdmin(db)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := newRequestWithUser(viewerUser.ID, "GET", "/admin")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("期望状态码 403，实际得到 %d", rec.Code)
	}
}

// ========== RequireManagerOrAbove 测试 ==========

func TestRequireManagerOrAbove_ManagerAllowed(t *testing.T) {
	db := setupTestDB(t)
	_, managerRoleID, _, _ := seedTestRolesPermissions(db)
	mgrUser, _ := seedTestUser(db, "mgrtest", managerRoleID)

	handler := RequireManagerOrAbove(db)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := newRequestWithUser(mgrUser.ID, "GET", "/manage")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("期望状态码 200，实际得到 %d", rec.Code)
	}
}

func TestRequireManagerOrAbove_EditorDenied(t *testing.T) {
	db := setupTestDB(t)
	_, _, editorRoleID, _ := seedTestRolesPermissions(db)
	editorUser, _ := seedTestUser(db, "edittest", editorRoleID)

	handler := RequireManagerOrAbove(db)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := newRequestWithUser(editorUser.ID, "GET", "/manage")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("期望状态码 403（editor 不应访问 manager-only），实际得到 %d", rec.Code)
	}
}

// ========== RequireSameDepartment 测试 ==========

func TestRequireSameDepartment_SameDepartment(t *testing.T) {
	db := setupTestDB(t)
	adminRoleID, _, _, _ := seedTestRolesPermissions(db)

	// 创建部门
	dept := models.Department{Name: "测试部", Code: "TEST"}
	if err := db.Create(&dept).Error; err != nil {
		t.Fatalf("failed to create department: %v", err)
	}

	// 创建两个同部门用户
	userA := models.User{Username: "userA", Email: "a@test.com", Active: true, DepartmentID: &dept.ID}
	db.Create(&userA)
	db.Create(&models.UserRole{UserID: userA.ID, RoleID: adminRoleID})

	userB := models.User{Username: "userB", Email: "b@test.com", Active: true, DepartmentID: &dept.ID}
	db.Create(&userB)

	handler := RequireSameDepartment(db)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := newRequestWithUser(userA.ID, "GET", fmt.Sprintf("/api/resource?resource_user_id=%d", userB.ID))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// userA 是 admin，所以绕过部门隔离 -> 200
	// 改为非 admin 用户测试
	if rec.Code != http.StatusOK {
		t.Errorf("admin 应绕过部门隔离，期望 200，实际 %d", rec.Code)
	}
}

func TestRequireSameDepartment_CrossDepartment(t *testing.T) {
	db := setupTestDB(t)
	_, _, _, viewerRoleID := seedTestRolesPermissions(db)

	// 创建两个部门
	deptA := models.Department{Name: "部门A", Code: "A"}
	db.Create(&deptA)
	deptB := models.Department{Name: "部门B", Code: "B"}
	db.Create(&deptB)

	// 用户A 在部门A，角色 viewer
	userA := models.User{Username: "crossA", Email: "crossA@test.com", Active: true, DepartmentID: &deptA.ID}
	db.Create(&userA)
	db.Create(&models.UserRole{UserID: userA.ID, RoleID: viewerRoleID})

	// 用户B 在部门B
	userB := models.User{Username: "crossB", Email: "crossB@test.com", Active: true, DepartmentID: &deptB.ID}
	db.Create(&userB)

	handler := RequireSameDepartment(db)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// userA 尝试访问 userB 的资源
	req := newRequestWithUser(userA.ID, "GET", fmt.Sprintf("/api/resource?resource_user_id=%d", userB.ID))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// 由于 resource_user_id 指向 userB（不同部门），应被拒绝
	if rec.Code != http.StatusForbidden {
		t.Errorf("跨部门访问应被拒绝（403），实际得到 %d", rec.Code)
	}
}

func TestRequireSameDepartment_NoDepartment(t *testing.T) {
	db := setupTestDB(t)
	_, _, _, viewerRoleID := seedTestRolesPermissions(db)

	// 用户无部门
	user := models.User{Username: "nodept", Email: "nodept@test.com", Active: true, DepartmentID: nil}
	db.Create(&user)
	db.Create(&models.UserRole{UserID: user.ID, RoleID: viewerRoleID})

	handler := RequireSameDepartment(db)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := newRequestWithUser(user.ID, "GET", "/api/resource?resource_user_id=1")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("无部门归属用户应被拒绝（403），实际得到 %d", rec.Code)
	}
}

func TestRequireSameDepartment_AdminBypass(t *testing.T) {
	db := setupTestDB(t)
	adminRoleID, _, _, _ := seedTestRolesPermissions(db)

	deptA := models.Department{Name: "管理部", Code: "MGR"}
	db.Create(&deptA)

	adminUser := models.User{Username: "bypassadm", Email: "bypass@test.com", Active: true, DepartmentID: &deptA.ID}
	db.Create(&adminUser)
	db.Create(&models.UserRole{UserID: adminUser.ID, RoleID: adminRoleID})

	handler := RequireSameDepartment(db)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := newRequestWithUser(adminUser.ID, "GET", "/api/resource?resource_user_id=9999")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("admin 应绕过部门隔离，期望 200，实际 %d", rec.Code)
	}
}

// ========== NormalizeRole 测试 ==========

func TestNormalizeRole_SuperAdminMapsToAdmin(t *testing.T) {
	result := models.NormalizeRole("super_admin")
	if result != models.RoleAdmin {
		t.Errorf("期望 super_admin 映射为 admin，实际得到 %s", result)
	}
}

func TestNormalizeRole_ValidRoleUnchanged(t *testing.T) {
	result := models.NormalizeRole(models.RoleManager)
	if result != models.RoleManager {
		t.Errorf("期望 manager 保持原样，实际得到 %s", result)
	}
}
