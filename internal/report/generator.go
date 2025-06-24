// Package report 提供日报生成功能
// 包括模板系统、内容聚合、格式化输出等
package report

import (
	"context"
	"fmt"
	"strings"
	"time"

	"daily-brief/internal/ai"
	"daily-brief/internal/config"
	"daily-brief/internal/git"
	"daily-brief/internal/gitlab"
	"daily-brief/internal/storage"
	"daily-brief/internal/workpoint"

	"github.com/sirupsen/logrus"
)

// Generator 日报生成器
// 负责收集数据、生成内容、应用模板
type Generator struct {
	database     storage.Database    // 数据库接口
	config       config.ReportConfig // 报告配置
	logger       *logrus.Logger      // 日志器
	gitScheduler *git.Scheduler      // Git调度器（获取统计信息）
	gitlabClient *gitlab.Client      // GitLab客户端
	workpointMgr *workpoint.Manager  // 工作要点管理器
	aiClient     *ai.Client          // AI客户端

	// 模板系统
	templates map[string]Template // 可用模板
}

// Template 日报模板接口
type Template interface {
	GetName() string                           // 获取模板名称
	GetDescription() string                    // 获取模板描述
	Generate(data *ReportData) (string, error) // 生成报告内容
}

// ReportData 日报数据结构
// 包含生成日报所需的所有数据
type ReportData struct {
	Date          time.Time                      `json:"date"`           // 报告日期
	Title         string                         `json:"title"`          // 报告标题
	Summary       *DaySummary                    `json:"summary"`        // 日总结
	Commits       []storage.Commit               `json:"commits"`        // Git提交记录
	GitLabCommits []*storage.Commit              `json:"gitlab_commits"` // GitLab提交记录
	WorkPoints    []*workpoint.WorkPointAnalysis `json:"work_points"`    // 工作要点（带分析）
	Statistics    *ReportStatistics              `json:"statistics"`     // 统计信息
	Categories    map[string][]interface{}       `json:"categories"`     // 按分类组织的内容
	Metadata      *ReportMetadata                `json:"metadata"`       // 元数据
	DataSources   []string                       `json:"data_sources"`   // 数据来源
}

// DaySummary 日总结
type DaySummary struct {
	TotalTasks        int      `json:"total_tasks"`        // 总任务数
	CompletedTasks    int      `json:"completed_tasks"`    // 完成任务数
	TotalCommits      int      `json:"total_commits"`      // 总提交数
	LinesAdded        int      `json:"lines_added"`        // 新增代码行数
	LinesDeleted      int      `json:"lines_deleted"`      // 删除代码行数
	FilesChanged      int      `json:"files_changed"`      // 变更文件数
	ProductivityScore float64  `json:"productivity_score"` // 生产力评分
	HighlightTasks    []string `json:"highlight_tasks"`    // 重点任务
}

// ReportStatistics 报告统计信息
type ReportStatistics struct {
	CommitsByType   map[string]int `json:"commits_by_type"`   // 按类型统计提交
	TasksByCategory map[string]int `json:"tasks_by_category"` // 按分类统计任务
	TasksByPriority map[string]int `json:"tasks_by_priority"` // 按优先级统计任务
	WorkingHours    float64        `json:"working_hours"`     // 工作时间
	EfficiencyScore float64        `json:"efficiency_score"`  // 效率评分
}

// ReportMetadata 报告元数据
type ReportMetadata struct {
	GeneratedAt time.Time `json:"generated_at"` // 生成时间
	GeneratedBy string    `json:"generated_by"` // 生成者
	Template    string    `json:"template"`     // 使用的模板
	Version     string    `json:"version"`      // 版本信息
	DataSources []string  `json:"data_sources"` // 数据源
}

