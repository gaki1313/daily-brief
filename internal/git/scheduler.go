// Package git Git调度器
// 提供定时扫描、批量处理等功能
package git

import (
	"context"
	"sync"
	"time"

	"daily-brief/internal/config"
	"daily-brief/internal/storage"

	"github.com/sirupsen/logrus"
)

// Scheduler Git扫描调度器
// 负责定时扫描所有仓库，管理扫描任务
type Scheduler struct {
	scanner    *Scanner             // Git扫描器
	config     config.GitConfig     // Git配置
	logger     *logrus.Logger       // 日志器
	ticker     *time.Ticker         // 定时器
	stopCh     chan struct{}        // 停止信号
	running    bool                 // 运行状态
	mu         sync.RWMutex         // 读写锁
	lastScan   time.Time            // 最后扫描时间
	scanCount  int64                // 扫描次数
	results    []*ScanResult        // 最近的扫描结果
}

// SchedulerStats 调度器统计信息
type SchedulerStats struct {
	Running     bool          `json:"running"`
	LastScan    time.Time     `json:"last_scan"`
	ScanCount   int64         `json:"scan_count"`
	Interval    time.Duration `json:"interval"`
	NextScan    time.Time     `json:"next_scan"`
	LastResults []*ScanResult `json:"last_results"`
}

// NewScheduler 创建新的Git调度器
func NewScheduler(cfg config.GitConfig, db storage.Database, logger *logrus.Logger) *Scheduler {
	scanner := NewScanner(cfg, db, logger)
	
	return &Scheduler{
		scanner: scanner,
		config:  cfg,
		logger:  logger,
		stopCh:  make(chan struct{}),
		results: make([]*ScanResult, 0),
	}
}

// Start 启动调度器
func (sch *Scheduler) Start(ctx context.Context) error {
	sch.mu.Lock()
	defer sch.mu.Unlock()
	
	if sch.running {
		return nil
	}
	
	interval := sch.config.GetScanInterval()
	sch.ticker = time.NewTicker(interval)
	sch.running = true
	
	sch.logger.WithField("interval", interval).Info("Git scanner scheduler started")
	
	// 启动调度循环
	go sch.run(ctx)
	
	return nil
}

// Stop 停止调度器
func (sch *Scheduler) Stop() error {
	sch.mu.Lock()
	defer sch.mu.Unlock()
	
	if !sch.running {
		return nil
	}
	
	close(sch.stopCh)
	sch.ticker.Stop()
	sch.running = false
	
	sch.logger.Info("Git scanner scheduler stopped")
	
	return nil
}

// run 调度器主循环
func (sch *Scheduler) run(ctx context.Context) {
	// 立即执行一次扫描
	sch.performScan()
	
	for {
		select {
		case <-ctx.Done():
			sch.logger.Info("Git scheduler context cancelled")
			return
		case <-sch.stopCh:
			sch.logger.Info("Git scheduler stop signal received")
			return
		case <-sch.ticker.C:
			sch.performScan()
		}
	}
}

// performScan 执行扫描
func (sch *Scheduler) performScan() {
	sch.logger.Info("Starting scheduled repository scan")
	
	startTime := time.Now()
	results, err := sch.scanner.ScanAllRepositories()
	duration := time.Since(startTime)
	
	sch.mu.Lock()
	sch.lastScan = startTime
	sch.scanCount++
	sch.results = results
	sch.mu.Unlock()
	
	if err != nil {
		sch.logger.WithError(err).Error("Scheduled scan failed")
		return
	}
	
	// 统计扫描结果
	totalRepos := len(results)
	successCount := 0
	totalNewCommits := 0
	
	for _, result := range results {
		if result.Error == nil {
			successCount++
			totalNewCommits += result.NewCommits
		}
	}
	
	sch.logger.WithFields(logrus.Fields{
		"total_repos":       totalRepos,
		"success_count":     successCount,
		"total_new_commits": totalNewCommits,
		"duration":          duration,
		"scan_count":        sch.scanCount,
	}).Info("Scheduled repository scan completed")
	
	// 记录详细的扫描结果
	for _, result := range results {
		if result.Error != nil {
			sch.logger.WithFields(logrus.Fields{
				"repository_name": result.Repository.Name,
				"error":          result.Error,
			}).Error("Repository scan failed")
		} else {
			sch.logger.WithFields(logrus.Fields{
				"repository_name": result.Repository.Name,
				"new_commits":     result.NewCommits,
				"duration":        result.Duration,
			}).Debug("Repository scan successful")
		}
	}
}

// ScanNow 立即执行一次扫描
func (sch *Scheduler) ScanNow() ([]*ScanResult, error) {
	sch.logger.Info("Manual scan triggered")
	
	results, err := sch.scanner.ScanAllRepositories()
	if err != nil {
		return nil, err
	}
	
	sch.mu.Lock()
	sch.lastScan = time.Now()
	sch.scanCount++
	sch.results = results
	sch.mu.Unlock()
	
	return results, nil
}

// ScanRepository 扫描指定仓库
func (sch *Scheduler) ScanRepository(repositoryID uint) (*ScanResult, error) {
	// 获取仓库信息
	repo, err := sch.scanner.database.GetRepository(repositoryID)
	if err != nil {
		return nil, err
	}
	
	return sch.scanner.ScanRepository(repo)
}

// GetStats 获取调度器统计信息
func (sch *Scheduler) GetStats() *SchedulerStats {
	sch.mu.RLock()
	defer sch.mu.RUnlock()
	
	interval := sch.config.GetScanInterval()
	nextScan := sch.lastScan.Add(interval)
	
	return &SchedulerStats{
		Running:     sch.running,
		LastScan:    sch.lastScan,
		ScanCount:   sch.scanCount,
		Interval:    interval,
		NextScan:    nextScan,
		LastResults: sch.results,
	}
}

// GetLastResults 获取最近的扫描结果
func (sch *Scheduler) GetLastResults() []*ScanResult {
	sch.mu.RLock()
	defer sch.mu.RUnlock()
	
	// 返回副本，避免并发修改
	results := make([]*ScanResult, len(sch.results))
	copy(results, sch.results)
	
	return results
}

// IsRunning 检查调度器是否正在运行
func (sch *Scheduler) IsRunning() bool {
	sch.mu.RLock()
	defer sch.mu.RUnlock()
	
	return sch.running
}

// UpdateInterval 更新扫描间隔
func (sch *Scheduler) UpdateInterval(interval time.Duration) error {
	sch.mu.Lock()
	defer sch.mu.Unlock()
	
	if sch.running && sch.ticker != nil {
		sch.ticker.Stop()
		sch.ticker = time.NewTicker(interval)
	}
	
	// 更新配置（注意：这里只是更新内存中的配置）
	sch.config.ScanInterval = interval.String()
	
	sch.logger.WithField("new_interval", interval).Info("Git scan interval updated")
	
	return nil
} 