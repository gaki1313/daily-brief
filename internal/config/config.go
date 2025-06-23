// Package config 提供应用程序的配置管理功能
// 定义了所有配置项的结构体，支持从YAML文件、环境变量等多种方式加载配置
package config

import (
	"fmt"
	"time"
)

// Config 是应用程序的主配置结构体
// 包含了应用运行所需的所有配置项
type Config struct {
	App       AppConfig       `mapstructure:"app" yaml:"app"`             // 应用基础配置
	Database  DatabaseConfig  `mapstructure:"database" yaml:"database"`   // 数据库配置
	Git       GitConfig       `mapstructure:"git" yaml:"git"`             // Git相关配置
	Feishu    FeishuConfig    `mapstructure:"feishu" yaml:"feishu"`       // 飞书集成配置
	Report    ReportConfig    `mapstructure:"report" yaml:"report"`       // 日报生成配置
	Logging   LoggingConfig   `mapstructure:"logging" yaml:"logging"`     // 日志配置
}

// AppConfig 应用程序基础配置
// 包含应用名称、版本、运行端口等基本信息
type AppConfig struct {
	Name    string `mapstructure:"name" yaml:"name"`       // 应用名称
	Version string `mapstructure:"version" yaml:"version"` // 应用版本
	Port    int    `mapstructure:"port" yaml:"port"`       // HTTP服务端口
	Mode    string `mapstructure:"mode" yaml:"mode"`       // 运行模式：debug/release
}

// DatabaseConfig 数据库配置
// 目前支持SQLite，未来可扩展支持MySQL、PostgreSQL等
type DatabaseConfig struct {
	Type string `mapstructure:"type" yaml:"type"` // 数据库类型：sqlite/mysql/postgres
	Path string `mapstructure:"path" yaml:"path"` // SQLite数据库文件路径
	DSN  string `mapstructure:"dsn" yaml:"dsn"`   // 数据库连接字符串（用于MySQL/PostgreSQL）
}

// GitConfig Git操作相关配置
// 配置Git仓库扫描的行为和限制
type GitConfig struct {
	ScanInterval      string `mapstructure:"scan_interval" yaml:"scan_interval"`           // 扫描间隔：1h, 30m, 等
	MaxCommitsPerDay  int    `mapstructure:"max_commits_per_day" yaml:"max_commits_per_day"` // 每日最大提交数量限制
	DefaultBranch     string `mapstructure:"default_branch" yaml:"default_branch"`         // 默认分支名
	ExcludePatterns   []string `mapstructure:"exclude_patterns" yaml:"exclude_patterns"`   // 排除的文件模式
}

// FeishuConfig 飞书集成配置
// 配置飞书机器人的认证信息和推送设置
type FeishuConfig struct {
	AppID      string `mapstructure:"app_id" yaml:"app_id"`           // 飞书应用ID
	AppSecret  string `mapstructure:"app_secret" yaml:"app_secret"`   // 飞书应用密钥
	WebhookURL string `mapstructure:"webhook_url" yaml:"webhook_url"` // 飞书机器人Webhook地址
	Enabled    bool   `mapstructure:"enabled" yaml:"enabled"`         // 是否启用飞书集成
	ChatID     string `mapstructure:"chat_id" yaml:"chat_id"`         // 群聊ID（可选）
}

// ReportConfig 日报生成配置
// 配置日报的模板、时区、工作时间等
type ReportConfig struct {
	DefaultTemplate string        `mapstructure:"default_template" yaml:"default_template"` // 默认模板名称
	Timezone        string        `mapstructure:"timezone" yaml:"timezone"`                 // 时区设置
	WorkingHours    WorkingHours  `mapstructure:"working_hours" yaml:"working_hours"`       // 工作时间配置
	AutoSend        bool          `mapstructure:"auto_send" yaml:"auto_send"`               // 是否自动发送
	SendTime        string        `mapstructure:"send_time" yaml:"send_time"`               // 自动发送时间
}

// WorkingHours 工作时间配置
// 定义工作日的开始和结束时间
type WorkingHours struct {
	Start string `mapstructure:"start" yaml:"start"` // 工作开始时间，格式：HH:MM
	End   string `mapstructure:"end" yaml:"end"`     // 工作结束时间，格式：HH:MM
}

