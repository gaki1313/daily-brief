// Package logger 提供应用程序的日志功能
// 基于logrus实现结构化日志，支持多种输出格式和日志轮转
package logger

import (
	"io"
	"os"
	"path/filepath"

	"daily-brief/internal/config"

	"github.com/sirupsen/logrus"
)

// Logger 是应用程序的日志器接口
// 封装了logrus.Logger，提供了统一的日志接口
type Logger struct {
	*logrus.Logger
}

// New 创建一个新的Logger实例
// 根据配置设置日志级别、格式和输出位置
func New(cfg config.LoggingConfig) *Logger {
	log := logrus.New()

	// 设置日志级别
	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		log.Warnf("Invalid log level '%s', using 'info' level", cfg.Level)
		level = logrus.InfoLevel
	}
	log.SetLevel(level)

	// 设置日志格式
	if cfg.Format == "json" {
		log.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
		})
	} else {
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
		})
	}

	// 设置输出位置
	setupOutput(log, cfg)

	return &Logger{Logger: log}
}

// setupOutput 设置日志输出位置
// 支持文件输出和控制台输出
func setupOutput(log *logrus.Logger, cfg config.LoggingConfig) {
	var writers []io.Writer

	// 总是输出到控制台
	writers = append(writers, os.Stdout)

	// 如果配置了文件输出
	if cfg.File != "" {
		// 确保日志目录存在
		if err := os.MkdirAll(filepath.Dir(cfg.File), 0755); err != nil {
			log.Errorf("Failed to create log directory: %v", err)
		} else {
			// 配置日志轮转
			fileWriter := &SimpleLumberjack{
				Filename:   cfg.File,
				MaxSize:    cfg.MaxSize,    // MB
				MaxAge:     cfg.MaxAge,     // days
				MaxBackups: cfg.MaxBackups,
				Compress:   cfg.Compress,
			}
			writers = append(writers, fileWriter)
		}
	}

	// 设置多重输出
	if len(writers) > 1 {
		log.SetOutput(io.MultiWriter(writers...))
	} else {
		log.SetOutput(writers[0])
	}
}

// WithField 添加单个字段到日志上下文
func (l *Logger) WithField(key string, value interface{}) *logrus.Entry {
	return l.Logger.WithField(key, value)
}

// WithFields 添加多个字段到日志上下文
func (l *Logger) WithFields(fields logrus.Fields) *logrus.Entry {
	return l.Logger.WithFields(fields)
}

// WithError 添加错误到日志上下文
func (l *Logger) WithError(err error) *logrus.Entry {
	return l.Logger.WithError(err)
} 