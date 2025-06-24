// Package git Git仓库操作和数据收集
// 提供Git仓库扫描、提交记录解析、增量同步等功能
package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"daily-brief/internal/config"
	"daily-brief/internal/storage"

	"github.com/sirupsen/logrus"
)

// Scanner Git仓库扫描器
// 负责扫描Git仓库，解析提交记录，并存储到数据库
type Scanner struct {
	config   config.GitConfig     // Git配置
	database storage.Database     // 数据库接口
	logger   *logrus.Logger       // 日志器
}

// ScanResult 扫描结果
type ScanResult struct {
	Repository   *storage.Repository  // 扫描的仓库
	NewCommits   int                  // 新增提交数量
	TotalCommits int                  // 总提交数量
	LastCommit   *storage.Commit      // 最新提交
	Error        error                // 扫描错误
	Duration     time.Duration        // 扫描耗时
}

// CommitInfo Git提交信息（解析前的原始数据）
type CommitInfo struct {
	Hash         string            // 提交哈希
	Author       string            // 作者
	Email        string            // 邮箱
	Date         time.Time         // 提交时间
	Message      string            // 提交信息
	FilesChanged []string          // 变更文件列表
	Stats        CommitStats       // 提交统计
}

// CommitStats 提交统计信息
type CommitStats struct {
	FilesChanged int // 变更文件数
	Additions    int // 新增行数
	Deletions    int // 删除行数
}

// NewScanner 创建新的Git扫描器
func NewScanner(cfg config.GitConfig, db storage.Database, logger *logrus.Logger) *Scanner {
	return &Scanner{
		config:   cfg,
		database: db,
		logger:   logger,
	}
}

// ScanRepository 扫描指定仓库
// 这是主要的扫描入口函数，会扫描仓库的所有提交记录
func (s *Scanner) ScanRepository(repo *storage.Repository) (*ScanResult, error) {
	startTime := time.Now()
	
	result := &ScanResult{
		Repository: repo,
	}
	
	s.logger.WithFields(logrus.Fields{
		"repository_id":   repo.ID,
		"repository_name": repo.Name,
		"repository_path": repo.Path,
	}).Info("Starting repository scan")
	
	// 验证仓库路径
	if !s.isValidGitRepository(repo.Path) {
		result.Error = fmt.Errorf("invalid git repository: %s", repo.Path)
		result.Duration = time.Since(startTime)
		return result, result.Error
	}
	
	// 获取最后一次扫描的提交哈希
	lastCommitHash, err := s.getLastScannedCommit(repo.ID)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to get last scanned commit, performing full scan")
		lastCommitHash = ""
	}
	
	// 获取新的提交记录
	commits, err := s.getCommitsSince(repo, lastCommitHash)
	if err != nil {
		result.Error = fmt.Errorf("failed to get commits: %w", err)
		result.Duration = time.Since(startTime)
		return result, result.Error
	}
	
	// 解析并存储提交记录
	newCommits := 0
	for _, commitInfo := range commits {
		commit := s.convertToStorageCommit(commitInfo, repo.ID)
		
		// 检查提交是否已存在
		if s.commitExists(commit.Hash) {
			continue
		}
		
		// 存储提交记录
		if err := s.database.CreateCommit(commit); err != nil {
			s.logger.WithError(err).WithField("commit_hash", commit.Hash).
				Error("Failed to save commit")
			continue
		}
		
		newCommits++
		result.LastCommit = commit
	}
	
	result.NewCommits = newCommits
	result.TotalCommits = len(commits)
	result.Duration = time.Since(startTime)
	
	s.logger.WithFields(logrus.Fields{
		"repository_id":  repo.ID,
		"new_commits":    newCommits,
		"total_commits":  result.TotalCommits,
		"duration":       result.Duration,
	}).Info("Repository scan completed")
	
	return result, nil
}

