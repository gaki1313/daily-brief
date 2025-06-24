// Package server HTTP服务器实现
// 提供RESTful API接口和Web界面
package server

import (
	"context"
	"fmt"

	"daily-brief/internal/config"
	"daily-brief/internal/feishu"
	"daily-brief/internal/git"
	"daily-brief/internal/gitlab"
	"daily-brief/internal/report"
	"daily-brief/internal/storage"
	"daily-brief/internal/workpoint"
	"daily-brief/pkg/logger"

	"github.com/gin-gonic/gin"
)

// Server HTTP服务器结构体
// 封装了路由、配置、数据库连接等核心组件
type Server struct {
	config          *config.Config           // 应用配置
	database        storage.Database         // 数据库接口
	logger          *logger.Logger           // 日志器
	router          *gin.Engine              // Gin路由引擎
	gitScheduler    *git.Scheduler           // Git扫描调度器
	gitlabClient    *gitlab.Client           // GitLab API客户端
	workpointMgr    *workpoint.Manager       // 工作要点管理器
	reportGen       *report.Generator        // 日报生成器
	feishuClient    *feishu.Client           // 飞书客户端
	feishuScheduler *feishu.SendScheduler    // 飞书发送调度器
	feishuFormatter *feishu.MessageFormatter // 飞书消息格式化器
	ctx             context.Context          // 服务器上下文
	cancelFunc      context.CancelFunc       // 取消函数
}

// New 创建新的HTTP服务器实例
// 初始化路由、中间件和处理器
func New(cfg *config.Config, db storage.Database, log *logger.Logger) *Server {
	ctx, cancel := context.WithCancel(context.Background())

	server := &Server{
		config:     cfg,
		database:   db,
		logger:     log,
		ctx:        ctx,
		cancelFunc: cancel,
	}

	// 初始化Git调度器
	server.gitScheduler = git.NewScheduler(cfg.Git, db, log.Logger)

	// 初始化GitLab客户端（如果启用）
	if cfg.Git.GitLab.Enabled {
		gitlabClient, err := gitlab.NewClient(cfg.Git.GitLab, db, log.Logger)
		if err != nil {
			log.WithError(err).Warn("Failed to initialize GitLab client")
		} else {
			server.gitlabClient = gitlabClient
		}
	}

	// 初始化工作要点管理器
	server.workpointMgr = workpoint.NewManager(db, cfg.Report, log.Logger)

	// 初始化日报生成器
	server.reportGen = report.NewGenerator(db, cfg.Report, log.Logger, server.gitScheduler, server.workpointMgr)

	// 初始化飞书客户端
	server.feishuClient = feishu.NewClient(cfg.Feishu, db, log.Logger)

	// 初始化飞书消息格式化器
	formatterConfig := feishu.FormatterConfig{
		MaxTextLength: 8000,
		UseRichText:   true,
		EnableEmoji:   true,
		Timezone:      cfg.Report.Timezone,
	}
	server.feishuFormatter = feishu.NewMessageFormatter(formatterConfig)

	// 初始化飞书调度器
	server.feishuScheduler = feishu.NewSendScheduler(cfg.Report, db, log.Logger,
		server.feishuClient, server.feishuFormatter, server.reportGen)

	// 初始化Gin路由
	server.setupRouter()

	return server
}

// setupRouter 设置路由和中间件
func (s *Server) setupRouter() {
	// 创建Gin引擎
	s.router = gin.New()

	// 添加全局中间件
	s.router.Use(s.corsMiddleware())
	s.router.Use(s.loggingMiddleware())
	s.router.Use(gin.Recovery())

	// 设置路由
	s.setupRoutes()
}

// Router 获取Gin路由引擎
// 用于HTTP服务器启动
func (s *Server) Router() *gin.Engine {
	return s.router
}

// GetConfig 获取服务器配置
func (s *Server) GetConfig() *config.Config {
	return s.config
}

// GetDatabase 获取数据库连接
func (s *Server) GetDatabase() storage.Database {
	return s.database
}

// GetLogger 获取日志器
func (s *Server) GetLogger() *logger.Logger {
	return s.logger
}

// StartServices 启动所有服务
func (s *Server) StartServices() error {
	// 启动Git调度器
	if err := s.gitScheduler.Start(s.ctx); err != nil {
		return fmt.Errorf("failed to start git scheduler: %w", err)
	}

	// 启动飞书调度器
	if err := s.feishuScheduler.Start(); err != nil {
		s.logger.WithError(err).Warn("Failed to start feishu scheduler (continuing without auto-send)")
	}

	s.logger.Info("All services started successfully")
	return nil
}

// StopServices 停止所有服务
func (s *Server) StopServices() error {
	// 停止Git调度器
	if err := s.gitScheduler.Stop(); err != nil {
		s.logger.WithError(err).Error("Failed to stop git scheduler")
	}

	// 停止飞书调度器
	if err := s.feishuScheduler.Stop(); err != nil {
		s.logger.WithError(err).Error("Failed to stop feishu scheduler")
	}

	// 取消上下文
	s.cancelFunc()

	s.logger.Info("All services stopped")
	return nil
}

// GetGitScheduler 获取Git调度器
func (s *Server) GetGitScheduler() *git.Scheduler {
	return s.gitScheduler
}

// GetReportGenerator 获取日报生成器
func (s *Server) GetReportGenerator() *report.Generator {
	return s.reportGen
}

// GetWorkPointManager 获取工作要点管理器
func (s *Server) GetWorkPointManager() *workpoint.Manager {
	return s.workpointMgr
}

// GetFeishuClient 获取飞书客户端
func (s *Server) GetFeishuClient() *feishu.Client {
	return s.feishuClient
}

// GetFeishuScheduler 获取飞书调度器
func (s *Server) GetFeishuScheduler() *feishu.SendScheduler {
	return s.feishuScheduler
}

// GetGitLabClient 获取GitLab客户端
func (s *Server) GetGitLabClient() *gitlab.Client {
	return s.gitlabClient
}
