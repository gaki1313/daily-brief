// Package server API处理器实现
// 提供RESTful API的具体实现
package server

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"daily-brief/internal/feishu"
	"daily-brief/internal/report"
	"daily-brief/internal/storage"

	"github.com/gin-gonic/gin"
)

// 通用响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// 分页参数结构
type PaginationParams struct {
	Page     int `form:"page,default=1"`
	PageSize int `form:"page_size,default=20"`
}

// ========== 仓库管理 API ==========

// getRepositories 获取仓库列表
// GET /api/v1/repositories
func (s *Server) getRepositories(c *gin.Context) {
	repositories, err := s.database.GetRepositories()
	if err != nil {
		s.logger.WithError(err).Error("Failed to get repositories")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to get repositories",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Success",
		Data:    repositories,
	})
}

// createRepository 创建新仓库
// POST /api/v1/repositories
func (s *Server) createRepository(c *gin.Context) {
	var repo storage.Repository
	if err := c.ShouldBindJSON(&repo); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	if err := s.database.CreateRepository(&repo); err != nil {
		s.logger.WithError(err).Error("Failed to create repository")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to create repository",
		})
		return
	}

	c.JSON(http.StatusCreated, Response{
		Code:    201,
		Message: "Repository created successfully",
		Data:    repo,
	})
}

// getRepository 获取单个仓库
// GET /api/v1/repositories/:id
func (s *Server) getRepository(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Invalid repository ID",
		})
		return
	}

	repo, err := s.database.GetRepository(uint(id))
	if err != nil {
		s.logger.WithError(err).Error("Failed to get repository")
		c.JSON(http.StatusNotFound, Response{
			Code:    404,
			Message: "Repository not found",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Success",
		Data:    repo,
	})
}

// updateRepository 更新仓库
// PUT /api/v1/repositories/:id
func (s *Server) updateRepository(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Invalid repository ID",
		})
		return
	}

	var repo storage.Repository
	if err := c.ShouldBindJSON(&repo); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	repo.ID = uint(id)
	if err := s.database.UpdateRepository(&repo); err != nil {
		s.logger.WithError(err).Error("Failed to update repository")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to update repository",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Repository updated successfully",
		Data:    repo,
	})
}

// deleteRepository 删除仓库
// DELETE /api/v1/repositories/:id
func (s *Server) deleteRepository(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Invalid repository ID",
		})
		return
	}

	if err := s.database.DeleteRepository(uint(id)); err != nil {
		s.logger.WithError(err).Error("Failed to delete repository")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to delete repository",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Repository deleted successfully",
	})
}

// scanRepository 扫描仓库提交记录
// POST /api/v1/repositories/:id/scan
func (s *Server) scanRepository(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Invalid repository ID",
		})
		return
	}

	// 执行扫描
	result, err := s.gitScheduler.ScanRepository(uint(id))
	if err != nil {
		s.logger.WithError(err).Error("Failed to scan repository")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to scan repository: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Repository scan completed",
		Data:    result,
	})
}

// scanAllRepositories 扫描所有仓库
// POST /api/v1/git/scan
func (s *Server) scanAllRepositories(c *gin.Context) {
	results, err := s.gitScheduler.ScanNow()
	if err != nil {
		s.logger.WithError(err).Error("Failed to scan all repositories")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to scan repositories: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "All repositories scan completed",
		Data:    results,
	})
}

// getGitStats 获取Git扫描统计信息
// GET /api/v1/git/stats
func (s *Server) getGitStats(c *gin.Context) {
	stats := s.gitScheduler.GetStats()

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Git statistics retrieved",
		Data:    stats,
	})
}

// getCommits 获取提交记录
// GET /api/v1/commits?repository_id=1&date=2024-01-01
func (s *Server) getCommits(c *gin.Context) {
	// 解析查询参数
	repositoryIDStr := c.Query("repository_id")
	dateStr := c.Query("date")

	if repositoryIDStr == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "repository_id is required",
		})
		return
	}

	repositoryID, err := strconv.ParseUint(repositoryIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Invalid repository_id",
		})
		return
	}

	// 解析日期，默认为今天
	var date time.Time
	if dateStr != "" {
		date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, Response{
				Code:    400,
				Message: "Invalid date format, use YYYY-MM-DD",
			})
			return
		}
	} else {
		date = time.Now()
	}

	// 获取提交记录
	commits, err := s.database.GetCommits(uint(repositoryID), date)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get commits")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to get commits: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Commits retrieved successfully",
		Data:    commits,
	})
}