// LoggingConfig 日志配置
// 配置日志级别、格式、输出位置和轮转策略
type LoggingConfig struct {
	Level     string `mapstructure:"level" yaml:"level"`         // 日志级别：debug/info/warn/error
	Format    string `mapstructure:"format" yaml:"format"`       // 日志格式：json/text
	File      string `mapstructure:"file" yaml:"file"`           // 日志文件路径
	MaxSize   int    `mapstructure:"max_size" yaml:"max_size"`   // 单个日志文件最大大小(MB)
	MaxAge    int    `mapstructure:"max_age" yaml:"max_age"`     // 日志文件保留天数
	MaxBackups int   `mapstructure:"max_backups" yaml:"max_backups"` // 最大备份文件数量
	Compress  bool   `mapstructure:"compress" yaml:"compress"`   // 是否压缩旧日志文件
}

// GetScanInterval 获取Git扫描间隔的Duration类型
// 将字符串格式的时间间隔转换为time.Duration类型
func (g *GitConfig) GetScanInterval() time.Duration {
	duration, err := time.ParseDuration(g.ScanInterval)
	if err != nil {
		// 如果解析失败，返回默认值1小时
		return time.Hour
	}
	return duration
}

// GetWorkingStartTime 获取工作开始时间
// 将字符串格式的时间转换为time.Time类型（仅包含时分）
func (w *WorkingHours) GetWorkingStartTime() (time.Time, error) {
	return time.Parse("15:04", w.Start)
}

// GetWorkingEndTime 获取工作结束时间
// 将字符串格式的时间转换为time.Time类型（仅包含时分）
func (w *WorkingHours) GetWorkingEndTime() (time.Time, error) {
	return time.Parse("15:04", w.End)
}

// IsWithinWorkingHours 检查指定时间是否在工作时间内
// 参数 t 是要检查的时间
// 返回 true 表示在工作时间内，false 表示不在工作时间内
func (w *WorkingHours) IsWithinWorkingHours(t time.Time) bool {
	start, err := w.GetWorkingStartTime()
	if err != nil {
		return true // 解析失败时默认认为在工作时间内
	}
	
	end, err := w.GetWorkingEndTime()
	if err != nil {
		return true // 解析失败时默认认为在工作时间内
	}
	
	// 提取当前时间的时分进行比较
	currentTime := time.Date(0, 1, 1, t.Hour(), t.Minute(), 0, 0, t.Location())
	startTime := time.Date(0, 1, 1, start.Hour(), start.Minute(), 0, 0, t.Location())
	endTime := time.Date(0, 1, 1, end.Hour(), end.Minute(), 0, 0, t.Location())
	
	return currentTime.After(startTime) && currentTime.Before(endTime)
}

// Validate 验证配置的有效性
// 检查必要的配置项是否设置正确
func (c *Config) Validate() error {
	// 检查应用端口是否在有效范围内
	if c.App.Port < 1 || c.App.Port > 65535 {
		return fmt.Errorf("invalid port number: %d, must be between 1 and 65535", c.App.Port)
	}
	
	// 检查应用运行模式是否有效
	if c.App.Mode != "debug" && c.App.Mode != "release" {
		return fmt.Errorf("invalid app mode: %s, must be 'debug' or 'release'", c.App.Mode)
	}
	
	// 检查数据库类型是否支持
	if c.Database.Type != "sqlite" {
		return fmt.Errorf("unsupported database type: %s, currently only 'sqlite' is supported", c.Database.Type)
	}
	
	// 检查SQLite数据库路径是否设置
	if c.Database.Type == "sqlite" && c.Database.Path == "" {
		return fmt.Errorf("database path is required for SQLite")
	}
	
	// 检查时区设置是否有效
	if c.Report.Timezone != "" {
		if _, err := time.LoadLocation(c.Report.Timezone); err != nil {
			return fmt.Errorf("invalid timezone: %s, error: %v", c.Report.Timezone, err)
		}
	}
	
	// 检查工作时间格式是否正确
	if _, err := c.Report.WorkingHours.GetWorkingStartTime(); err != nil {
		return fmt.Errorf("invalid working start time format: %s, expected HH:MM", c.Report.WorkingHours.Start)
	}
	
	if _, err := c.Report.WorkingHours.GetWorkingEndTime(); err != nil {
		return fmt.Errorf("invalid working end time format: %s, expected HH:MM", c.Report.WorkingHours.End)
	}
	
	return nil
}

 