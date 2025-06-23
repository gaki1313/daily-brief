// Package storage 数据存储层
// 定义了所有数据模型和数据库操作
package storage

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Repository 代表一个Git仓库
// 存储仓库的基本信息和配置
type Repository struct {
	ID        uint      `gorm:"primaryKey" json:"id"`                    // 主键ID
	Name      string    `gorm:"not null;size:255" json:"name"`           // 仓库名称
	Path      string    `gorm:"not null;size:500" json:"path"`           // 本地路径
	URL       string    `gorm:"size:500" json:"url"`                     // 远程URL
	Branch    string    `gorm:"default:main;size:100" json:"branch"`     // 分支名
	Enabled   bool      `gorm:"default:true" json:"enabled"`             // 是否启用
	CreatedAt time.Time `json:"created_at"`                              // 创建时间
	UpdatedAt time.Time `json:"updated_at"`                              // 更新时间

	// 关联关系
	Commits []Commit `gorm:"foreignKey:RepositoryID" json:"commits,omitempty"` // 关联的提交记录
}

// Commit 代表一个Git提交记录
// 存储提交的详细信息和统计数据
type Commit struct {
	ID           uint      `gorm:"primaryKey" json:"id"`                        // 主键ID
	Hash         string    `gorm:"uniqueIndex;not null;size:40" json:"hash"`    // 提交哈希
	RepositoryID uint      `gorm:"not null;index" json:"repository_id"`         // 关联的仓库ID
	Author       string    `gorm:"not null;size:255" json:"author"`             // 作者
	Email        string    `gorm:"size:255" json:"email"`                       // 作者邮箱
	Message      string    `gorm:"not null;type:text" json:"message"`           // 提交信息
	Date         time.Time `gorm:"not null;index" json:"date"`                  // 提交时间
	FilesChanged int       `gorm:"default:0" json:"files_changed"`              // 变更文件数
	Additions    int       `gorm:"default:0" json:"additions"`                  // 新增行数
	Deletions    int       `gorm:"default:0" json:"deletions"`                  // 删除行数
	Type         string    `gorm:"size:50;index" json:"type"`                   // 提交类型
	CreatedAt    time.Time `json:"created_at"`                                  // 创建时间

	// 关联关系
	Repository Repository `gorm:"foreignKey:RepositoryID" json:"repository,omitempty"` // 关联的仓库
}

// WorkPoint 代表一个工作要点
// 用户手动添加的工作内容和计划
type WorkPoint struct {
	ID          uint      `gorm:"primaryKey" json:"id"`                    // 主键ID
	Title       string    `gorm:"not null;size:255" json:"title"`          // 标题
	Description string    `gorm:"type:text" json:"description"`            // 详细描述
	Category    string    `gorm:"not null;size:50;index" json:"category"`  // 分类
	Priority    int       `gorm:"default:0" json:"priority"`               // 优先级
	Tags        string    `gorm:"type:text" json:"tags"`                   // 标签(JSON数组)
	Date        time.Time `gorm:"not null;index" json:"date"`              // 日期
	Completed   bool      `gorm:"default:false" json:"completed"`          // 是否完成
	CreatedAt   time.Time `json:"created_at"`                              // 创建时间
	UpdatedAt   time.Time `json:"updated_at"`                              // 更新时间
}

// DailyReport 代表一份日报
// 存储生成的日报内容和元数据
type DailyReport struct {
	ID        uint       `gorm:"primaryKey" json:"id"`                         // 主键ID
	Date      time.Time  `gorm:"uniqueIndex;not null" json:"date"`             // 日报日期
	Title     string     `gorm:"not null;size:255" json:"title"`               // 标题
	Content   string     `gorm:"not null;type:text" json:"content"`            // 内容
	Template  string     `gorm:"not null;size:100" json:"template"`            // 模板名称
	Status    string     `gorm:"default:draft;size:50" json:"status"`          // 状态
	SentAt    *time.Time `json:"sent_at"`                                      // 发送时间
	CreatedAt time.Time  `json:"created_at"`                                   // 创建时间
	UpdatedAt time.Time  `json:"updated_at"`                                   // 更新时间
}