// getCommitsByDateRange 获取日期范围内的提交记录
// GET /api/v1/commits/range?start=2024-01-01&end=2024-01-07
func (s *Server) getCommitsByDateRange(c *gin.Context) {
	startStr := c.Query("start")
	endStr := c.Query("end")

	if startStr == "" || endStr == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "start and end dates are required",
		})
		return
	}

	start, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Invalid start date format, use YYYY-MM-DD",
		})
		return
	}

	end, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Invalid end date format, use YYYY-MM-DD",
		})
		return
	}

	// 设置结束时间为当天的最后一秒
	end = time.Date(end.Year(), end.Month(), end.Day(), 23, 59, 59, 999999999, end.Location())

	commits, err := s.database.GetCommitsByDateRange(start, end)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get commits by date range")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to get commits: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Commits retrieved successfully",
		Data:    commits,
	})
}

// ========== 工作要点 API ==========

// getWorkPoints 获取工作要点列表
// GET /api/v1/points
func (s *Server) getWorkPoints(c *gin.Context) {
	// 获取日期参数，默认为今天
	dateStr := c.Query("date")
	var date time.Time
	var err error

	if dateStr != "" {
		date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, Response{
				Code:    400,
				Message: "Invalid date format, expected YYYY-MM-DD",
			})
			return
		}
	} else {
		date = time.Now()
	}

	points, err := s.database.GetWorkPoints(date)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get work points")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to get work points",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Success",
		Data:    points,
	})
}

// CreateWorkPointRequest 创建工作要点的请求结构
type CreateWorkPointRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

// createWorkPoint 创建工作要点（带智能分析）
// POST /api/v1/points
func (s *Server) createWorkPoint(c *gin.Context) {
	var req CreateWorkPointRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	// 使用工作要点管理器创建（带智能分析）
	point, err := s.workpointMgr.CreateWorkPoint(req.Title, req.Description, req.Category)
	if err != nil {
		s.logger.WithError(err).Error("Failed to create work point")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to create work point: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, Response{
		Code:    201,
		Message: "Work point created successfully with AI analysis",
		Data:    point,
	})
}

// getWorkPoint 获取单个工作要点（占位符）
func (s *Server) getWorkPoint(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, Response{
		Code:    501,
		Message: "Not implemented yet",
	})
}

// updateWorkPoint 更新工作要点
// PUT /api/v1/points/:id
func (s *Server) updateWorkPoint(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Invalid work point ID",
		})
		return
	}

	var point storage.WorkPoint
	if err := c.ShouldBindJSON(&point); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	point.ID = uint(id)
	if err := s.database.UpdateWorkPoint(&point); err != nil {
		s.logger.WithError(err).Error("Failed to update work point")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to update work point",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Work point updated successfully",
		Data:    point,
	})
}

// deleteWorkPoint 删除工作要点
// DELETE /api/v1/points/:id
func (s *Server) deleteWorkPoint(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Invalid work point ID",
		})
		return
	}

	if err := s.database.DeleteWorkPoint(uint(id)); err != nil {
		s.logger.WithError(err).Error("Failed to delete work point")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to delete work point",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Work point deleted successfully",
	})
}

// analyzeWorkPoint 分析工作要点
// POST /api/v1/points/analyze
func (s *Server) analyzeWorkPoint(c *gin.Context) {
	var req CreateWorkPointRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	// 创建临时工作要点进行分析
	tempPoint := &storage.WorkPoint{
		Title:       req.Title,
		Description: req.Description,
		Category:    req.Category,
	}

	// 执行分析
	analysis := s.workpointMgr.AnalyzeWorkPoint(tempPoint)

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Work point analysis completed",
		Data:    analysis,
	})
}

// getWorkPointsWithAnalysis 获取工作要点并包含分析信息
// GET /api/v1/points/analysis?date=2024-01-01
func (s *Server) getWorkPointsWithAnalysis(c *gin.Context) {
	// 解析日期参数
	dateStr := c.Query("date")
	var date time.Time
	var err error

	if dateStr != "" {
		date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, Response{
				Code:    400,
				Message: "Invalid date format, use YYYY-MM-DD",
			})
			return
		}
	} else {
		date = time.Now()
	}

	analyses, err := s.workpointMgr.GetWorkPointsWithAnalysis(date)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get work points with analysis")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to get work points with analysis: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Work points with analysis retrieved successfully",
		Data:    analyses,
	})
}

// getWorkPointsByCategory 按分类获取工作要点
// GET /api/v1/points/category/:category?date=2024-01-01
func (s *Server) getWorkPointsByCategory(c *gin.Context) {
	category := c.Param("category")
	if category == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Category is required",
		})
		return
	}

	// 解析日期参数
	dateStr := c.Query("date")
	var date time.Time
	var err error

	if dateStr != "" {
		date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, Response{
				Code:    400,
				Message: "Invalid date format, use YYYY-MM-DD",
			})
			return
		}
	} else {
		date = time.Now()
	}

	points, err := s.workpointMgr.GetWorkPointsByCategory(category, date)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get work points by category")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to get work points by category: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Work points retrieved successfully",
		Data:    points,
	})
}

