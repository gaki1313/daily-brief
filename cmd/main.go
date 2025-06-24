// Package main 是日报生成器的主程序入口
// 负责初始化各个组件、配置管理、启动HTTP服务器等核心功能
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"daily-brief/internal/config"
	"daily-brief/internal/server"
	"daily-brief/internal/storage"
	"daily-brief/pkg/logger"

	"github.com/gin-gonic/gin"
)

// main 是程序的主入口函数
// 按照以下步骤初始化和启动应用：
// 1. 加载配置文件
// 2. 初始化日志系统
// 3. 初始化数据库连接
// 4. 启动HTTP服务器
// 5. 优雅关闭处理
func main() {
	// 第一步：加载应用配置
	// 配置文件加载失败会导致程序无法启动
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 第二步：初始化日志系统
	// 根据配置设置日志级别、格式和输出位置
	log := logger.New(cfg.Logging)
	log.Info("Starting Daily Brief Generator...")
	log.Infof("Application version: %s", cfg.App.Version)

	// 第三步：初始化数据库
	// 建立数据库连接并执行必要的迁移操作
	db, err := storage.NewDatabase(cfg.Database, log)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	log.Info("Database initialized successfully")

	// 第四步：执行数据库迁移
	// 确保数据库表结构是最新的
	if err := storage.RunMigrations(db.GetDB()); err != nil {
		log.Fatalf("Failed to run database migrations: %v", err)
	}
	log.Info("Database migrations completed successfully")

	// 第五步：设置Gin运行模式
	// 根据配置决定是否启用调试模式
	if cfg.App.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// 第六步：初始化HTTP服务器
	// 创建路由和中间件
	srv := server.New(cfg, db, log)
	
	// 第六点五步：启动后台服务（Git调度器等）
	if err := srv.StartServices(); err != nil {
		log.Fatalf("Failed to start background services: %v", err)
	}
	log.Info("Background services started successfully")
	
	// 第七步：配置HTTP服务器参数
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.App.Port),
		Handler:      srv.Router(),
		ReadTimeout:  30 * time.Second,  // 读取超时时间
		WriteTimeout: 30 * time.Second,  // 写入超时时间
		IdleTimeout:  60 * time.Second,  // 空闲连接超时时间
	}

	// 第八步：启动HTTP服务器（非阻塞）
	go func() {
		log.Infof("Starting HTTP server on port %d", cfg.App.Port)
		log.Infof("Server running in %s mode", cfg.App.Mode)
		log.Infof("Access the application at: http://localhost:%d", cfg.App.Port)
		
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// 第九步：等待中断信号进行优雅关闭
	// 支持的信号：SIGINT (Ctrl+C), SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	
	// 阻塞等待信号
	sig := <-quit
	log.Infof("Received signal: %v, starting graceful shutdown...", sig)

	// 第十步：优雅关闭HTTP服务器
	// 设置30秒的关闭超时时间
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 关闭HTTP服务器，等待现有连接处理完成
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Errorf("HTTP server forced to shutdown: %v", err)
	} else {
		log.Info("HTTP server shutdown gracefully")
	}

	// 第十点五步：停止后台服务
	if err := srv.StopServices(); err != nil {
		log.Errorf("Failed to stop background services: %v", err)
	} else {
		log.Info("Background services stopped gracefully")
	}

	// 第十一步：关闭数据库连接
	if err := db.Close(); err != nil {
		log.Errorf("Failed to close database connection: %v", err)
	} else {
		log.Info("Database connection closed")
	}

	log.Info("Daily Brief Generator stopped")
} 