// NewGenerator 创建新的日报生成器
func NewGenerator(
	db storage.Database,
	cfg config.ReportConfig,
	logger *logrus.Logger,
	gitScheduler *git.Scheduler,
	workpointMgr *workpoint.Manager,
) *Generator {
	generator := &Generator{
		database:     db,
		config:       cfg,
		logger:       logger,
		gitScheduler: gitScheduler,
		workpointMgr: workpointMgr,
		templates:    make(map[string]Template),
	}

	// 注册默认模板
	generator.registerDefaultTemplates()

	return generator
}

// NewGeneratorWithGitLab 创建包含GitLab客户端的日报生成器
func NewGeneratorWithGitLab(
	db storage.Database,
	cfg config.ReportConfig,
	logger *logrus.Logger,
	gitScheduler *git.Scheduler,
	workpointMgr *workpoint.Manager,
	gitlabClient *gitlab.Client,
) *Generator {
	g := &Generator{
		database:     db,
		config:       cfg,
		logger:       logger,
		gitScheduler: gitScheduler,
		workpointMgr: workpointMgr,
		gitlabClient: gitlabClient,
		templates:    make(map[string]Template),
	}

	// 初始化AI客户端
	if cfg.AI.Enabled {
		g.aiClient = ai.NewClient(cfg.AI, logger)
		logger.WithFields(logrus.Fields{
			"provider":     cfg.AI.Provider,
			"ollama_url":   cfg.AI.OllamaURL,
			"ollama_model": cfg.AI.OllamaModel,
			"enabled":      cfg.AI.Enabled,
		}).Info("AI client initialized")
	}

	g.registerDefaultTemplates()
	return g
}

// SetGitLabClient 设置GitLab客户端
func (g *Generator) SetGitLabClient(client *gitlab.Client) {
	g.gitlabClient = client
}

// registerDefaultTemplates 注册默认模板
func (g *Generator) registerDefaultTemplates() {
	// 注册标准模板
	g.templates["standard"] = NewStandardTemplate()
	g.templates["simple"] = NewSimpleTemplate()
	g.templates["detailed"] = NewDetailedTemplate()
	g.templates["markdown"] = NewMarkdownTemplate()

	g.logger.Infof("Registered %d report templates", len(g.templates))
}

// GenerateReport 生成指定日期的日报
func (g *Generator) GenerateReport(date time.Time, templateName string) (*storage.DailyReport, error) {
	g.logger.WithFields(logrus.Fields{
		"date":     date.Format("2006-01-02"),
		"template": templateName,
	}).Info("Starting report generation")

	// 收集数据
	data, err := g.collectReportData(date)
	if err != nil {
		return nil, fmt.Errorf("failed to collect report data: %w", err)
	}

	// 选择模板
	if templateName == "" {
		templateName = g.config.DefaultTemplate
	}

	template, exists := g.templates[templateName]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", templateName)
	}

	// 生成内容
	content, err := template.Generate(data)
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	// 创建日报记录
	report := &storage.DailyReport{
		Date:     date,
		Title:    data.Title,
		Content:  content,
		Template: templateName,
		Status:   "draft",
	}

	// 保存到数据库
	if err := g.database.CreateDailyReport(report); err != nil {
		return nil, fmt.Errorf("failed to save report: %w", err)
	}

	g.logger.WithFields(logrus.Fields{
		"report_id": report.ID,
		"date":      date.Format("2006-01-02"),
		"template":  templateName,
		"length":    len(content),
	}).Info("Report generated successfully")

	return report, nil
}