// IntegrationLog 代表集成操作日志
// 记录与第三方平台的交互日志
type IntegrationLog struct {
	ID         uint      `gorm:"primaryKey" json:"id"`                    // 主键ID
	Platform   string    `gorm:"not null;size:50;index" json:"platform"` // 平台名称
	Action     string    `gorm:"not null;size:100" json:"action"`         // 操作类型
	Status     string    `gorm:"not null;size:50" json:"status"`          // 操作状态
	Request    string    `gorm:"type:text" json:"request"`                // 请求内容
	Response   string    `gorm:"type:text" json:"response"`               // 响应内容
	Error      string    `gorm:"type:text" json:"error"`                  // 错误信息
	Duration   int64     `gorm:"default:0" json:"duration"`               // 耗时(毫秒)
	CreatedAt  time.Time `json:"created_at"`                              // 创建时间
}

// BeforeCreate GORM钩子，在创建记录前执行
func (r *Repository) BeforeCreate(tx *gorm.DB) error {
	// 可以在这里添加创建前的逻辑
	return nil
}

// BeforeUpdate GORM钩子，在更新记录前执行
func (r *Repository) BeforeUpdate(tx *gorm.DB) error {
	// 可以在这里添加更新前的逻辑
	return nil
}

// IsActive 检查仓库是否处于活跃状态
func (r *Repository) IsActive() bool {
	return r.Enabled
}

// GetDisplayName 获取仓库的显示名称
func (r *Repository) GetDisplayName() string {
	if r.Name != "" {
		return r.Name
	}
	return r.Path
}

// GetCommitType 根据提交信息推断提交类型
func (c *Commit) GetCommitType() string {
	if c.Type != "" {
		return c.Type
	}
	
	// 根据提交信息推断类型
	message := c.Message
	if len(message) > 0 {
		switch {
		case contains(message, "feat", "feature", "add"):
			return "feature"
		case contains(message, "fix", "bug"):
			return "bugfix"
		case contains(message, "refactor"):
			return "refactor"
		case contains(message, "docs", "doc"):
			return "docs"
		case contains(message, "test"):
			return "test"
		case contains(message, "style"):
			return "style"
		case contains(message, "chore"):
			return "chore"
		default:
			return "unknown"
		}
	}
	
	return "unknown"
}

// GetChangesSummary 获取变更摘要
func (c *Commit) GetChangesSummary() string {
	if c.FilesChanged == 0 {
		return "No changes"
	}
	
	return fmt.Sprintf("%d files changed, +%d -%d", 
		c.FilesChanged, c.Additions, c.Deletions)
}

// IsCompleted 检查工作要点是否已完成
func (w *WorkPoint) IsCompleted() bool {
	return w.Completed || w.Category == "completed"
}

// IsInProgress 检查工作要点是否正在进行中
func (w *WorkPoint) IsInProgress() bool {
	return w.Category == "ongoing" || w.Category == "in_progress"
}

// IsPlanned 检查工作要点是否为计划项
func (w *WorkPoint) IsPlanned() bool {
	return w.Category == "planned" || w.Category == "todo"
}

// GetPriorityText 获取优先级文本
func (w *WorkPoint) GetPriorityText() string {
	switch w.Priority {
	case 3:
		return "High"
	case 2:
		return "Medium"
	case 1:
		return "Low"
	default:
		return "Normal"
	}
}

// IsDraft 检查日报是否为草稿状态
func (d *DailyReport) IsDraft() bool {
	return d.Status == "draft"
}

// IsPublished 检查日报是否已发布
func (d *DailyReport) IsPublished() bool {
	return d.Status == "published"
}

// IsSent 检查日报是否已发送
func (d *DailyReport) IsSent() bool {
	return d.SentAt != nil
}

// MarkAsSent 标记日报为已发送
func (d *DailyReport) MarkAsSent() {
	now := time.Now()
	d.SentAt = &now
	d.Status = "sent"
}

// IsSuccess 检查集成操作是否成功
func (i *IntegrationLog) IsSuccess() bool {
	return i.Status == "success"
}

// IsError 检查集成操作是否失败
func (i *IntegrationLog) IsError() bool {
	return i.Status == "error" || i.Status == "failed"
}

// GetDurationText 获取耗时文本
func (i *IntegrationLog) GetDurationText() string {
	if i.Duration < 1000 {
		return fmt.Sprintf("%dms", i.Duration)
	}
	return fmt.Sprintf("%.2fs", float64(i.Duration)/1000.0)
}

// contains 检查字符串是否包含任一关键词
func contains(text string, keywords ...string) bool {
	text = strings.ToLower(text)
	for _, keyword := range keywords {
		if strings.Contains(text, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

 