// getCategoryStats 获取分类统计信息
// GET /api/v1/points/stats/category?start=2024-01-01&end=2024-01-07
func (s *Server) getCategoryStats(c *gin.Context) {
	startStr := c.Query("start")
	endStr := c.Query("end")

	// 默认查询最近7天
	end := time.Now()
	start := end.AddDate(0, 0, -7)

	if startStr != "" {
		var err error
		start, err = time.Parse("2006-01-02", startStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, Response{
				Code:    400,
				Message: "Invalid start date format, use YYYY-MM-DD",
			})
			return
		}
	}

	if endStr != "" {
		var err error
		end, err = time.Parse("2006-01-02", endStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, Response{
				Code:    400,
				Message: "Invalid end date format, use YYYY-MM-DD",
			})
			return
		}
	}

	stats, err := s.workpointMgr.GetCategoryStats(start, end)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get category stats")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to get category stats: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Category statistics retrieved successfully",
		Data:    stats,
	})
}

// ========== 其他API占位符 ==========

// ========== 日报管理API ==========

// GenerateReportRequest 生成日报请求
type GenerateReportRequest struct {
	Date     string `json:"date" binding:"required"` // 日期 YYYY-MM-DD
	Template string `json:"template"`                // 模板名称，可选
}

// getDailyReports 获取日报列表
// GET /api/v1/reports?limit=10
func (s *Server) getDailyReports(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	limit := 10

	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}

	reports, err := s.database.GetDailyReports(limit)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get daily reports")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to get daily reports: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Daily reports retrieved successfully",
		Data:    reports,
	})
}

// generateDailyReport 生成日报
// POST /api/v1/reports/generate
func (s *Server) generateDailyReport(c *gin.Context) {
	var req GenerateReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	// 解析日期
	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Invalid date format, use YYYY-MM-DD",
		})
		return
	}

	// 生成日报
	report, err := s.reportGen.GenerateReport(date, req.Template)
	if err != nil {
		s.logger.WithError(err).Error("Failed to generate daily report")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to generate daily report: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Daily report generated successfully",
		Data:    report,
	})
}

// getDailyReport 获取指定日期的日报
// GET /api/v1/reports/:date
func (s *Server) getDailyReport(c *gin.Context) {
	dateStr := c.Param("date")

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Invalid date format, use YYYY-MM-DD",
		})
		return
	}

	report, err := s.database.GetDailyReport(date)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get daily report")
		c.JSON(http.StatusNotFound, Response{
			Code:    404,
			Message: "Daily report not found for date: " + dateStr,
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Daily report retrieved successfully",
		Data:    report,
	})
}

// updateDailyReport 更新日报
// PUT /api/v1/reports/:id
func (s *Server) updateDailyReport(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Invalid report ID",
		})
		return
	}

	var report storage.DailyReport
	if err := c.ShouldBindJSON(&report); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	report.ID = uint(id)

	if err := s.database.UpdateDailyReport(&report); err != nil {
		s.logger.WithError(err).Error("Failed to update daily report")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to update daily report: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Daily report updated successfully",
		Data:    report,
	})
}

// deleteDailyReport 删除日报
// DELETE /api/v1/reports/:id
func (s *Server) deleteDailyReport(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Invalid report ID",
		})
		return
	}

	if err := s.database.DeleteDailyReport(uint(id)); err != nil {
		s.logger.WithError(err).Error("Failed to delete daily report")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to delete daily report: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Daily report deleted successfully",
	})
}

// previewReport 预览日报（不保存）
// GET /api/v1/reports/preview?date=2024-01-01&template=standard
func (s *Server) previewReport(c *gin.Context) {
	dateStr := c.Query("date")
	template := c.Query("template")

	// 默认使用今天的日期
	date := time.Now()
	if dateStr != "" {
		var err error
		date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, Response{
				Code:    400,
				Message: "Invalid date format, use YYYY-MM-DD",
			})
			return
		}
	}

	content, err := s.reportGen.PreviewReport(date, template)
	if err != nil {
		s.logger.WithError(err).Error("Failed to preview report")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to preview report: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Report preview generated successfully",
		Data: gin.H{
			"date":     date.Format("2006-01-02"),
			"template": template,
			"content":  content,
		},
	})
}

// getReportTemplates 获取可用的报告模板
// GET /api/v1/reports/templates
func (s *Server) getReportTemplates(c *gin.Context) {
	templates := s.reportGen.GetAvailableTemplates()

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Available templates retrieved successfully",
		Data:    templates,
	})
}

// getReportData 获取报告数据（用于自定义处理）
// GET /api/v1/reports/data/:date
func (s *Server) getReportData(c *gin.Context) {
	dateStr := c.Param("date")

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Invalid date format, use YYYY-MM-DD",
		})
		return
	}

	data, err := s.reportGen.GetReportData(date)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get report data")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to get report data: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Report data retrieved successfully",
		Data:    data,
	})
}

// sendDailyReport 发送日报
func (s *Server) sendDailyReport(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, Response{
		Code:    501,
		Message: "Not implemented yet",
	})
}

// ========== 飞书集成 API ==========