// GenerateGitLabReport 生成GitLab专用日报
func (g *Generator) GenerateGitLabReport(date time.Time, templateName string) (*storage.DailyReport, error) {
	g.logger.WithFields(logrus.Fields{
		"date":     date.Format("2006-01-02"),
		"template": templateName,
		"source":   "gitlab",
	}).Info("Starting GitLab report generation")

	// 强制使用GitLab数据源收集数据
	data, err := g.collectGitLabReportData(date)
	if err != nil {
		return nil, fmt.Errorf("failed to collect GitLab report data: %w", err)
	}

	// 如果没有指定模板，使用GitLab专用模板
	if templateName == "" {
		templateName = "gitlab"
	}

	template, exists := g.templates[templateName]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", templateName)
	}

	// 生成内容
	content, err := template.Generate(data)
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	// 创建日报记录
	report := &storage.DailyReport{
		Date:     date,
		Title:    fmt.Sprintf("GitLab Daily Brief - %s", date.Format("2006年01月02日")),
		Content:  content,
		Template: templateName,
		Status:   "draft",
	}

	// 保存到数据库（创建或更新）
	if err := g.createOrUpdateDailyReport(report); err != nil {
		return nil, fmt.Errorf("failed to save report: %w", err)
	}

	g.logger.WithFields(logrus.Fields{
		"report_id": report.ID,
		"date":      date.Format("2006-01-02"),
		"template":  templateName,
		"source":    "gitlab",
		"length":    len(content),
	}).Info("GitLab report generated successfully")

	return report, nil
}

// GenerateAIGitLabReport 使用AI生成GitLab日报
func (g *Generator) GenerateAIGitLabReport(date time.Time, templateName string, workItems []string) (*storage.DailyReport, error) {
	g.logger.WithFields(logrus.Fields{
		"date":     date.Format("2006-01-02"),
		"template": templateName,
		"source":   "gitlab-ai",
	}).Info("Starting AI-powered GitLab report generation")

	// 强制使用GitLab数据源收集数据
	data, err := g.collectGitLabReportData(date)
	if err != nil {
		return nil, fmt.Errorf("failed to collect GitLab report data: %w", err)
	}

	var content string

	// 如果启用了AI且有提交记录，使用AI生成报告
	if g.aiClient != nil && g.aiClient.IsEnabled() && len(data.Commits) > 0 {
		g.logger.Info("Using AI to generate intelligent report")

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(g.config.AI.Timeout)*time.Second)
		defer cancel()

		aiContent, err := g.aiClient.GenerateReport(ctx, data.Commits, date.Format("2006-01-02"), "enterprise-api", workItems)
		if err != nil {
			g.logger.WithError(err).Warn("AI generation failed, falling back to template")
			// AI失败时回退到模板生成
			content, err = g.generateWithTemplate(data, templateName)
			if err != nil {
				return nil, fmt.Errorf("failed to generate content with template: %w", err)
			}
		} else {
			content = aiContent
			g.logger.WithField("ai_content_length", len(content)).Info("AI report generated successfully")
		}
	} else {
		// 没有AI或没有提交记录时使用模板
		content, err = g.generateWithTemplate(data, templateName)
		if err != nil {
			return nil, fmt.Errorf("failed to generate content with template: %w", err)
		}
	}

	// 创建日报记录
	report := &storage.DailyReport{
		Date:     date,
		Title:    fmt.Sprintf("GitLab AI Daily Brief - %s", date.Format("2006年01月02日")),
		Content:  content,
		Template: templateName,
		Status:   "draft",
	}

	// 保存到数据库（创建或更新）
	if err := g.createOrUpdateDailyReport(report); err != nil {
		return nil, fmt.Errorf("failed to save report: %w", err)
	}

	g.logger.WithFields(logrus.Fields{
		"report_id": report.ID,
		"date":      date.Format("2006-01-02"),
		"template":  templateName,
		"source":    "gitlab-ai",
		"length":    len(content),
		"ai_used":   g.aiClient != nil && g.aiClient.IsEnabled() && len(data.Commits) > 0,
	}).Info("AI GitLab report generated successfully")

	return report, nil
}

