// Package storage 数据库迁移实现
package storage

import (
	"fmt"

	"gorm.io/gorm"
)

// RunMigrations 执行数据库迁移
// 自动创建和更新数据库表结构
func RunMigrations(db *gorm.DB) error {
	// 执行自动迁移
	// GORM会根据结构体定义自动创建表和索引
	err := db.AutoMigrate(
		&Repository{},      // 仓库表
		&Commit{},          // 提交记录表
		&WorkPoint{},       // 工作要点表
		&DailyReport{},     // 日报表
		&IntegrationLog{},  // 集成日志表
	)
	
	if err != nil {
		return fmt.Errorf("failed to run auto migration: %w", err)
	}
	
	// 创建自定义索引
	if err := createCustomIndexes(db); err != nil {
		return fmt.Errorf("failed to create custom indexes: %w", err)
	}
	
	return nil
}

// createCustomIndexes 创建自定义索引
// 用于优化查询性能
func createCustomIndexes(db *gorm.DB) error {
	// 创建复合索引以优化查询性能
	indexes := []struct {
		table   string
		name    string
		columns string
	}{
		// 提交记录表的复合索引
		{
			table:   "commits",
			name:    "idx_commits_repo_date",
			columns: "repository_id, date",
		},
		{
			table:   "commits",
			name:    "idx_commits_author_date",
			columns: "author, date",
		},
		
		// 工作要点表的复合索引
		{
			table:   "work_points",
			name:    "idx_work_points_date_category",
			columns: "date, category",
		},
		{
			table:   "work_points",
			name:    "idx_work_points_category_priority",
			columns: "category, priority",
		},
		
		// 集成日志表的复合索引
		{
			table:   "integration_logs",
			name:    "idx_integration_logs_platform_created",
			columns: "platform, created_at",
		},
	}
	
	// 创建索引
	for _, idx := range indexes {
		sql := fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s (%s)", 
			idx.name, idx.table, idx.columns)
		
		if err := db.Exec(sql).Error; err != nil {
			return fmt.Errorf("failed to create index %s: %w", idx.name, err)
		}
	}
	
	return nil
}

// CreateInitialData 创建初始数据
// 可选：插入一些默认数据
func CreateInitialData(db *gorm.DB) error {
	// 检查是否已有数据
	var count int64
	if err := db.Model(&Repository{}).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to count repositories: %w", err)
	}
	
	// 如果已有数据，跳过初始化
	if count > 0 {
		return nil
	}
	
	// 可以在这里添加默认仓库或其他初始数据
	// 例如：
	// defaultRepo := &Repository{
	//     Name:    "Default Repository",
	//     Path:    "./",
	//     Branch:  "main",
	//     Enabled: true,
	// }
	// if err := db.Create(defaultRepo).Error; err != nil {
	//     return fmt.Errorf("failed to create default repository: %w", err)
	// }
	
	return nil
}

// DropAllTables 删除所有表（仅用于开发和测试）
func DropAllTables(db *gorm.DB) error {
	return db.Migrator().DropTable(
		&Repository{},
		&Commit{},
		&WorkPoint{},
		&DailyReport{},
		&IntegrationLog{},
	)
}

// GetMigrationInfo 获取迁移信息
func GetMigrationInfo(db *gorm.DB) (map[string]interface{}, error) {
	info := make(map[string]interface{})
	
	// 检查表是否存在
	tables := []string{"repositories", "commits", "work_points", "daily_reports", "integration_logs"}
	
	for _, table := range tables {
		exists := db.Migrator().HasTable(table)
		info[table] = map[string]interface{}{
			"exists": exists,
		}
		
		if exists {
			// 获取表的记录数
			var count int64
			db.Table(table).Count(&count)
			info[table].(map[string]interface{})["count"] = count
		}
	}
	
	return info, nil
} 