// getIntegrationStatus 获取集成状态
// GET /api/v1/integrations/status
func (s *Server) getIntegrationStatus(c *gin.Context) {
	status := gin.H{
		"feishu": gin.H{
			"enabled":   s.feishuClient.IsEnabled(),
			"stats":     s.feishuClient.GetIntegrationStats(),
			"scheduler": s.feishuScheduler.GetStats(),
		},
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Integration status retrieved successfully",
		Data:    status,
	})
}

// getFeishuStatus 获取飞书状态
// GET /api/v1/integrations/feishu/status
func (s *Server) getFeishuStatus(c *gin.Context) {
	status := gin.H{
		"enabled":    s.feishuClient.IsEnabled(),
		"stats":      s.feishuClient.GetIntegrationStats(),
		"scheduler":  s.feishuScheduler.GetStats(),
		"is_running": s.feishuScheduler.IsRunning(),
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Feishu status retrieved successfully",
		Data:    status,
	})
}

// testFeishuConnection 测试飞书连接
// POST /api/v1/integrations/feishu/test
func (s *Server) testFeishuConnection(c *gin.Context) {
	ctx := c.Request.Context()

	if err := s.feishuClient.TestConnection(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, Response{
			Code:    503,
			Message: "Feishu connection test failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Feishu connection test successful",
	})
}

// getFeishuStats 获取飞书统计信息
// GET /api/v1/integrations/feishu/stats
func (s *Server) getFeishuStats(c *gin.Context) {
	stats := s.feishuClient.GetIntegrationStats()
	schedulerStats := s.feishuScheduler.GetStats()

	data := gin.H{
		"client":    stats,
		"scheduler": schedulerStats,
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Feishu stats retrieved successfully",
		Data:    data,
	})
}

// SendToFeishuRequest 发送到飞书的请求结构
type SendToFeishuRequest struct {
	MessageType string `json:"message_type" binding:"required"` // text, card, webhook
	Content     string `json:"content" binding:"required"`      // 消息内容
}

// sendToFeishu 发送消息到飞书
// POST /api/v1/integrations/feishu/send
func (s *Server) sendToFeishu(c *gin.Context) {
	var req SendToFeishuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	// 根据消息类型发送
	switch req.MessageType {
	case "webhook":
		// 通过webhook发送
		webhookContent := map[string]interface{}{
			"msg_type": "text",
			"content": map[string]interface{}{
				"text": req.Content,
			},
		}
		if err := s.feishuClient.SendWebhookMessage(ctx, webhookContent); err != nil {
			c.JSON(http.StatusInternalServerError, Response{
				Code:    500,
				Message: "Failed to send webhook message: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, Response{
			Code:    200,
			Message: "Webhook message sent successfully",
		})

	case "text", "card":
		// 通过API发送
		feishuConfig := s.feishuClient.GetConfig()
		if feishuConfig.ChatID == "" {
			c.JSON(http.StatusBadRequest, Response{
				Code:    400,
				Message: "Chat ID not configured for API messages",
			})
			return
		}

		msgType := feishu.MessageTypeText
		if req.MessageType == "card" {
			msgType = feishu.MessageTypeCard
		}

		resp, err := s.feishuClient.SendMessage(ctx, feishuConfig.ChatID, msgType, req.Content)
		if err != nil {
			c.JSON(http.StatusInternalServerError, Response{
				Code:    500,
				Message: "Failed to send message: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, Response{
			Code:    200,
			Message: "Message sent successfully",
			Data: gin.H{
				"message_id": resp.Data.MessageID,
			},
		})

	default:
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Invalid message type, supported: text, card, webhook",
		})
	}
}

// SendReportToFeishuRequest 发送日报到飞书的请求结构
type SendReportToFeishuRequest struct {
	Date        string   `json:"date" binding:"required"` // 日期 YYYY-MM-DD
	Template    string   `json:"template"`                // 模板名称
	MessageType string   `json:"message_type"`            // text, card
	WorkItems   []string `json:"work_items,omitempty"`    // 用户自定义工作成果
}

// sendReportToFeishu 发送日报到飞书
// POST /api/v1/integrations/feishu/send/report
func (s *Server) sendReportToFeishu(c *gin.Context) {
	var req SendReportToFeishuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	// 解析日期
	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Invalid date format, use YYYY-MM-DD",
		})
		return
	}

	// 设置默认值
	if req.Template == "" {
		req.Template = s.config.Report.DefaultTemplate
	}
	if req.MessageType == "" {
		req.MessageType = "text"
	}

	// 映射消息类型
	msgType := feishu.MessageTypeText
	if req.MessageType == "card" {
		msgType = feishu.MessageTypeCard
	}

	// 发送日报
	result := s.feishuScheduler.SendManually(date, req.Template, msgType)

	if !result.Success {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to send report: " + result.Error,
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Report sent successfully",
		Data: gin.H{
			"message_id": result.MessageID,
			"sent_at":    result.SentAt,
			"duration":   result.Duration,
		},
	})
}

