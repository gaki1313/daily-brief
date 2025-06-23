// Package server HTTP服务器实现
// 提供RESTful API接口和Web界面
package server

import (
	"daily-brief/internal/config"
	"daily-brief/internal/storage"
	"daily-brief/pkg/logger"

	"github.com/gin-gonic/gin"
)

// Server HTTP服务器结构体
// 封装了路由、配置、数据库连接等核心组件
type Server struct {
	config   *config.Config       // 应用配置
	database storage.Database     // 数据库接口
	logger   *logger.Logger       // 日志器
	router   *gin.Engine          // Gin路由引擎
}

// New 创建新的HTTP服务器实例
// 初始化路由、中间件和处理器
func New(cfg *config.Config, db storage.Database, log *logger.Logger) *Server {
	server := &Server{
		config:   cfg,
		database: db,
		logger:   log,
	}

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