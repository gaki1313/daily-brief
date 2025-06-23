// Package storage 数据库连接和操作实现
package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"daily-brief/internal/config"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Database 数据库操作接口
type Database interface {
	// 基础操作
	GetDB() *gorm.DB
	Close() error
	
	// 仓库操作
	CreateRepository(repo *Repository) error
	GetRepository(id uint) (*Repository, error)
	GetRepositories() ([]Repository, error)
	UpdateRepository(repo *Repository) error
	DeleteRepository(id uint) error
	
	// 提交记录操作
	CreateCommit(commit *Commit) error
	GetCommits(repositoryID uint, date time.Time) ([]Commit, error)
	GetCommitsByDateRange(start, end time.Time) ([]Commit, error)
	
	// 工作要点操作
	CreateWorkPoint(point *WorkPoint) error
	GetWorkPoints(date time.Time) ([]WorkPoint, error)
	GetWorkPointsByCategory(category string, date time.Time) ([]WorkPoint, error)
	UpdateWorkPoint(point *WorkPoint) error
	DeleteWorkPoint(id uint) error
	
	// 日报操作
	CreateDailyReport(report *DailyReport) error
	GetDailyReport(date time.Time) (*DailyReport, error)
	GetDailyReports(limit int) ([]DailyReport, error)
	UpdateDailyReport(report *DailyReport) error
	DeleteDailyReport(id uint) error
	
	// 集成日志操作
	CreateIntegrationLog(log *IntegrationLog) error
	GetIntegrationLogs(platform string, limit int) ([]IntegrationLog, error)
}

// database 数据库实现
type database struct {
	db  *gorm.DB
	cfg config.DatabaseConfig
	log logger.Interface
}

// NewDatabase 创建新的数据库连接
// 根据配置初始化数据库连接和日志
func NewDatabase(cfg config.DatabaseConfig, appLog interface{}) (*database, error) {
	var db *gorm.DB
	var err error
	
	// 配置GORM日志
	gormLogger := logger.Default.LogMode(logger.Info)
	if appLog != nil {
		gormLogger = logger.Default.LogMode(logger.Silent)
	}
	
	switch cfg.Type {
	case "sqlite":
		db, err = initSQLite(cfg, gormLogger)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Type)
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}
	
	return &database{
		db:  db,
		cfg: cfg,
		log: gormLogger,
	}, nil
}

// initSQLite 初始化SQLite数据库
func initSQLite(cfg config.DatabaseConfig, log logger.Interface) (*gorm.DB, error) {
	// 确保数据库目录存在
	dbDir := filepath.Dir(cfg.Path)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}
	
	// 连接SQLite数据库
	db, err := gorm.Open(sqlite.Open(cfg.Path), &gorm.Config{
		Logger: log,
		NowFunc: func() time.Time {
			return time.Now().Local()
		},
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SQLite database: %w", err)
	}
	
	// 配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying database connection: %w", err)
	}
	
	// SQLite不需要连接池，但设置一些基本参数
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxLifetime(time.Hour)
	
	return db, nil
}

// GetDB 获取GORM数据库实例
func (d *database) GetDB() *gorm.DB {
	return d.db
}

// Close 关闭数据库连接
func (d *database) Close() error {
	sqlDB, err := d.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying database connection: %w", err)
	}
	return sqlDB.Close()
}

// CreateRepository 创建新仓库
func (d *database) CreateRepository(repo *Repository) error {
	if err := d.db.Create(repo).Error; err != nil {
		return fmt.Errorf("failed to create repository: %w", err)
	}
	return nil
}

// GetRepository 根据ID获取仓库
func (d *database) GetRepository(id uint) (*Repository, error) {
	var repo Repository
	if err := d.db.First(&repo, id).Error; err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}
	return &repo, nil
}

// GetRepositories 获取所有仓库
func (d *database) GetRepositories() ([]Repository, error) {
	var repos []Repository
	if err := d.db.Find(&repos).Error; err != nil {
		return nil, fmt.Errorf("failed to get repositories: %w", err)
	}
	return repos, nil
}

// UpdateRepository 更新仓库信息
func (d *database) UpdateRepository(repo *Repository) error {
	if err := d.db.Save(repo).Error; err != nil {
		return fmt.Errorf("failed to update repository: %w", err)
	}
	return nil
}

// DeleteRepository 删除仓库
func (d *database) DeleteRepository(id uint) error {
	if err := d.db.Delete(&Repository{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete repository: %w", err)
	}
	return nil
}

// CreateCommit 创建新提交记录
func (d *database) CreateCommit(commit *Commit) error {
	if err := d.db.Create(commit).Error; err != nil {
		return fmt.Errorf("failed to create commit: %w", err)
	}
	return nil
}

// GetCommits 获取指定仓库和日期的提交记录
func (d *database) GetCommits(repositoryID uint, date time.Time) ([]Commit, error) {
	var commits []Commit
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)
	
	err := d.db.Where("repository_id = ? AND date >= ? AND date < ?", 
		repositoryID, startOfDay, endOfDay).
		Order("date DESC").
		Find(&commits).Error
		
	if err != nil {
		return nil, fmt.Errorf("failed to get commits: %w", err)
	}
	return commits, nil
}

