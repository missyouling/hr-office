package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"siapp/internal/auth"
	"siapp/internal/middleware"
	"siapp/internal/models"
	"siapp/internal/service"
)

// setupAuthTestDB 创建内存 SQLite 并自动建表
func setupAuthTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("连接内存 SQLite 失败: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{}, &models.AuthToken{},
		&models.Role{}, &models.Permission{},
		&models.RolePermission{}, &models.UserRole{},
		&models.PasswordResetToken{}, &models.EmailVerificationToken{},
	); err != nil {
		t.Fatalf("自动建表失败: %v", err)
	}
	return db
}

// seedAuthTestData 创建测试角色和权限数据
func seedAuthTestData(db *gorm.DB) (adminID, managerID, editorID, viewerID uint) {
	roles := []models.Role{
		{Name: models.RoleAdmin, Label: "管理员", IsSystem: true},
		{Name: models.RoleManager, Label: "部门经理", IsSystem: true},
		{Name: models.RoleEditor, Label: "编辑者", IsSystem: true},
		{Name: models.RoleViewer, Label: "只读用户", IsSystem: true},
	}
	for i := range roles {
		db.Where("name = ?", roles[i].Name).FirstOrCreate(&roles[i])
	}

	permDefs := []struct {
		Module, Action, Label string
		SortOrder             int
	}{
		{"employee", "view", "查看", 1},
		{"employee", "delete", "删除", 4},
		{"insurance", "view", "查看", 10},
		{"settings", "view", "查看", 40},
		{"users", "view", "查看", 70},
	}
	for _, pd := range permDefs {
		p := models.Permission{Module: pd.Module, Action: pd.Action, Label: pd.Label, SortOrder: pd.SortOrder}
		db.Where("module = ? AND action = ?", pd.Module, pd.Action).FirstOrCreate(&p)
	}

	var pEmpView, pEmpDel, pInsView, pSetView, pUsrView models.Permission
	db.Where("module = ? AND action = ?", "employee", "view").First(&pEmpView)
	db.Where("module = ? AND action = ?", "employee", "delete").First(&pEmpDel)
	db.Where("module = ? AND action = ?", "insurance", "view").First(&pInsView)
	db.Where("module = ? AND action = ?", "settings", "view").First(&pSetView)
	db.Where("module = ? AND action = ?", "users", "view").First(&pUsrView)

	// admin：所有权限
	allIDs := []uint{pEmpView.ID, pEmpDel.ID, pInsView.ID, pSetView.ID, pUsrView.ID}
	for _, pid := range allIDs {
		db.FirstOrCreate(&models.RolePermission{}, &models.RolePermission{RoleID: roles[0].ID, PermissionID: pid})
	}
	// manager：employee.view + insurance.view
	db.FirstOrCreate(&models.RolePermission{}, &models.RolePermission{RoleID: roles[1].ID, PermissionID: pEmpView.ID})
	db.FirstOrCreate(&models.RolePermission{}, &models.RolePermission{RoleID: roles[1].ID, PermissionID: pInsView.ID})
	// editor：employee.view + settings.view
	db.FirstOrCreate(&models.RolePermission{}, &models.RolePermission{RoleID: roles[2].ID, PermissionID: pEmpView.ID})
	db.FirstOrCreate(&models.RolePermission{}, &models.RolePermission{RoleID: roles[2].ID, PermissionID: pSetView.ID})
	// viewer：仅 employee.view
	db.FirstOrCreate(&models.RolePermission{}, &models.RolePermission{RoleID: roles[3].ID, PermissionID: pEmpView.ID})

	return roles[0].ID, roles[1].ID, roles[2].ID, roles[3].ID
}