// getIntegrationLogs 获取集成日志
// GET /api/v1/integrations/logs
func (s *Server) getIntegrationLogs(c *gin.Context) {
	platform := c.Query("platform") // 平台过滤，可选
	limit := 50                     // 默认限制50条

	if limitStr := c.Query("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	logs, err := s.database.GetIntegrationLogs(platform, limit)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get integration logs")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to get integration logs: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Integration logs retrieved successfully",
		Data: gin.H{
			"logs":     logs,
			"platform": platform,
			"limit":    limit,
			"count":    len(logs),
		},
	})
}

// ========== 调度器管理 API ==========

// getSchedulerStatus 获取调度器状态
// GET /api/v1/integrations/feishu/scheduler/status
func (s *Server) getSchedulerStatus(c *gin.Context) {
	stats := s.feishuScheduler.GetStats()
	isRunning := s.feishuScheduler.IsRunning()

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Scheduler status retrieved successfully",
		Data: gin.H{
			"is_running": isRunning,
			"stats":      stats,
		},
	})
}

// startScheduler 启动调度器
// POST /api/v1/integrations/feishu/scheduler/start
func (s *Server) startScheduler(c *gin.Context) {
	if err := s.feishuScheduler.Start(); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to start scheduler: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Scheduler started successfully",
	})
}

// stopScheduler 停止调度器
// POST /api/v1/integrations/feishu/scheduler/stop
func (s *Server) stopScheduler(c *gin.Context) {
	if err := s.feishuScheduler.Stop(); err != nil {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to stop scheduler: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Scheduler stopped successfully",
	})
}

// testScheduler 测试调度器发送
// POST /api/v1/integrations/feishu/scheduler/test
func (s *Server) testScheduler(c *gin.Context) {
	result := s.feishuScheduler.TestSend()

	if !result.Success {
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Scheduler test failed: " + result.Error,
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Scheduler test successful",
		Data: gin.H{
			"message_id": result.MessageID,
			"sent_at":    result.SentAt,
			"duration":   result.Duration,
		},
	})
}

// ========== 统计信息 API ==========

// getCommitStats 获取提交统计
func (s *Server) getCommitStats(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, Response{
		Code:    501,
		Message: "Not implemented yet",
	})
}

// getReportStats 获取日报统计
func (s *Server) getReportStats(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, Response{
		Code:    501,
		Message: "Not implemented yet",
	})
}

// ========== GitLab集成 API ==========

// getGitLabStatus 获取GitLab集成状态
// GET /api/v1/integrations/gitlab/status
func (s *Server) getGitLabStatus(c *gin.Context) {
	if s.gitlabClient == nil {
		c.JSON(http.StatusOK, Response{
			Code:    200,
			Message: "Success",
			Data: gin.H{
				"enabled":   false,
				"connected": false,
				"message":   "GitLab integration is disabled",
			},
		})
		return
	}

	status := gin.H{
		"enabled":   s.gitlabClient.IsEnabled(),
		"connected": false,
		"message":   "Not tested",
	}

	// 测试连接
	if err := s.gitlabClient.TestConnection(); err != nil {
		status["message"] = "Connection failed: " + err.Error()
	} else {
		status["connected"] = true
		status["message"] = "Connected successfully"
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Success",
		Data:    status,
	})
}

// testGitLabConnection 测试GitLab连接
// POST /api/v1/integrations/gitlab/test
func (s *Server) testGitLabConnection(c *gin.Context) {
	if s.gitlabClient == nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "GitLab integration is not enabled",
		})
		return
	}

	if err := s.gitlabClient.TestConnection(); err != nil {
		s.logger.WithError(err).Error("GitLab connection test failed")
		c.JSON(http.StatusServiceUnavailable, Response{
			Code:    503,
			Message: "GitLab connection test failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "GitLab connection test successful",
	})
}

// getGitLabUser 获取GitLab当前用户信息
// GET /api/v1/integrations/gitlab/user
func (s *Server) getGitLabUser(c *gin.Context) {
	if s.gitlabClient == nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "GitLab integration is not enabled",
		})
		return
	}

	user, err := s.gitlabClient.GetCurrentUser()
	if err != nil {
		s.logger.WithError(err).Error("Failed to get GitLab user")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to get GitLab user: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Success",
		Data: gin.H{
			"id":       user.ID,
			"username": user.Username,
			"name":     user.Name,
			"email":    user.Email,
			"avatar":   user.AvatarURL,
		},
	})
}

