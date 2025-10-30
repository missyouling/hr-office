package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// 配置结构
type Config struct {
	SourceDBType     string `json:"source_db_type"`
	SourceDBPath     string `json:"source_db_path"`
	SourceDBHost     string `json:"source_db_host"`
	SourceDBPort     string `json:"source_db_port"`
	SourceDBName     string `json:"source_db_name"`
	SourceDBUser     string `json:"source_db_user"`
	SourceDBPassword string `json:"source_db_password"`
	SupabaseURL      string `json:"supabase_url"`
	SupabaseKey      string `json:"supabase_service_key"`
}

// 旧数据库用户结构
type OldUser struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

func main() {
	log.Println("🚀 开始数据迁移到 Supabase...")

	// 1. 读取配置
	config, err := loadConfig()
	if err != nil {
		log.Fatalf("❌ 加载配置失败: %v", err)
	}

	// 2. 连接源数据库
	sourceDB, err := connectSourceDB(config)
	if err != nil {
		log.Fatalf("❌ 连接源数据库失败: %v", err)
	}
	defer sourceDB.Close()

	// 3. 读取旧用户数据
	oldUsers, err := fetchOldUsers(sourceDB)
	if err != nil {
		log.Fatalf("❌ 读取旧用户数据失败: %v", err)
	}
	log.Printf("✅ 找到 %d 个用户", len(oldUsers))

	// 4. 生成用户迁移报告
	log.Println("\n📋 用户迁移说明:")
	log.Println("   由于 Supabase Auth 需要用户自己设置密码，")
	log.Println("   此脚本将生成用户列表报告，请按以下方式迁移：")
	log.Println("\n   方式1: 在 Supabase Dashboard 中批量导入用户")
	log.Println("   方式2: 使用 Supabase CLI 批量创建用户")
	log.Println("   方式3: 为每个用户发送密码重置邮件")

	reportFile := "user_migration_report.json"
	report := map[string]interface{}{
		"total_users": len(oldUsers),
		"migrated_at": time.Now(),
		"old_users":   oldUsers,
		"instructions": []string{
			"1. 在 Supabase Dashboard → Authentication → Users 中手动导入用户",
			"2. 或使用 Supabase CLI: supabase db seed",
			"3. 导入后，更新 user_mappings 表以保持 ID 映射关系",
			"4. 然后运行业务数据迁移脚本",
		},
	}

	reportJSON, _ := json.MarshalIndent(report, "", "  ")
	if err := ioutil.WriteFile(reportFile, reportJSON, 0644); err != nil {
		log.Fatalf("❌ 写入报告文件失败: %v", err)
	}

	log.Printf("\n✅ 用户迁移报告已生成: %s", reportFile)
	log.Println("\n下一步:")
	log.Println("1. 查看报告文件了解用户数据")
	log.Println("2. 在 Supabase Dashboard 中创建用户账户")
	log.Println("3. 记录新旧用户 ID 的映射关系")
	log.Println("4. 运行业务数据迁移脚本")
	log.Println("\n🎉 第一阶段完成！")
}

// 加载配置
func loadConfig() (*Config, error) {
	configFile := "migration-config.json"

	// 如果配置文件不存在，创建示例配置
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		example := &Config{
			SourceDBType: "sqlite",
			SourceDBPath: "./backend/data/siapp.db",
			SupabaseURL:  "https://your-project.supabase.co",
			SupabaseKey:  "your-service-role-key",
		}
		exampleJSON, _ := json.MarshalIndent(example, "", "  ")
		if err := ioutil.WriteFile(configFile, exampleJSON, 0644); err != nil {
			return nil, fmt.Errorf("创建示例配置失败: %w", err)
		}
		return nil, fmt.Errorf("请先编辑 %s 填写实际配置", configFile)
	}

	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	return &config, nil
}

// 连接源数据库
func connectSourceDB(config *Config) (*sql.DB, error) {
	var dsn string
	var driver string

	if config.SourceDBType == "sqlite" {
		driver = "sqlite3"
		dsn = config.SourceDBPath
	} else if config.SourceDBType == "postgres" {
		driver = "postgres"
		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			config.SourceDBHost,
			config.SourceDBPort,
			config.SourceDBUser,
			config.SourceDBPassword,
			config.SourceDBName,
		)
	} else {
		return nil, fmt.Errorf("不支持的数据库类型: %s", config.SourceDBType)
	}

	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	return db, nil
}

// 读取旧用户数据
func fetchOldUsers(db *sql.DB) ([]OldUser, error) {
	query := `SELECT id, name, email, created_at FROM users ORDER BY id`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]OldUser, 0)
	for rows.Next() {
		var user OldUser
		if err := rows.Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, rows.Err()
}
