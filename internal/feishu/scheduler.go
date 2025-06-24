// Package feishu 定时任务调度器
// 负责自动发送日报的定时任务管理
package feishu

import (
	"context"
	"fmt"
	"sync"
	"time"

	"daily-brief/internal/config"
	"daily-brief/internal/report"
	"daily-brief/internal/storage"

	"github.com/sirupsen/logrus"
)

// SendScheduler 发送调度器
// 管理自动发送日报的定时任务
type SendScheduler struct {
	config       config.ReportConfig  // 报告配置
	database     storage.Database     // 数据库接口
	logger       *logrus.Logger       // 日志器
	client       *Client              // 飞书客户端
	formatter    *MessageFormatter    // 消息格式化器
	reportGen    *report.Generator    // 日报生成器
	
	// 调度控制
	ctx          context.Context      // 上下文
	cancelFunc   context.CancelFunc   // 取消函数
	ticker       *time.Ticker         // 定时器
	running      bool                 // 运行状态
	mutex        sync.RWMutex         // 读写锁
	
	// 统计信息
	stats        SchedulerStats       // 调度器统计信息
}

// SchedulerStats 调度器统计信息
type SchedulerStats struct {
	StartTime         time.Time `json:"start_time"`         // 启动时间
	LastSendTime      time.Time `json:"last_send_time"`     // 上次发送时间
	TotalSendCount    int       `json:"total_send_count"`   // 总发送次数
	SuccessCount      int       `json:"success_count"`      // 成功次数
	FailureCount      int       `json:"failure_count"`      // 失败次数
	LastError         string    `json:"last_error"`         // 最后错误
	NextSendTime      time.Time `json:"next_send_time"`     // 下次发送时间
}

// SendRequest 发送请求
type SendRequest struct {
	Date        time.Time   `json:"date"`         // 日期
	Template    string      `json:"template"`     // 模板
	MessageType MessageType `json:"message_type"` // 消息类型
	Force       bool        `json:"force"`        // 强制发送（忽略重复检查）
}

// SendResult 发送结果
type SendResult struct {
	Success     bool      `json:"success"`      // 是否成功
	MessageID   string    `json:"message_id"`   // 消息ID
	Error       string    `json:"error"`        // 错误信息
	SentAt      time.Time `json:"sent_at"`      // 发送时间
	Duration    int64     `json:"duration"`     // 发送耗时（毫秒）
}

// NewSendScheduler 创建新的发送调度器
func NewSendScheduler(
	cfg config.ReportConfig,
	db storage.Database,
	logger *logrus.Logger,
	client *Client,
	formatter *MessageFormatter,
	reportGen *report.Generator,
) *SendScheduler {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &SendScheduler{
		config:     cfg,
		database:   db,
		logger:     logger,
		client:     client,
		formatter:  formatter,
		reportGen:  reportGen,
		ctx:        ctx,
		cancelFunc: cancel,
		stats: SchedulerStats{
			StartTime: time.Now(),
		},
	}
}

// Start 启动调度器
func (s *SendScheduler) Start() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	if s.running {
		return fmt.Errorf("scheduler is already running")
	}
	
	if !s.config.AutoSend {
		s.logger.Info("Auto send is disabled, scheduler will not start")
		return nil
	}
	
	// 计算下次发送时间
	nextSend, err := s.calculateNextSendTime()
	if err != nil {
		return fmt.Errorf("failed to calculate next send time: %w", err)
	}
	
	s.stats.NextSendTime = nextSend
	s.running = true
	
	// 启动定时器
	go s.run()
	
	s.logger.WithFields(logrus.Fields{
		"next_send_time": nextSend.Format("2006-01-02 15:04:05"),
	}).Info("Send scheduler started")
	
	return nil
}

// Stop 停止调度器
func (s *SendScheduler) Stop() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	if !s.running {
		return nil
	}
	
	s.running = false
	s.cancelFunc()
	
	if s.ticker != nil {
		s.ticker.Stop()
	}
	
	s.logger.Info("Send scheduler stopped")
	return nil
}