// getGitLabProjects 获取GitLab项目列表
// GET /api/v1/integrations/gitlab/projects
func (s *Server) getGitLabProjects(c *gin.Context) {
	if s.gitlabClient == nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "GitLab integration is not enabled",
		})
		return
	}

	projects, err := s.gitlabClient.GetUserProjects()
	if err != nil {
		s.logger.WithError(err).Error("Failed to get GitLab projects")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to get GitLab projects: " + err.Error(),
		})
		return
	}

	// 简化项目信息返回
	var projectList []gin.H
	for _, project := range projects {
		projectList = append(projectList, gin.H{
			"id":             project.ID,
			"name":           project.Name,
			"path":           project.Path,
			"description":    project.Description,
			"default_branch": project.DefaultBranch,
			"visibility":     project.Visibility,
			"last_activity":  project.LastActivityAt,
			"web_url":        project.WebURL,
		})
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Success",
		Data: gin.H{
			"projects": projectList,
			"count":    len(projectList),
		},
	})
}

// getGitLabCommits 获取指定日期的GitLab提交记录
// GET /api/v1/integrations/gitlab/commits/:date
func (s *Server) getGitLabCommits(c *gin.Context) {
	if s.gitlabClient == nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "GitLab integration is not enabled",
		})
		return
	}

	dateStr := c.Param("date")
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Invalid date format, expected YYYY-MM-DD",
		})
		return
	}

	commits, err := s.gitlabClient.GetUserCommits(date)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get GitLab commits")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to get GitLab commits: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Success",
		Data: gin.H{
			"date":    dateStr,
			"commits": commits,
			"count":   len(commits),
		},
	})
}

// syncGitLabCommits 同步指定日期的GitLab提交记录到数据库
// POST /api/v1/integrations/gitlab/sync/:date
func (s *Server) syncGitLabCommits(c *gin.Context) {
	if s.gitlabClient == nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "GitLab integration is not enabled",
		})
		return
	}

	dateStr := c.Param("date")
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Invalid date format, expected YYYY-MM-DD",
		})
		return
	}

	// 获取GitLab提交记录
	commits, err := s.gitlabClient.GetUserCommits(date)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get GitLab commits")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to get GitLab commits: " + err.Error(),
		})
		return
	}

	// 同步到数据库
	if err := s.gitlabClient.SyncCommitsToDatabase(commits); err != nil {
		s.logger.WithError(err).Error("Failed to sync GitLab commits to database")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to sync commits to database: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "GitLab commits synced successfully",
		Data: gin.H{
			"date":         dateStr,
			"synced_count": len(commits),
		},
	})
}

// generateGitLabReport 生成GitLab专用日报
// POST /api/v1/reports/gitlab/generate
func (s *Server) generateGitLabReport(c *gin.Context) {
	var req GenerateReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	// 检查GitLab客户端是否可用
	if s.gitlabClient == nil || !s.gitlabClient.IsEnabled() {
		c.JSON(http.StatusServiceUnavailable, Response{
			Code:    503,
			Message: "GitLab integration is not enabled or configured",
		})
		return
	}

	// 解析日期
	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Invalid date format, use YYYY-MM-DD",
		})
		return
	}

	// 创建包含GitLab客户端的日报生成器
	gitlabReportGen := report.NewGeneratorWithGitLab(
		s.database,
		s.config.Report,
		s.logger.Logger,
		s.gitScheduler,
		s.workpointMgr,
		s.gitlabClient,
	)

	// 使用GitLab专用方法生成日报
	report, err := gitlabReportGen.GenerateGitLabReport(date, req.Template)
	if err != nil {
		s.logger.WithError(err).Error("Failed to generate GitLab daily report")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to generate GitLab daily report: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "GitLab daily report generated successfully",
		Data:    report,
	})
}

// getGitLabReportData 获取GitLab日报数据
// GET /api/v1/reports/gitlab/data/:date
func (s *Server) getGitLabReportData(c *gin.Context) {
	dateStr := c.Param("date")

	// 检查GitLab客户端是否可用
	if s.gitlabClient == nil || !s.gitlabClient.IsEnabled() {
		c.JSON(http.StatusServiceUnavailable, Response{
			Code:    503,
			Message: "GitLab integration is not enabled or configured",
		})
		return
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Invalid date format, use YYYY-MM-DD",
		})
		return
	}

	// 创建包含GitLab客户端的日报生成器
	gitlabReportGen := report.NewGeneratorWithGitLab(
		s.database,
		s.config.Report,
		s.logger.Logger,
		s.gitScheduler,
		s.workpointMgr,
		s.gitlabClient,
	)

	// 收集GitLab日报数据
	data, err := gitlabReportGen.GetReportData(date)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get GitLab report data")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to get GitLab report data: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "GitLab report data retrieved successfully",
		Data:    data,
	})
}