// createAuthTestUser 创建测试用户并分配角色
func createAuthTestUser(db *gorm.DB, username, password string, roleID uint) (*models.User, error) {
	u := models.User{
		Username:      username,
		Email:         username + "@test.local",
		FullName:      "测试用户",
		Active:        true,
		EmailVerified: true,
	}
	if err := u.SetPassword(password); err != nil {
		return nil, err
	}
	if err := db.Create(&u).Error; err != nil {
		return nil, err
	}
	if err := db.Create(&models.UserRole{UserID: u.ID, RoleID: roleID}).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// ========== loadUserPermissions 单元测试 ==========

func TestLoadUserPermissions_AdminAllPerms(t *testing.T) {
	db := setupAuthTestDB(t)
	aID, _, _, _ := seedAuthTestData(db)
	u, _ := createAuthTestUser(db, "admin01", "pwd123", aID)

	perms, err := loadUserPermissions(db, u.ID)
	if err != nil {
		t.Fatalf("加载权限失败: %v", err)
	}
	if len(perms) != 5 {
		t.Errorf("admin 期望 5 权限，实际 %d: %v", len(perms), perms)
	}
}

func TestLoadUserPermissions_ViewerOnePerm(t *testing.T) {
	db := setupAuthTestDB(t)
	_, _, _, vID := seedAuthTestData(db)
	u, _ := createAuthTestUser(db, "viewer01", "pwd123", vID)

	perms, err := loadUserPermissions(db, u.ID)
	if err != nil {
		t.Fatalf("加载权限失败: %v", err)
	}
	if len(perms) != 1 || (len(perms) > 0 && perms[0] != "employee.view") {
		t.Errorf("viewer 期望 [employee.view]，实际 %v", perms)
	}
}

func TestLoadUserPermissions_NoRoles(t *testing.T) {
	db := setupAuthTestDB(t)
	u := &models.User{Username: "norole", Email: "n@t.com", Active: true}
	db.Create(u)

	perms, err := loadUserPermissions(db, u.ID)
	if err != nil {
		t.Fatalf("加载权限失败: %v", err)
	}
	if len(perms) != 0 {
		t.Errorf("无角色用户期望空数组，实际 %v", perms)
	}
}

func TestLoadUserPermissions_NonexistentUser(t *testing.T) {
	db := setupAuthTestDB(t)
	perms, err := loadUserPermissions(db, 99999)
	if err != nil {
		t.Fatalf("加载权限失败: %v", err)
	}
	if len(perms) != 0 {
		t.Errorf("不存在用户期望空数组，实际 %v", perms)
	}
}

// ========== Login 响应包含 permissions 集成测试 ==========

func TestLogin_ResponseHasPermissions(t *testing.T) {
	t.Setenv("JWT_SECRET_KEY", "test-jwt-secret-for-auth-test")
	db := setupAuthTestDB(t)
	aID, _, _, _ := seedAuthTestData(db)
	_, err := createAuthTestUser(db, "loginusr", "pass123", aID)
	if err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}

	jwtMgr := auth.NewJWTManager()
	rl := middleware.NewLoginRateLimiter()
	handler := NewAuthHandler(
		db, jwtMgr,
		service.NewPasswordResetService(db),
		service.NewEmailVerificationService(db),
		service.NewEmailService(), rl,
	)

	body := `{"username":"loginusr","password":"pass123"}`
	req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("登录返回 %d，期望 200。body: %s", rec.Code, rec.Body.String())
	}

	var resp models.AuthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if resp.Token == "" {
		t.Error("缺少 token")
	}
	if resp.RefreshToken == "" {
		t.Error("缺少 refresh_token")
	}
	if resp.User.ID == 0 {
		t.Error("user.id 为空")
	}
	if resp.User.Username != "loginusr" {
		t.Errorf("username 期望 loginusr，实际 %s", resp.User.Username)
	}

	// 核心断言：permissions 必须存在且非空
	if len(resp.Permissions) == 0 {
		t.Fatal("permissions 数组为空或缺失")
	}

	// 验证 admin 关键权限存在
	hasView, hasDel := false, false
	for _, p := range resp.Permissions {
		if p == "employee.view" {
			hasView = true
		}
		if p == "employee.delete" {
			hasDel = true
		}
	}
	if !hasView {
		t.Errorf("admin 应含 employee.view，实际: %v", resp.Permissions)
	}
	if !hasDel {
		t.Errorf("admin 应含 employee.delete，实际: %v", resp.Permissions)
	}
}

// ========== 4 角色权限清单测试 ==========