// run 运行调度循环
func (s *SendScheduler) run() {
	// 等待到发送时间
	nextSend := s.stats.NextSendTime
	now := time.Now()
	
	if nextSend.After(now) {
		waitDuration := nextSend.Sub(now)
		s.logger.WithField("wait_duration", waitDuration).Info("Waiting for next send time")
		
		select {
		case <-time.After(waitDuration):
			// 时间到了，执行发送
		case <-s.ctx.Done():
			// 被取消
			return
		}
	}
	
	// 创建定时器，每24小时检查一次
	s.ticker = time.NewTicker(24 * time.Hour)
	defer s.ticker.Stop()
	
	for {
		select {
		case <-s.ticker.C:
			s.checkAndSend()
		case <-s.ctx.Done():
			return
		}
	}
}

// checkAndSend 检查并发送日报
func (s *SendScheduler) checkAndSend() {
	now := time.Now()
	
	// 检查是否到了发送时间
	if !s.shouldSend(now) {
		return
	}
	
	// 发送今天的日报
	request := SendRequest{
		Date:        now,
		Template:    s.config.DefaultTemplate,
		MessageType: MessageTypeText,
		Force:       false,
	}
	
	result := s.SendReport(request)
	
	s.mutex.Lock()
	s.stats.TotalSendCount++
	s.stats.LastSendTime = now
	
	if result.Success {
		s.stats.SuccessCount++
		s.stats.LastError = ""
		s.logger.WithFields(logrus.Fields{
			"message_id": result.MessageID,
			"duration":   result.Duration,
		}).Info("Daily report sent successfully")
	} else {
		s.stats.FailureCount++
		s.stats.LastError = result.Error
		s.logger.WithError(fmt.Errorf(result.Error)).Error("Failed to send daily report")
	}
	
	// 计算下次发送时间
	nextSend, err := s.calculateNextSendTime()
	if err != nil {
		s.logger.WithError(err).Error("Failed to calculate next send time")
	} else {
		s.stats.NextSendTime = nextSend
	}
	s.mutex.Unlock()
}

// shouldSend 检查是否应该发送
func (s *SendScheduler) shouldSend(now time.Time) bool {
	// 解析发送时间
	sendTime, err := time.Parse("15:04", s.config.SendTime)
	if err != nil {
		s.logger.WithError(err).Error("Invalid send time format")
		return false
	}
	
	// 构造今天的发送时间
	todaySendTime := time.Date(now.Year(), now.Month(), now.Day(), 
		sendTime.Hour(), sendTime.Minute(), 0, 0, now.Location())
	
	// 检查是否在发送时间窗口内（±5分钟）
	timeDiff := now.Sub(todaySendTime)
	if timeDiff < 0 {
		timeDiff = -timeDiff
	}
	
	return timeDiff <= 5*time.Minute
}

// calculateNextSendTime 计算下次发送时间
func (s *SendScheduler) calculateNextSendTime() (time.Time, error) {
	sendTime, err := time.Parse("15:04", s.config.SendTime)
	if err != nil {
		return time.Time{}, err
	}
	
	now := time.Now()
	nextSend := time.Date(now.Year(), now.Month(), now.Day(), 
		sendTime.Hour(), sendTime.Minute(), 0, 0, now.Location())
	
	// 如果今天的发送时间已过，则设置为明天
	if nextSend.Before(now) {
		nextSend = nextSend.Add(24 * time.Hour)
	}
	
	return nextSend, nil
}

