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
			points.GET("", s.getWorkPoints)                              // 获取工作要点列表
			points.POST("", s.createWorkPoint)                           // 创建工作要点（带智能分析）
			points.POST("/analyze", s.analyzeWorkPoint)                  // 分析工作要点（不保存）
			points.GET("/analysis", s.getWorkPointsWithAnalysis)         // 获取工作要点和分析信息
			points.GET("/category/:category", s.getWorkPointsByCategory) // 按分类获取工作要点
			points.GET("/stats/category", s.getCategoryStats)            // 获取分类统计信息
			points.GET("/:id", s.getWorkPoint)                           // 获取单个工作要点
			points.PUT("/:id", s.updateWorkPoint)                        // 更新工作要点
			points.DELETE("/:id", s.deleteWorkPoint)                     // 删除工作要点
		}

		// 日报管理 API
		reports := api.Group("/reports")
		{
			reports.GET("", s.getDailyReports)               // 获取日报列表
			reports.POST("/generate", s.generateDailyReport) // 生成日报
			reports.GET("/preview", s.previewReport)         // 预览日报
			reports.GET("/templates", s.getReportTemplates)  // 获取模板列表
			reports.GET("/data/:date", s.getReportData)      // 获取报告数据
			reports.GET("/:date", s.getDailyReport)          // 获取指定日期日报
			reports.PUT("/:id", s.updateDailyReport)         // 更新日报
			reports.DELETE("/:id", s.deleteDailyReport)      // 删除日报
			reports.POST("/:id/send", s.sendDailyReport)     // 发送日报

			// GitLab专用日报子路由
			gitlab := reports.Group("/gitlab")
			{
				gitlab.POST("/generate", s.generateGitLabReport) // 生成GitLab日报
				gitlab.GET("/preview", s.previewGitLabReport)    // 预览GitLab日报
				gitlab.GET("/data/:date", s.getGitLabReportData) // 获取GitLab日报数据
			}
		}

		// 集成管理 API
		integrations := api.Group("/integrations")
		{
			integrations.GET("/status", s.getIntegrationStatus)
			integrations.GET("/logs", s.getIntegrationLogs)

			// 飞书集成子路由
			feishu := integrations.Group("/feishu")
			{
				feishu.GET("/status", s.getFeishuStatus)
				feishu.POST("/test", s.testFeishuConnection)
				feishu.GET("/stats", s.getFeishuStats)
				feishu.POST("/send", s.sendToFeishu)
				feishu.POST("/send/report", s.sendReportToFeishu)
				feishu.POST("/send/gitlab-report", s.sendGitLabReportToFeishu)      // 发送GitLab日报到飞书
				feishu.POST("/send/ai-gitlab-report", s.sendAIGitLabReportToFeishu) // 发送AI生成的GitLab日报到飞书

				// 调度器管理
				scheduler := feishu.Group("/scheduler")
				{
					scheduler.GET("/status", s.getSchedulerStatus)
					scheduler.POST("/start", s.startScheduler)
					scheduler.POST("/stop", s.stopScheduler)
					scheduler.POST("/test", s.testScheduler)
				}
			}

			// GitLab集成子路由
			gitlab := integrations.Group("/gitlab")
			{
				gitlab.GET("/status", s.getGitLabStatus)
				gitlab.POST("/test", s.testGitLabConnection)
				gitlab.GET("/user", s.getGitLabUser)
				gitlab.GET("/projects", s.getGitLabProjects)
				gitlab.GET("/commits/:date", s.getGitLabCommits)
				gitlab.POST("/sync/:date", s.syncGitLabCommits)
			}
		}

		// Git相关 API
		git := api.Group("/git")
		{
			git.POST("/scan", s.scanAllRepositories) // 扫描所有仓库
			git.GET("/stats", s.getGitStats)         // 获取Git统计信息
		}

		// 提交记录 API
		commits := api.Group("/commits")
		{
			commits.GET("", s.getCommits)                  // 获取提交记录
			commits.GET("/range", s.getCommitsByDateRange) // 获取日期范围内的提交记录
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
		"title":   s.config.App.Name,
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
