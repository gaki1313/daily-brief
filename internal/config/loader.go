// Package config 配置加载器实现
// 使用Viper库支持从多种来源加载配置：YAML文件、环境变量、命令行参数等
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// configFileNames 配置文件名称的优先级列表
// 按照优先级从高到低排列，程序会依次查找这些文件
var configFileNames = []string{
	"config.local", // 本地开发配置（最高优先级，通常不提交到版本控制）
	"config.dev",   // 开发环境配置
	"config.prod",  // 生产环境配置
	"config",       // 默认配置文件
}

// configPaths 配置文件查找路径列表
// 按照优先级从高到低排列，程序会在这些路径中查找配置文件
var configPaths = []string{
	".",                // 当前目录
	"./configs",        // configs目录
	"./config",         // config目录
	"/etc/daily-brief", // 系统配置目录
}

// Load 加载应用配置
// 支持从多种来源加载配置，优先级从高到低：
// 1. 环境变量
// 2. 配置文件
// 3. 默认值
// 返回完整的配置对象和可能的错误
func Load() (*Config, error) {
	v := viper.New()

	// 第一步：设置默认配置值
	// 这些默认值确保即使没有配置文件，应用也能正常启动
	setDefaults(v)

	// 第二步：配置文件查找和加载
	if err := loadConfigFile(v); err != nil {
		return nil, fmt.Errorf("failed to load config file: %w", err)
	}

	// 第三步：配置环境变量支持
	// 环境变量的优先级高于配置文件
	setupEnvironmentVariables(v)

	// 第四步：将配置解析到结构体
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 第五步：验证配置的有效性
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

// setDefaults 设置默认配置值
// 定义了所有配置项的默认值，确保应用的基本功能
func setDefaults(v *viper.Viper) {
	// 应用基础配置默认值
	v.SetDefault("app.name", "Daily Brief Generator")
	v.SetDefault("app.version", "1.0.0")
	v.SetDefault("app.port", 8080)
	v.SetDefault("app.mode", "release")

	// 数据库配置默认值
	v.SetDefault("database.type", "sqlite")
	v.SetDefault("database.path", "./data/daily-brief.db")

	// Git配置默认值
	v.SetDefault("git.scan_interval", "1h")
	v.SetDefault("git.max_commits_per_day", 50)
	v.SetDefault("git.default_branch", "main")
	v.SetDefault("git.exclude_patterns", []string{
		"*.log", "*.tmp", "node_modules/*", ".git/*",
	})

	// 飞书配置默认值
	v.SetDefault("feishu.enabled", false)
	v.SetDefault("feishu.app_id", "")
	v.SetDefault("feishu.app_secret", "")
	v.SetDefault("feishu.webhook_url", "")
	v.SetDefault("feishu.chat_id", "")

	// 日报配置默认值
	v.SetDefault("report.default_template", "standard")
	v.SetDefault("report.timezone", "Asia/Shanghai")
	v.SetDefault("report.working_hours.start", "09:00")
	v.SetDefault("report.working_hours.end", "18:00")
	v.SetDefault("report.auto_send", false)
	v.SetDefault("report.send_time", "18:30")

	// AI配置默认值
	v.SetDefault("report.ai.enabled", false)
	v.SetDefault("report.ai.provider", "ollama")
	v.SetDefault("report.ai.ollama_url", "http://localhost:11434")
	v.SetDefault("report.ai.ollama_model", "deepseek-r1:7b")
	v.SetDefault("report.ai.openai_model", "gpt-3.5-turbo")
	v.SetDefault("report.ai.temperature", 0.7)
	v.SetDefault("report.ai.max_tokens", 2000)
	v.SetDefault("report.ai.timeout", 60)

	// 日志配置默认值
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")
	v.SetDefault("logging.file", "./logs/app.log")
	v.SetDefault("logging.max_size", 100)
	v.SetDefault("logging.max_age", 30)
	v.SetDefault("logging.max_backups", 10)
	v.SetDefault("logging.compress", true)
}

// loadConfigFile 加载配置文件
// 按照优先级查找并加载配置文件
func loadConfigFile(v *viper.Viper) error {
	// 设置配置文件格式
	v.SetConfigType("yaml")

	// 首先尝试从环境变量获取配置文件路径
	if configFile := os.Getenv("CONFIG_FILE"); configFile != "" {
		v.SetConfigFile(configFile)
		if err := v.ReadInConfig(); err != nil {
			return fmt.Errorf("failed to read config file %s: %w", configFile, err)
		}
		fmt.Printf("Using config file: %s\n", v.ConfigFileUsed())
		return nil
	}

	// 在指定路径中查找配置文件
	var configFound bool
	var lastErr error

	for _, path := range configPaths {
		for _, name := range configFileNames {
			configFile := filepath.Join(path, name+".yaml")

			// 检查文件是否存在
			if _, err := os.Stat(configFile); os.IsNotExist(err) {
				// 尝试.yml扩展名
				configFile = filepath.Join(path, name+".yml")
				if _, err := os.Stat(configFile); os.IsNotExist(err) {
					continue
				}
			}

			// 尝试加载配置文件
			v.SetConfigFile(configFile)
			if err := v.ReadInConfig(); err != nil {
				lastErr = err
				continue
			}

			fmt.Printf("Using config file: %s\n", v.ConfigFileUsed())
			configFound = true
			break
		}

		if configFound {
			break
		}
	}

	// 如果没有找到任何配置文件，使用默认配置
	if !configFound {
		fmt.Println("No config file found, using default configuration")
		return nil
	}

	return lastErr
}

// setupEnvironmentVariables 设置环境变量支持
// 配置Viper从环境变量读取配置，环境变量的优先级高于配置文件
func setupEnvironmentVariables(v *viper.Viper) {
	// 设置环境变量前缀
	v.SetEnvPrefix("DAILY_BRIEF")

	// 设置环境变量key的分隔符
	// 例如：DAILY_BRIEF_APP_PORT 对应配置项 app.port
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 自动读取环境变量
	v.AutomaticEnv()

	// 手动绑定一些重要的环境变量
	envBindings := map[string]string{
		"app.port":           "PORT",
		"app.mode":           "GIN_MODE",
		"database.path":      "DB_PATH",
		"feishu.app_id":      "FEISHU_APP_ID",
		"feishu.app_secret":  "FEISHU_APP_SECRET",
		"feishu.webhook_url": "FEISHU_WEBHOOK_URL",
		"logging.level":      "LOG_LEVEL",
		"logging.file":       "LOG_FILE",
	}

	for configKey, envKey := range envBindings {
		if err := v.BindEnv(configKey, envKey); err != nil {
			// 绑定失败不是致命错误，记录但继续执行
			fmt.Printf("Warning: failed to bind environment variable %s: %v\n", envKey, err)
		}
	}
}

// GetConfigPath 获取配置文件路径
// 用于调试和日志记录
func GetConfigPath() string {
	v := viper.New()
	setupEnvironmentVariables(v)

	if err := loadConfigFile(v); err != nil {
		return "default configuration (no file)"
	}

	return v.ConfigFileUsed()
}

// ReloadConfig 重新加载配置
// 用于热更新配置，无需重启应用
func ReloadConfig() (*Config, error) {
	return Load()
}

// SaveConfig 保存配置到文件
// 用于将当前配置保存到指定文件
func SaveConfig(config *Config, filename string) error {
	v := viper.New()
	v.SetConfigType("yaml")
	v.SetConfigFile(filename)

	// 将配置结构体转换为map
	configMap := make(map[string]interface{})

	// 应用配置
	configMap["app"] = map[string]interface{}{
		"name":    config.App.Name,
		"version": config.App.Version,
		"port":    config.App.Port,
		"mode":    config.App.Mode,
	}

	// 数据库配置
	configMap["database"] = map[string]interface{}{
		"type": config.Database.Type,
		"path": config.Database.Path,
	}

	// Git配置
	configMap["git"] = map[string]interface{}{
		"scan_interval":       config.Git.ScanInterval,
		"max_commits_per_day": config.Git.MaxCommitsPerDay,
		"default_branch":      config.Git.DefaultBranch,
		"exclude_patterns":    config.Git.ExcludePatterns,
	}

	// 飞书配置
	configMap["feishu"] = map[string]interface{}{
		"app_id":      config.Feishu.AppID,
		"app_secret":  config.Feishu.AppSecret,
		"webhook_url": config.Feishu.WebhookURL,
		"enabled":     config.Feishu.Enabled,
		"chat_id":     config.Feishu.ChatID,
	}

	// 日报配置
	configMap["report"] = map[string]interface{}{
		"default_template": config.Report.DefaultTemplate,
		"timezone":         config.Report.Timezone,
		"auto_send":        config.Report.AutoSend,
		"send_time":        config.Report.SendTime,
		"working_hours": map[string]interface{}{
			"start": config.Report.WorkingHours.Start,
			"end":   config.Report.WorkingHours.End,
		},
	}

	// 日志配置
	configMap["logging"] = map[string]interface{}{
		"level":       config.Logging.Level,
		"format":      config.Logging.Format,
		"file":        config.Logging.File,
		"max_size":    config.Logging.MaxSize,
		"max_age":     config.Logging.MaxAge,
		"max_backups": config.Logging.MaxBackups,
		"compress":    config.Logging.Compress,
	}

	// 合并配置
	for key, value := range configMap {
		v.Set(key, value)
	}

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// 写入配置文件
	if err := v.WriteConfig(); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