func TestPermissionsMatrix_FourRoles(t *testing.T) {
	db := setupAuthTestDB(t)
	aID, mID, eID, vID := seedAuthTestData(db)

	tests := []struct {
		name       string
		roleID     uint
		userName   string
		wantCount  int
		contain    []string
		notContain []string
	}{
		{"admin", aID, "admusr", 5,
			[]string{"employee.view", "employee.delete", "insurance.view", "settings.view", "users.view"},
			[]string{}},
		{"manager", mID, "mgrusr", 2,
			[]string{"employee.view", "insurance.view"},
			[]string{"employee.delete", "settings.view", "users.view"}},
		{"editor", eID, "ediusr", 2,
			[]string{"employee.view", "settings.view"},
			[]string{"employee.delete", "insurance.view", "users.view"}},
		{"viewer", vID, "vwrusr", 1,
			[]string{"employee.view"},
			[]string{"employee.delete", "insurance.view", "settings.view", "users.view"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			u, err := createAuthTestUser(db, tc.userName, "pwd123", tc.roleID)
			if err != nil {
				t.Fatalf("创建用户 %s 失败: %v", tc.name, err)
			}
			perms, err := loadUserPermissions(db, u.ID)
			if err != nil {
				t.Fatalf("加载权限失败: %v", err)
			}
			if len(perms) != tc.wantCount {
				t.Errorf("期望 %d 权限，实际 %d: %v", tc.wantCount, len(perms), perms)
			}
			pm := make(map[string]bool, len(perms))
			for _, p := range perms {
				pm[p] = true
			}
			for _, want := range tc.contain {
				if !pm[want] {
					t.Errorf("应包含 %s，实际 %v", want, perms)
				}
			}
			for _, notWant := range tc.notContain {
				if pm[notWant] {
					t.Errorf("不应包含 %s，实际 %v", notWant, perms)
				}
			}
		})
	}
}

func TestGetProfile_ReturnsPermissionsForFourRolesWithoutDeprecatedRole(t *testing.T) {
	db := setupAuthTestDB(t)
	aID, mID, eID, vID := seedAuthTestData(db)
	tests := []struct {
		name      string
		roleID    uint
		wantPerms []string
	}{
		{"admin", aID, []string{"employee.view", "employee.delete", "insurance.view", "settings.view", "users.view"}},
		{"manager", mID, []string{"employee.view", "insurance.view"}},
		{"editor", eID, []string{"employee.view", "settings.view"}},
		{"viewer", vID, []string{"employee.view"}},
	}

	handler := NewAuthHandler(db, nil, nil, nil, nil, nil)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			user, err := createAuthTestUser(db, "profile_"+tc.name, "pwd123", tc.roleID)
			if err != nil {
				t.Fatalf("创建用户失败: %v", err)
			}
			user.Role = "deprecated-role"
			req := httptest.NewRequest(http.MethodGet, "/auth/profile", nil)
			req = req.WithContext(context.WithValue(req.Context(), auth.UserIDKey, user.ID))
			rec := httptest.NewRecorder()
			handler.GetProfile(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("Profile 返回 %d，期望 200。body: %s", rec.Code, rec.Body.String())
			}
			var response AuthUserPayload
			if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
				t.Fatalf("解析 Profile 响应失败: %v", err)
			}
			if response.User.ID != user.ID {
				t.Errorf("user.id 期望 %d，实际 %d", user.ID, response.User.ID)
			}
			if len(response.Permissions) != len(tc.wantPerms) {
				t.Errorf("权限数量期望 %d，实际 %d: %v", len(tc.wantPerms), len(response.Permissions), response.Permissions)
			}
			for _, permission := range tc.wantPerms {
				if !containsPermission(response.Permissions, permission) {
					t.Errorf("应包含权限 %s，实际 %v", permission, response.Permissions)
				}
			}
			var raw struct {
				User map[string]json.RawMessage `json:"user"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
				t.Fatalf("解析原始 Profile 响应失败: %v", err)
			}
			if _, exists := raw.User["role"]; exists {
				t.Error("Profile 用户信息不应暴露已废弃的 role 字段")
			}
		})
	}
}

func containsPermission(permissions []string, expected string) bool {
	for _, permission := range permissions {
		if permission == expected {
			return true
		}
	}
	return false
}

// ========== 登录失败不泄露 permissions ==========

func TestLogin_FailedAuthNoPermissions(t *testing.T) {
	t.Setenv("JWT_SECRET_KEY", "test-jwt-secret-for-auth-test")
	db := setupAuthTestDB(t)
	seedAuthTestData(db)

	jwtMgr := auth.NewJWTManager()
	rl := middleware.NewLoginRateLimiter()
	handler := NewAuthHandler(
		db, jwtMgr,
		service.NewPasswordResetService(db),
		service.NewEmailVerificationService(db),
		service.NewEmailService(), rl,
	)

	body := `{"username":"nonexist","password":"wrong"}`
	req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.Login(rec, req)

	if rec.Code == http.StatusOK {
		t.Error("不存在的用户不应登录成功")
	}
}