// collectReportData 收集生成日报所需的数据
func (g *Generator) collectReportData(date time.Time) (*ReportData, error) {
	data := &ReportData{
		Date:       date,
		Title:      fmt.Sprintf("Daily Brief - %s", date.Format("2006年01月02日")),
		Categories: make(map[string][]interface{}),
		Metadata: &ReportMetadata{
			GeneratedAt: time.Now(),
			GeneratedBy: "Daily Brief Generator",
			Version:     "1.0.0",
			DataSources: []string{"git", "workpoints"},
		},
		DataSources: []string{},
	}

	// 收集本地Git提交记录
	commits, err := g.database.GetCommitsByDateRange(
		time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location()),
		time.Date(date.Year(), date.Month(), date.Day(), 23, 59, 59, 0, date.Location()),
	)
	if err != nil {
		g.logger.WithError(err).Warn("Failed to get commits for report")
		commits = []storage.Commit{}
	} else if len(commits) > 0 {
		data.DataSources = append(data.DataSources, "local_git")
	}

	// 尝试从GitLab获取提交记录
	if g.gitlabClient != nil && g.gitlabClient.IsEnabled() {
		gitlabCommits, err := g.gitlabClient.GetUserCommits(date)
		if err != nil {
			g.logger.WithError(err).Warn("Failed to get GitLab commits for report")
		} else {
			// 合并GitLab提交记录
			for _, gitlabCommit := range gitlabCommits {
				commits = append(commits, *gitlabCommit)
			}
			if len(gitlabCommits) > 0 {
				data.DataSources = append(data.DataSources, "gitlab")
			}
		}
	}

	data.Commits = commits

	// 收集工作要点
	workPoints, err := g.workpointMgr.GetWorkPointsWithAnalysis(date)
	if err != nil {
		g.logger.WithError(err).Warn("Failed to get work points for report")
		workPoints = []*workpoint.WorkPointAnalysis{}
	} else if len(workPoints) > 0 {
		data.DataSources = append(data.DataSources, "workpoints")
	}
	data.WorkPoints = workPoints

	// 生成统计信息
	data.Statistics = g.generateStatistics(commits, workPoints)

	// 生成日总结
	data.Summary = g.generateDaySummary(commits, workPoints, data.Statistics)

	// 按分类组织内容
	g.organizeByCategories(data)

	// 更新元数据
	data.Metadata.DataSources = data.DataSources

	return data, nil
}

// collectGitLabReportData 专门收集GitLab日报数据
func (g *Generator) collectGitLabReportData(date time.Time) (*ReportData, error) {
	data := &ReportData{
		Date:       date,
		Title:      fmt.Sprintf("GitLab Daily Brief - %s", date.Format("2006年01月02日")),
		Categories: make(map[string][]interface{}),
		Metadata: &ReportMetadata{
			GeneratedAt: time.Now(),
			GeneratedBy: "Daily Brief Generator (GitLab)",
			Version:     "1.0.0",
			DataSources: []string{"gitlab"},
		},
		DataSources: []string{"gitlab"},
	}

	// 仅从GitLab获取提交记录（使用超级优化版本）
	var commits []storage.Commit
	if g.gitlabClient != nil && g.gitlabClient.IsEnabled() {
		// 使用超级优化的提交获取方法
		gitlabCommits, err := g.gitlabClient.GetUserCommitsUltraOptimized(date)
		if err != nil {
			return nil, fmt.Errorf("failed to get GitLab commits: %w", err)
		}

		for _, gitlabCommit := range gitlabCommits {
			commits = append(commits, *gitlabCommit)
		}

		g.logger.WithField("commit_count", len(commits)).Info("Retrieved GitLab commits for report")
	} else {
		return nil, fmt.Errorf("GitLab client is not enabled or available")
	}

	data.Commits = commits

	// 收集工作要点（可选）
	workPoints, err := g.workpointMgr.GetWorkPointsWithAnalysis(date)
	if err != nil {
		g.logger.WithError(err).Warn("Failed to get work points for GitLab report")
		workPoints = []*workpoint.WorkPointAnalysis{}
	} else if len(workPoints) > 0 {
		data.DataSources = append(data.DataSources, "workpoints")
	}
	data.WorkPoints = workPoints

	// 生成统计信息
	data.Statistics = g.generateStatistics(commits, workPoints)

	// 生成日总结
	data.Summary = g.generateDaySummary(commits, workPoints, data.Statistics)

	// 按分类组织内容
	g.organizeByCategories(data)

	// 更新元数据
	data.Metadata.DataSources = data.DataSources

	return data, nil
}