// ScanAllRepositories 扫描所有启用的仓库
func (s *Scanner) ScanAllRepositories() ([]*ScanResult, error) {
	repositories, err := s.database.GetRepositories()
	if err != nil {
		return nil, fmt.Errorf("failed to get repositories: %w", err)
	}
	
	var results []*ScanResult
	for _, repo := range repositories {
		if !repo.Enabled {
			s.logger.WithField("repository_name", repo.Name).
				Debug("Skipping disabled repository")
			continue
		}
		
		result, err := s.ScanRepository(&repo)
		if err != nil {
			s.logger.WithError(err).WithField("repository_name", repo.Name).
				Error("Failed to scan repository")
		}
		
		results = append(results, result)
	}
	
	return results, nil
}

// isValidGitRepository 验证是否为有效的Git仓库
func (s *Scanner) isValidGitRepository(path string) bool {
	// 检查目录是否存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	
	// 检查是否为Git仓库
	gitDir := filepath.Join(path, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return false
	}
	
	// 尝试执行git命令验证
	cmd := exec.Command("git", "status")
	cmd.Dir = path
	return cmd.Run() == nil
}

// getLastScannedCommit 获取最后一次扫描的提交哈希
func (s *Scanner) getLastScannedCommit(repositoryID uint) (string, error) {
	commits, err := s.database.GetCommits(repositoryID, time.Now())
	if err != nil {
		return "", err
	}
	
	if len(commits) > 0 {
		return commits[0].Hash, nil
	}
	
	return "", nil
}

// getCommitsSince 获取指定提交之后的所有提交
func (s *Scanner) getCommitsSince(repo *storage.Repository, since string) ([]*CommitInfo, error) {
	args := []string{"log", "--pretty=format:%H|%an|%ae|%at|%s", "--numstat"}
	
	if since != "" {
		args = append(args, fmt.Sprintf("%s..HEAD", since))
	} else {
		// 限制提交数量，避免首次扫描太久
		args = append(args, fmt.Sprintf("--max-count=%d", s.config.MaxCommitsPerDay))
	}
	
	// 添加分支过滤
	if repo.Branch != "" {
		args = append(args, repo.Branch)
	} else {
		args = append(args, s.config.DefaultBranch)
	}
	
	cmd := exec.Command("git", args...)
	cmd.Dir = repo.Path
	
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute git log: %w", err)
	}
	
	return s.parseGitLogOutput(string(output))
}

// parseGitLogOutput 解析git log输出
func (s *Scanner) parseGitLogOutput(output string) ([]*CommitInfo, error) {
	var commits []*CommitInfo
	lines := strings.Split(output, "\n")
	
	var currentCommit *CommitInfo
	var statsSection bool
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		// 解析提交头信息（格式：hash|author|email|timestamp|subject）
		if strings.Contains(line, "|") && !statsSection {
			if currentCommit != nil {
				commits = append(commits, currentCommit)
			}
			
			parts := strings.Split(line, "|")
			if len(parts) >= 5 {
				timestamp, _ := strconv.ParseInt(parts[3], 10, 64)
				
				currentCommit = &CommitInfo{
					Hash:    parts[0],
					Author:  parts[1],
					Email:   parts[2],
					Date:    time.Unix(timestamp, 0),
					Message: strings.Join(parts[4:], "|"), // 合并可能包含|的消息
				}
				statsSection = true
			}
			continue
		}
		
		// 解析统计信息（格式：additions deletions filename）
		if statsSection && currentCommit != nil {
			if matched, _ := regexp.MatchString(`^\d+\s+\d+\s+`, line); matched {
				parts := strings.Fields(line)
				if len(parts) >= 3 {
					additions, _ := strconv.Atoi(parts[0])
					deletions, _ := strconv.Atoi(parts[1])
					filename := strings.Join(parts[2:], " ")
					
					currentCommit.Stats.Additions += additions
					currentCommit.Stats.Deletions += deletions
					currentCommit.Stats.FilesChanged++
					currentCommit.FilesChanged = append(currentCommit.FilesChanged, filename)
				}
			} else {
				// 统计信息结束，准备处理下一个提交
				statsSection = false
			}
		}
	}
	
	// 添加最后一个提交
	if currentCommit != nil {
		commits = append(commits, currentCommit)
	}
	
	return commits, nil
}