// previewGitLabReport 预览GitLab日报
// GET /api/v1/reports/gitlab/preview
func (s *Server) previewGitLabReport(c *gin.Context) {
	dateStr := c.Query("date")
	template := c.DefaultQuery("template", "standard")

	// 检查GitLab客户端是否可用
	if s.gitlabClient == nil || !s.gitlabClient.IsEnabled() {
		c.JSON(http.StatusServiceUnavailable, Response{
			Code:    503,
			Message: "GitLab integration is not enabled or configured",
		})
		return
	}

	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Invalid date format, use YYYY-MM-DD",
		})
		return
	}

	// 创建包含GitLab客户端的日报生成器
	gitlabReportGen := report.NewGeneratorWithGitLab(
		s.database,
		s.config.Report,
		s.logger.Logger,
		s.gitScheduler,
		s.workpointMgr,
		s.gitlabClient,
	)

	// 预览GitLab日报
	content, err := gitlabReportGen.PreviewReport(date, template)
	if err != nil {
		s.logger.WithError(err).Error("Failed to preview GitLab report")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to preview GitLab report: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "GitLab report preview generated successfully",
		Data: map[string]interface{}{
			"content":  content,
			"date":     dateStr,
			"template": template,
		},
	})
}

// sendGitLabReportToFeishu 推送GitLab日报到飞书
// POST /api/v1/integrations/feishu/send/gitlab-report
func (s *Server) sendGitLabReportToFeishu(c *gin.Context) {
	var req SendReportToFeishuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	// 解析日期
	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Invalid date format, use YYYY-MM-DD",
		})
		return
	}

	// 设置默认值
	if req.Template == "" {
		req.Template = s.config.Report.DefaultTemplate
	}
	if req.MessageType == "" {
		req.MessageType = "text"
	}

	// 检查GitLab集成是否可用
	if s.gitlabClient == nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "GitLab integration is not enabled",
		})
		return
	}

	// 检查飞书集成是否可用
	if !s.feishuClient.IsEnabled() {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Feishu integration is not enabled",
		})
		return
	}

	// 使用GitLab生成器生成日报
	gitlabGenerator := report.NewGeneratorWithGitLab(
		s.database,
		s.config.Report,
		s.logger.Logger,
		s.gitScheduler,
		s.workpointMgr,
		s.gitlabClient,
	)

	// 生成GitLab日报
	_, err = gitlabGenerator.GenerateGitLabReport(date, req.Template)
	if err != nil {
		s.logger.WithError(err).Error("Failed to generate GitLab report")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to generate GitLab report: " + err.Error(),
		})
		return
	}

	// 获取GitLab专用报告数据
	reportData, err := gitlabGenerator.GetGitLabReportData(date)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get GitLab report data")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to get GitLab report data: " + err.Error(),
		})
		return
	}

	// 直接使用飞书客户端发送GitLab报告
	startTime := time.Now()
	msgType := feishu.MessageTypeText
	if req.MessageType == "card" {
		msgType = feishu.MessageTypeCard
	}

	// 格式化消息
	var content string
	switch msgType {
	case feishu.MessageTypeText:
		content, err = s.feishuFormatter.FormatReportAsText(reportData)
		if err != nil {
			s.logger.WithError(err).Error("Failed to format GitLab report as text")
			c.JSON(http.StatusInternalServerError, Response{
				Code:    500,
				Message: "Failed to format GitLab report: " + err.Error(),
			})
			return
		}
	case feishu.MessageTypeCard:
		cardContent, err := s.feishuFormatter.FormatReportAsCard(reportData)
		if err != nil {
			s.logger.WithError(err).Error("Failed to format GitLab report as card")
			c.JSON(http.StatusInternalServerError, Response{
				Code:    500,
				Message: "Failed to format GitLab report: " + err.Error(),
			})
			return
		}
		contentBytes, err := s.feishuFormatter.ToJSON(cardContent)
		if err != nil {
			s.logger.WithError(err).Error("Failed to serialize GitLab report card")
			c.JSON(http.StatusInternalServerError, Response{
				Code:    500,
				Message: "Failed to serialize GitLab report: " + err.Error(),
			})
			return
		}
		content = contentBytes
	}

	// 发送到飞书
	var result feishu.SendResult
	result.SentAt = startTime

	if s.feishuClient.GetConfig().WebhookURL != "" {
		// 使用Webhook发送
		webhook, err := s.feishuFormatter.FormatReportAsWebhook(reportData)
		if err != nil {
			s.logger.WithError(err).Error("Failed to format GitLab report as webhook")
			c.JSON(http.StatusInternalServerError, Response{
				Code:    500,
				Message: "Failed to format GitLab report webhook: " + err.Error(),
			})
			return
		}

		err = s.feishuClient.SendWebhookMessage(c.Request.Context(), webhook)
		if err != nil {
			result.Success = false
			result.Error = err.Error()
		} else {
			result.Success = true
		}
	} else if s.feishuClient.GetConfig().ChatID != "" {
		// 使用API发送
		resp, err := s.feishuClient.SendMessage(c.Request.Context(), s.feishuClient.GetConfig().ChatID, msgType, content)
		if err != nil {
			result.Success = false
			result.Error = err.Error()
		} else {
			result.Success = true
			result.MessageID = resp.Data.MessageID
		}
	} else {
		result.Success = false
		result.Error = "No send target configured (webhook_url or chat_id required)"
	}

	result.Duration = time.Since(startTime).Milliseconds()

	if !result.Success {
		s.logger.WithError(fmt.Errorf(result.Error)).Error("Failed to send GitLab report to Feishu")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to send report: " + result.Error,
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "GitLab report sent to Feishu successfully",
		Data: gin.H{
			"message_id":  result.MessageID,
			"sent_at":     result.SentAt,
			"duration":    result.Duration,
			"data_source": "gitlab",
		},
	})
}