// generateStatistics 生成统计信息
func (g *Generator) generateStatistics(commits []storage.Commit, workPoints []*workpoint.WorkPointAnalysis) *ReportStatistics {
	stats := &ReportStatistics{
		CommitsByType:   make(map[string]int),
		TasksByCategory: make(map[string]int),
		TasksByPriority: make(map[string]int),
	}

	// 统计提交类型
	for _, commit := range commits {
		if commit.Type != "" {
			stats.CommitsByType[commit.Type]++
		}
	}

	// 统计工作要点
	for _, wp := range workPoints {
		// 按分类统计
		if wp.OriginalPoint.Category != "" {
			stats.TasksByCategory[wp.OriginalPoint.Category]++
		}

		// 按优先级统计
		priority := g.priorityIntToString(wp.OriginalPoint.Priority)
		stats.TasksByPriority[priority]++
	}

	// 计算效率评分（简单算法）
	totalTasks := len(workPoints)
	totalCommits := len(commits)

	if totalTasks > 0 {
		stats.EfficiencyScore = float64(totalCommits) / float64(totalTasks) * 10
		if stats.EfficiencyScore > 10 {
			stats.EfficiencyScore = 10
		}
	}

	// 估算工作时间（基于提交时间分布）
	stats.WorkingHours = g.estimateWorkingHours(commits)

	return stats
}

// generateDaySummary 生成日总结
func (g *Generator) generateDaySummary(commits []storage.Commit, workPoints []*workpoint.WorkPointAnalysis, stats *ReportStatistics) *DaySummary {
	summary := &DaySummary{
		TotalTasks:     len(workPoints),
		TotalCommits:   len(commits),
		HighlightTasks: []string{},
	}

	// 统计代码变更
	for _, commit := range commits {
		summary.LinesAdded += commit.Additions
		summary.LinesDeleted += commit.Deletions
		summary.FilesChanged += commit.FilesChanged
	}

	// 统计完成任务
	for _, wp := range workPoints {
		if wp.OriginalPoint.Completed {
			summary.CompletedTasks++
		}

		// 提取重点任务（高优先级或紧急标签）
		if wp.OriginalPoint.Priority >= 3 || strings.Contains(wp.OriginalPoint.Tags, "urgent") {
			summary.HighlightTasks = append(summary.HighlightTasks, wp.OriginalPoint.Title)
		}
	}

	// 计算生产力评分
	summary.ProductivityScore = g.calculateProductivityScore(summary, stats)

	return summary
}

// organizeByCategories 按分类组织内容
func (g *Generator) organizeByCategories(data *ReportData) {
	// 按提交类型组织
	for _, commit := range data.Commits {
		category := commit.Type
		if category == "" {
			category = "other"
		}
		data.Categories[category] = append(data.Categories[category], commit)
	}

	// 按工作要点分类组织
	for _, wp := range data.WorkPoints {
		category := wp.OriginalPoint.Category
		if category == "" {
			category = "other"
		}
		data.Categories[category] = append(data.Categories[category], wp)
	}
}

// priorityIntToString 将优先级数字转换为字符串
func (g *Generator) priorityIntToString(priority int) string {
	switch priority {
	case 3:
		return "high"
	case 2:
		return "medium"
	case 1:
		return "low"
	default:
		return "normal"
	}
}