// convertToStorageCommit 将Git提交信息转换为存储模型
func (s *Scanner) convertToStorageCommit(info *CommitInfo, repositoryID uint) *storage.Commit {
	commit := &storage.Commit{
		Hash:         info.Hash,
		RepositoryID: repositoryID,
		Author:       info.Author,
		Email:        info.Email,
		Message:      info.Message,
		Date:         info.Date,
		FilesChanged: info.Stats.FilesChanged,
		Additions:    info.Stats.Additions,
		Deletions:    info.Stats.Deletions,
	}
	
	// 推断提交类型
	commit.Type = s.inferCommitType(info.Message)
	
	return commit
}

// inferCommitType 根据提交信息推断提交类型
func (s *Scanner) inferCommitType(message string) string {
	message = strings.ToLower(message)
	
	// 常见的提交类型模式
	patterns := map[string][]string{
		"feature": {"feat:", "feature:", "add", "新增", "添加"},
		"bugfix":  {"fix:", "bug:", "修复", "解决", "bugfix"},
		"docs":    {"docs:", "doc:", "文档", "readme"},
		"style":   {"style:", "格式", "样式"},
		"refactor":{"refactor:", "重构", "优化"},
		"test":    {"test:", "测试", "test"},
		"chore":   {"chore:", "构建", "配置", "依赖"},
		"perf":    {"perf:", "性能", "优化"},
		"ci":      {"ci:", "持续集成", "workflow"},
		"revert":  {"revert:", "回退", "撤销"},
	}
	
	for commitType, keywords := range patterns {
		for _, keyword := range keywords {
			if strings.Contains(message, keyword) {
				return commitType
			}
		}
	}
	
	return "other"
}

// commitExists 检查提交是否已存在
func (s *Scanner) commitExists(hash string) bool {
	_, err := s.database.GetCommitByHash(hash)
	return err == nil
}

// GetRepositoryStats 获取仓库统计信息
func (s *Scanner) GetRepositoryStats(repositoryID uint, days int) (*RepositoryStats, error) {
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -days)
	
	commits, err := s.database.GetCommitsByDateRange(startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get commits: %w", err)
	}
	
	stats := &RepositoryStats{
		RepositoryID: repositoryID,
		Period:       fmt.Sprintf("%d days", days),
		StartDate:    startDate,
		EndDate:      endDate,
	}
	
	// 统计提交信息
	typeCount := make(map[string]int)
	
	for _, commit := range commits {
		if commit.RepositoryID != repositoryID {
			continue
		}
		
		stats.TotalCommits++
		stats.TotalAdditions += commit.Additions
		stats.TotalDeletions += commit.Deletions
		stats.TotalFilesChanged += commit.FilesChanged
		
		typeCount[commit.Type]++
		
		if stats.LastCommitDate.Before(commit.Date) {
			stats.LastCommitDate = commit.Date
		}
	}
	
	stats.CommitsByType = typeCount
	
	return stats, nil
}

// RepositoryStats 仓库统计信息
type RepositoryStats struct {
	RepositoryID      uint              `json:"repository_id"`
	Period            string            `json:"period"`
	StartDate         time.Time         `json:"start_date"`
	EndDate           time.Time         `json:"end_date"`
	TotalCommits      int               `json:"total_commits"`
	TotalAdditions    int               `json:"total_additions"`
	TotalDeletions    int               `json:"total_deletions"`
	TotalFilesChanged int               `json:"total_files_changed"`
	LastCommitDate    time.Time         `json:"last_commit_date"`
	CommitsByType     map[string]int    `json:"commits_by_type"`
} 