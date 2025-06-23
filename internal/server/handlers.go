// Package server API处理器实现
// 提供RESTful API的具体实现
package server

import (
	"net/http"
	"strconv"
	"time"

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

	// TODO: 实现Git扫描逻辑
	s.logger.Infof("Scanning repository ID: %d", id)

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "Repository scan started",
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

// createWorkPoint 创建工作要点
// POST /api/v1/points
func (s *Server) createWorkPoint(c *gin.Context) {
	var point storage.WorkPoint
	if err := c.ShouldBindJSON(&point); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	// 如果没有设置日期，使用今天
	if point.Date.IsZero() {
		point.Date = time.Now()
	}

	if err := s.database.CreateWorkPoint(&point); err != nil {
		s.logger.WithError(err).Error("Failed to create work point")
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "Failed to create work point",
		})
		return
	}

	c.JSON(http.StatusCreated, Response{
		Code:    201,
		Message: "Work point created successfully",
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

// ========== 其他API占位符 ==========

// getDailyReports 获取日报列表
func (s *Server) getDailyReports(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, Response{
		Code:    501,
		Message: "Not implemented yet",
	})
}

// generateDailyReport 生成日报
func (s *Server) generateDailyReport(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, Response{
		Code:    501,
		Message: "Not implemented yet",
	})
}

// getDailyReport 获取指定日期的日报
func (s *Server) getDailyReport(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, Response{
		Code:    501,
		Message: "Not implemented yet",
	})
}

// updateDailyReport 更新日报
func (s *Server) updateDailyReport(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, Response{
		Code:    501,
		Message: "Not implemented yet",
	})
}

// deleteDailyReport 删除日报
func (s *Server) deleteDailyReport(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, Response{
		Code:    501,
		Message: "Not implemented yet",
	})
}

// sendDailyReport 发送日报
func (s *Server) sendDailyReport(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, Response{
		Code:    501,
		Message: "Not implemented yet",
	})
}

// getIntegrationStatus 获取集成状态
func (s *Server) getIntegrationStatus(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, Response{
		Code:    501,
		Message: "Not implemented yet",
	})
}

// sendToFeishu 发送到飞书
func (s *Server) sendToFeishu(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, Response{
		Code:    501,
		Message: "Not implemented yet",
	})
}

// getIntegrationLogs 获取集成日志
func (s *Server) getIntegrationLogs(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, Response{
		Code:    501,
		Message: "Not implemented yet",
	})
}

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