// estimateWorkingHours 估算工作时间
func (g *Generator) estimateWorkingHours(commits []storage.Commit) float64 {
	if len(commits) == 0 {
		return 0
	}

	// 简单算法：基于提交时间分布估算
	var firstCommit, lastCommit time.Time

	for i, commit := range commits {
		if i == 0 {
			firstCommit = commit.Date
			lastCommit = commit.Date
		} else {
			if commit.Date.Before(firstCommit) {
				firstCommit = commit.Date
			}
			if commit.Date.After(lastCommit) {
				lastCommit = commit.Date
			}
		}
	}

	duration := lastCommit.Sub(firstCommit)
	hours := duration.Hours()

	// 限制在合理范围内（0-12小时）
	if hours > 12 {
		hours = 12
	}
	if hours < 0.5 && len(commits) > 0 {
		hours = 0.5 // 至少0.5小时
	}

	return hours
}

// calculateProductivityScore 计算生产力评分
func (g *Generator) calculateProductivityScore(summary *DaySummary, stats *ReportStatistics) float64 {
	score := 0.0

	// 基于完成任务数
	if summary.TotalTasks > 0 {
		completionRate := float64(summary.CompletedTasks) / float64(summary.TotalTasks)
		score += completionRate * 40 // 最高40分
	}

	// 基于提交数量
	if summary.TotalCommits > 0 {
		commitScore := float64(summary.TotalCommits) * 5 // 每个提交5分
		if commitScore > 30 {
			commitScore = 30 // 最高30分
		}
		score += commitScore
	}

	// 基于代码变更量
	if summary.LinesAdded > 0 {
		codeScore := float64(summary.LinesAdded) / 100 * 10 // 每100行10分
		if codeScore > 20 {
			codeScore = 20 // 最高20分
		}
		score += codeScore
	}

	// 基于重点任务
	score += float64(len(summary.HighlightTasks)) * 5 // 每个重点任务5分

	// 确保评分在0-100范围内
	if score > 100 {
		score = 100
	}

	return score
}

// GetAvailableTemplates 获取可用模板列表
func (g *Generator) GetAvailableTemplates() map[string]string {
	templates := make(map[string]string)
	for name, template := range g.templates {
		templates[name] = template.GetDescription()
	}
	return templates
}

// PreviewReport 预览报告（不保存）
func (g *Generator) PreviewReport(date time.Time, templateName string) (string, error) {
	// 收集数据
	data, err := g.collectReportData(date)
	if err != nil {
		return "", fmt.Errorf("failed to collect report data: %w", err)
	}

	// 选择模板
	if templateName == "" {
		templateName = g.config.DefaultTemplate
	}

	template, exists := g.templates[templateName]
	if !exists {
		return "", fmt.Errorf("template not found: %s", templateName)
	}

	// 生成内容
	content, err := template.Generate(data)
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}

	return content, nil
}

// GetReportData 获取报告数据（用于API）- 包含所有数据源
func (g *Generator) GetReportData(date time.Time) (*ReportData, error) {
	return g.collectReportData(date)
}

// GetGitLabReportData 获取GitLab专用报告数据（用于API）- 仅包含GitLab数据
func (g *Generator) GetGitLabReportData(date time.Time) (*ReportData, error) {
	return g.collectGitLabReportData(date)
}

// createOrUpdateDailyReport 创建或更新日报（避免唯一约束冲突）
func (g *Generator) createOrUpdateDailyReport(report *storage.DailyReport) error {
	// 尝试获取现有的日报
	existing, err := g.database.GetDailyReport(report.Date)
	if err != nil {
		// 如果不存在，创建新的
		return g.database.CreateDailyReport(report)
	}

	// 如果存在，更新内容
	existing.Title = report.Title
	existing.Content = report.Content
	existing.Template = report.Template
	existing.Status = report.Status
	existing.UpdatedAt = time.Now()

	return g.database.UpdateDailyReport(existing)
}

// generateWithTemplate 使用模板生成报告内容
func (g *Generator) generateWithTemplate(data *ReportData, templateName string) (string, error) {
	// 如果没有指定模板，使用默认模板
	if templateName == "" {
		templateName = g.config.DefaultTemplate
	}

	template, exists := g.templates[templateName]
	if !exists {
		return "", fmt.Errorf("template not found: %s", templateName)
	}

	// 生成内容
	return template.Generate(data)
}