// SendReport 发送日报
func (s *SendScheduler) SendReport(request SendRequest) *SendResult {
	startTime := time.Now()
	result := &SendResult{
		SentAt: startTime,
	}
	
	// 检查飞书客户端是否可用
	if !s.client.IsEnabled() {
		result.Error = "Feishu integration is not enabled"
		return result
	}
	
	// 获取或生成日报数据
	reportData, err := s.getReportData(request.Date, request.Template)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to get report data: %v", err)
		return result
	}
	
	// 格式化消息
	var content string
	switch request.MessageType {
	case MessageTypeText:
		content, err = s.formatter.FormatReportAsText(reportData)
		if err != nil {
			result.Error = fmt.Sprintf("Failed to format message: %v", err)
			return result
		}
	case MessageTypeCard:
		cardContent, err := s.formatter.FormatReportAsCard(reportData)
		if err != nil {
			result.Error = fmt.Sprintf("Failed to format card: %v", err)
			return result
		}
		contentBytes, err := s.formatter.ToJSON(cardContent)
		if err != nil {
			result.Error = fmt.Sprintf("Failed to serialize card: %v", err)
			return result
		}
		content = contentBytes
	default:
		content, err = s.formatter.FormatReportAsText(reportData)
		if err != nil {
			result.Error = fmt.Sprintf("Failed to format message: %v", err)
			return result
		}
	}
	
	// 选择发送方式
	var sendErr error
	if s.client.config.WebhookURL != "" {
		// 使用Webhook发送
		webhook, err := s.formatter.FormatReportAsWebhook(reportData)
		if err != nil {
			result.Error = fmt.Sprintf("Failed to format webhook: %v", err)
			return result
		}
		sendErr = s.client.SendWebhookMessage(s.ctx, webhook)
	} else if s.client.config.ChatID != "" {
		// 使用API发送
		resp, err := s.client.SendMessage(s.ctx, s.client.config.ChatID, request.MessageType, content)
		if err != nil {
			sendErr = err
		} else {
			result.MessageID = resp.Data.MessageID
		}
	} else {
		result.Error = "No send target configured (webhook_url or chat_id required)"
		return result
	}
	
	// 处理发送结果
	if sendErr != nil {
		result.Error = sendErr.Error()
		return result
	}
	
	result.Success = true
	result.Duration = time.Since(startTime).Milliseconds()
	
	// 更新日报状态为已发送
	s.updateReportStatus(request.Date, request.Template)
	
	return result
}

// getReportData 获取报告数据
func (s *SendScheduler) getReportData(date time.Time, template string) (*report.ReportData, error) {
	// 首先尝试从数据库获取已生成的日报
	existingReport, err := s.database.GetDailyReport(date)
	if err == nil && existingReport != nil {
		// 如果已存在，直接获取原始数据
		return s.reportGen.GetReportData(date)
	}
	
	// 如果不存在，生成新的日报
	_, err = s.reportGen.GenerateReport(date, template)
	if err != nil {
		return nil, fmt.Errorf("failed to generate report: %w", err)
	}
	
	// 获取生成的数据
	return s.reportGen.GetReportData(date)
}

// updateReportStatus 更新日报状态
func (s *SendScheduler) updateReportStatus(date time.Time, template string) {
	report, err := s.database.GetDailyReport(date)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get report for status update")
		return
	}
	
	if report != nil {
		report.Status = "sent"
		sentTime := time.Now()
		report.SentAt = &sentTime
		
		if err := s.database.UpdateDailyReport(report); err != nil {
			s.logger.WithError(err).Error("Failed to update report status")
		}
	}
}

// GetStats 获取调度器统计信息
func (s *SendScheduler) GetStats() SchedulerStats {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.stats
}

// IsRunning 检查调度器是否运行中
func (s *SendScheduler) IsRunning() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.running
}

// TestSend 测试发送（用于调试）
func (s *SendScheduler) TestSend() *SendResult {
	request := SendRequest{
		Date:        time.Now(),
		Template:    s.config.DefaultTemplate,
		MessageType: MessageTypeText,
		Force:       true,
	}
	
	return s.SendReport(request)
}

// SendManually 手动发送日报
func (s *SendScheduler) SendManually(date time.Time, template string, msgType MessageType) *SendResult {
	request := SendRequest{
		Date:        date,
		Template:    template,
		MessageType: msgType,
		Force:       true,
	}
	
	result := s.SendReport(request)
	
	// 更新统计信息
	s.mutex.Lock()
	s.stats.TotalSendCount++
	if result.Success {
		s.stats.SuccessCount++
		s.stats.LastError = ""
	} else {
		s.stats.FailureCount++
		s.stats.LastError = result.Error
	}
	s.mutex.Unlock()
	
	return result
}

// Retry 重试发送失败的日报
func (s *SendScheduler) Retry(date time.Time, template string) *SendResult {
	s.logger.WithFields(logrus.Fields{
		"date":     date.Format("2006-01-02"),
		"template": template,
	}).Info("Retrying failed report send")
	
	return s.SendManually(date, template, MessageTypeText)
} 