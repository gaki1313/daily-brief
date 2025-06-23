// Package server 中间件实现
package server

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// corsMiddleware CORS跨域中间件
// 允许前端应用访问API
func (s *Server) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// loggingMiddleware 请求日志中间件
// 记录所有HTTP请求的详细信息
func (s *Server) loggingMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// 使用结构化日志记录请求信息
		s.logger.WithFields(logrus.Fields{
			"timestamp":   param.TimeStamp.Format(time.RFC3339),
			"status":      param.StatusCode,
			"latency":     param.Latency,
			"client_ip":   param.ClientIP,
			"method":      param.Method,
			"path":        param.Path,
			"user_agent":  param.Request.UserAgent(),
			"error":       param.ErrorMessage,
		}).Info("HTTP Request")

		return ""
	})
}

// authMiddleware 认证中间件（预留）
// 未来可以添加JWT或其他认证机制
func (s *Server) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: 实现认证逻辑
		// 例如验证JWT Token、API Key等
		c.Next()
	}
}

// rateLimitMiddleware 限流中间件（预留）
// 防止API被滥用
func (s *Server) rateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: 实现限流逻辑
		// 例如使用Redis存储请求计数
		c.Next()
	}
} 