// GetCommitsByDateRange 获取日期范围内的提交记录
func (d *database) GetCommitsByDateRange(start, end time.Time) ([]Commit, error) {
	var commits []Commit
	err := d.db.Preload("Repository").
		Where("date >= ? AND date <= ?", start, end).
		Order("date DESC").
		Find(&commits).Error
		
	if err != nil {
		return nil, fmt.Errorf("failed to get commits by date range: %w", err)
	}
	return commits, nil
}

// CreateWorkPoint 创建新工作要点
func (d *database) CreateWorkPoint(point *WorkPoint) error {
	if err := d.db.Create(point).Error; err != nil {
		return fmt.Errorf("failed to create work point: %w", err)
	}
	return nil
}

// GetWorkPoints 获取指定日期的工作要点
func (d *database) GetWorkPoints(date time.Time) ([]WorkPoint, error) {
	var points []WorkPoint
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)
	
	err := d.db.Where("date >= ? AND date < ?", startOfDay, endOfDay).
		Order("priority DESC, created_at DESC").
		Find(&points).Error
		
	if err != nil {
		return nil, fmt.Errorf("failed to get work points: %w", err)
	}
	return points, nil
}

// GetWorkPointsByCategory 获取指定类别和日期的工作要点
func (d *database) GetWorkPointsByCategory(category string, date time.Time) ([]WorkPoint, error) {
	var points []WorkPoint
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)
	
	err := d.db.Where("category = ? AND date >= ? AND date < ?", 
		category, startOfDay, endOfDay).
		Order("priority DESC, created_at DESC").
		Find(&points).Error
		
	if err != nil {
		return nil, fmt.Errorf("failed to get work points by category: %w", err)
	}
	return points, nil
}

// UpdateWorkPoint 更新工作要点
func (d *database) UpdateWorkPoint(point *WorkPoint) error {
	if err := d.db.Save(point).Error; err != nil {
		return fmt.Errorf("failed to update work point: %w", err)
	}
	return nil
}

// DeleteWorkPoint 删除工作要点
func (d *database) DeleteWorkPoint(id uint) error {
	if err := d.db.Delete(&WorkPoint{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete work point: %w", err)
	}
	return nil
}

// CreateDailyReport 创建新日报
func (d *database) CreateDailyReport(report *DailyReport) error {
	if err := d.db.Create(report).Error; err != nil {
		return fmt.Errorf("failed to create daily report: %w", err)
	}
	return nil
}

// GetDailyReport 获取指定日期的日报
func (d *database) GetDailyReport(date time.Time) (*DailyReport, error) {
	var report DailyReport
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	
	err := d.db.Where("date >= ? AND date < ?", 
		startOfDay, startOfDay.Add(24*time.Hour)).
		First(&report).Error
		
	if err != nil {
		return nil, fmt.Errorf("failed to get daily report: %w", err)
	}
	return &report, nil
}

// GetDailyReports 获取最近的日报列表
func (d *database) GetDailyReports(limit int) ([]DailyReport, error) {
	var reports []DailyReport
	err := d.db.Order("date DESC").Limit(limit).Find(&reports).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get daily reports: %w", err)
	}
	return reports, nil
}

// UpdateDailyReport 更新日报
func (d *database) UpdateDailyReport(report *DailyReport) error {
	if err := d.db.Save(report).Error; err != nil {
		return fmt.Errorf("failed to update daily report: %w", err)
	}
	return nil
}

// DeleteDailyReport 删除日报
func (d *database) DeleteDailyReport(id uint) error {
	if err := d.db.Delete(&DailyReport{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete daily report: %w", err)
	}
	return nil
}

// CreateIntegrationLog 创建集成日志
func (d *database) CreateIntegrationLog(log *IntegrationLog) error {
	if err := d.db.Create(log).Error; err != nil {
		return fmt.Errorf("failed to create integration log: %w", err)
	}
	return nil
}

// GetIntegrationLogs 获取集成日志
func (d *database) GetIntegrationLogs(platform string, limit int) ([]IntegrationLog, error) {
	var logs []IntegrationLog
	query := d.db.Order("created_at DESC")
	
	if platform != "" {
		query = query.Where("platform = ?", platform)
	}
	
	if limit > 0 {
		query = query.Limit(limit)
	}
	
	err := query.Find(&logs).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get integration logs: %w", err)
	}
	return logs, nil
} 