// sendAIGitLabReportToFeishu 推送AI生成的GitLab日报到飞书
// POST /api/v1/integrations/feishu/send/ai-gitlab-report
func (s *Server) sendAIGitLabReportToFeishu(c *gin.Context) {
	var req SendReportToFeishuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	// 解析日期
	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Invalid date format, use YYYY-MM-DD",
		})
		return
	}

	// 设置默认值
	if req.Template == "" {
		req.Template = s.config.Report.DefaultTemplate
	}
	if req.MessageType == "" {
		req.MessageType = "text"
	}

	// 检查GitLab集成是否可用
	if s.gitlabClient == nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "GitLab integration is not enabled",
		})
		return
	}

	// 检查飞书集成是否可用
	if !s.feishuClient.IsEnabled() {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Feishu integration is not enabled",
		})
		return
	}

	// 使用GitLab生成器生成AI日报
	gitlabGenerator := report.NewGeneratorWithGitLab(
		s.database,
		s.config.Report,
		s.logger.Logger,
		s.gitScheduler,
		s.workpointMgr,
		s.gitlabClient,
	)

	// 生成AI GitLab日报
	aiReport, err := gitlabGenerator.GenerateAIGitLabReport(date, req.Template, req.WorkItems)
	if err != nil {
		s.logger.WithError(err).Error("Failed to generate AI GitLab report")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to generate AI GitLab report: " + err.Error(),
		})
		return
	}

	// 直接使用飞书客户端发送AI生成的内容
	startTime := time.Now()
	msgType := feishu.MessageTypeText
	if req.MessageType == "card" {
		msgType = feishu.MessageTypeCard
	}

	// 使用AI生成的内容
	var content string
	switch msgType {
	case feishu.MessageTypeText:
		// 直接使用AI生成的内容
		content = aiReport.Content
	case feishu.MessageTypeCard:
		// 对于卡片消息，我们需要获取原始数据来格式化
		reportData, err := gitlabGenerator.GetGitLabReportData(date)
		if err != nil {
			s.logger.WithError(err).Error("Failed to get GitLab report data for card formatting")
			c.JSON(http.StatusInternalServerError, Response{
				Code:    500,
				Message: "Failed to get GitLab report data: " + err.Error(),
			})
			return
		}

		cardContent, err := s.feishuFormatter.FormatReportAsCard(reportData)
		if err != nil {
			s.logger.WithError(err).Error("Failed to format AI GitLab report as card")
			c.JSON(http.StatusInternalServerError, Response{
				Code:    500,
				Message: "Failed to format AI GitLab report: " + err.Error(),
			})
			return
		}
		contentBytes, err := s.feishuFormatter.ToJSON(cardContent)
		if err != nil {
			s.logger.WithError(err).Error("Failed to serialize AI GitLab report card")
			c.JSON(http.StatusInternalServerError, Response{
				Code:    500,
				Message: "Failed to serialize AI GitLab report: " + err.Error(),
			})
			return
		}
		content = contentBytes
	}

	// 发送到飞书
	var result feishu.SendResult
	result.SentAt = startTime

	if s.feishuClient.GetConfig().WebhookURL != "" {
		// 使用Webhook发送AI生成的内容
		webhook := map[string]interface{}{
			"msg_type": "text",
			"content": map[string]interface{}{
				"text": aiReport.Content,
			},
		}

		err = s.feishuClient.SendWebhookMessage(c.Request.Context(), webhook)
		if err != nil {
			result.Success = false
			result.Error = err.Error()
		} else {
			result.Success = true
		}
	} else if s.feishuClient.GetConfig().ChatID != "" {
		// 使用API发送
		resp, err := s.feishuClient.SendMessage(c.Request.Context(), s.feishuClient.GetConfig().ChatID, msgType, content)
		if err != nil {
			result.Success = false
			result.Error = err.Error()
		} else {
			result.Success = true
			result.MessageID = resp.Data.MessageID
		}
	} else {
		result.Success = false
		result.Error = "No send target configured (webhook_url or chat_id required)"
	}

	result.Duration = time.Since(startTime).Milliseconds()

	if !result.Success {
		s.logger.WithError(fmt.Errorf(result.Error)).Error("Failed to send AI GitLab report to Feishu")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to send AI report: " + result.Error,
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "AI GitLab report sent to Feishu successfully",
		Data: gin.H{
			"message_id":  result.MessageID,
			"sent_at":     result.SentAt,
			"duration":    result.Duration,
			"data_source": "gitlab-ai",
			"ai_enabled":  s.config.Report.AI.Enabled,
		},
	})
}
