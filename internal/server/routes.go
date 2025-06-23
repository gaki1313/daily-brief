// Package server 路由定义
package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// setupRoutes 设置所有路由
// 按照功能模块组织API路由
func (s *Server) setupRoutes() {
	// 健康检查
	s.router.GET("/health", s.healthCheck)

	// API路由组
	api := s.router.Group("/api/v1")
	{
		// 仓库管理 API
		repositories := api.Group("/repositories")
		{
			repositories.GET("", s.getRepositories)
			repositories.POST("", s.createRepository)
			repositories.GET("/:id", s.getRepository)
			repositories.PUT("/:id", s.updateRepository)
			repositories.DELETE("/:id", s.deleteRepository)
			repositories.POST("/:id/scan", s.scanRepository)
		}

		// 工作要点 API
		points := api.Group("/points")
		{
			points.GET("", s.getWorkPoints)
			points.POST("", s.createWorkPoint)
			points.GET("/:id", s.getWorkPoint)
			points.PUT("/:id", s.updateWorkPoint)
			points.DELETE("/:id", s.deleteWorkPoint)
		}

		// 日报管理 API
		reports := api.Group("/reports")
		{
			reports.GET("", s.getDailyReports)
			reports.POST("/generate", s.generateDailyReport)
			reports.GET("/:date", s.getDailyReport)
			reports.PUT("/:id", s.updateDailyReport)
			reports.DELETE("/:id", s.deleteDailyReport)
			reports.POST("/:id/send", s.sendDailyReport)
		}

		// 集成管理 API
		integrations := api.Group("/integrations")
		{
			integrations.GET("/status", s.getIntegrationStatus)
			integrations.POST("/feishu/send", s.sendToFeishu)
			integrations.GET("/logs", s.getIntegrationLogs)
		}

		// 统计信息 API
		stats := api.Group("/stats")
		{
			stats.GET("/commits", s.getCommitStats)
			stats.GET("/reports", s.getReportStats)
		}
	}

	// 静态文件服务（如果有Web界面）
	s.router.Static("/static", "./web/static")
	s.router.LoadHTMLGlob("web/templates/*")

	// Web界面路由（可选）
	s.router.GET("/", s.indexPage)
	s.router.GET("/dashboard", s.dashboardPage)
}

// healthCheck 健康检查接口
// GET /health
func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"version": s.config.App.Version,
		"name":    s.config.App.Name,
	})
}

// indexPage 首页
// GET /
func (s *Server) indexPage(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{
		"title": s.config.App.Name,
		"version": s.config.App.Version,
	})
}

// dashboardPage 仪表板页面
// GET /dashboard
func (s *Server) dashboardPage(c *gin.Context) {
	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"title": "Dashboard - " + s.config.App.Name,
	})
} 