// Package report 日报模板实现
// 提供多种格式的日报模板
package report

import (
	"fmt"
	"sort"
	"strings"

	"daily-brief/internal/storage"
	"daily-brief/internal/workpoint"
)

// StandardTemplate 标准日报模板
// 适合大多数场景的平衡模板
type StandardTemplate struct{}

// SimpleTemplate 简洁日报模板
// 简化版本，突出重点
type SimpleTemplate struct{}

// DetailedTemplate 详细日报模板
// 包含完整信息的详细版本
type DetailedTemplate struct{}

// MarkdownTemplate Markdown格式模板
// 适合技术团队的Markdown格式
type MarkdownTemplate struct{}

// NewStandardTemplate 创建标准模板
func NewStandardTemplate() *StandardTemplate {
	return &StandardTemplate{}
}

// NewSimpleTemplate 创建简洁模板
func NewSimpleTemplate() *SimpleTemplate {
	return &SimpleTemplate{}
}

// NewDetailedTemplate 创建详细模板
func NewDetailedTemplate() *DetailedTemplate {
	return &DetailedTemplate{}
}

// NewMarkdownTemplate 创建Markdown模板
func NewMarkdownTemplate() *MarkdownTemplate {
	return &MarkdownTemplate{}
}

// ========== 标准模板实现 ==========

func (t *StandardTemplate) GetName() string {
	return "standard"
}

func (t *StandardTemplate) GetDescription() string {
	return "标准日报模板，适合大多数场景"
}

func (t *StandardTemplate) Generate(data *ReportData) (string, error) {
	var builder strings.Builder
	
	// 标题和日期
	builder.WriteString(fmt.Sprintf("📊 %s\n", data.Title))
	builder.WriteString(fmt.Sprintf("📅 日期：%s\n", data.Date.Format("2006年01月02日 星期一")))
	builder.WriteString(strings.Repeat("=", 50) + "\n\n")
	
	// 概览统计
	builder.WriteString("📈 今日概览\n")
	builder.WriteString(fmt.Sprintf("• 完成任务：%d/%d (%.1f%%)\n", 
		data.Summary.CompletedTasks, 
		data.Summary.TotalTasks,
		float64(data.Summary.CompletedTasks)/float64(data.Summary.TotalTasks)*100))
	builder.WriteString(fmt.Sprintf("• 代码提交：%d 次\n", data.Summary.TotalCommits))
	builder.WriteString(fmt.Sprintf("• 代码变更：+%d -%d 行，%d 文件\n", 
		data.Summary.LinesAdded, data.Summary.LinesDeleted, data.Summary.FilesChanged))
	builder.WriteString(fmt.Sprintf("• 工作时长：%.1f 小时\n", data.Statistics.WorkingHours))
	builder.WriteString(fmt.Sprintf("• 生产力评分：%.1f/100\n\n", data.Summary.ProductivityScore))
	
	// 重点任务
	if len(data.Summary.HighlightTasks) > 0 {
		builder.WriteString("⭐ 重点任务\n")
		for _, task := range data.Summary.HighlightTasks {
			builder.WriteString(fmt.Sprintf("• %s\n", task))
		}
		builder.WriteString("\n")
	}
	
	// 工作要点
	if len(data.WorkPoints) > 0 {
		builder.WriteString("📋 工作要点\n")
		
		// 按优先级排序
		sortedWorkPoints := make([]*workpoint.WorkPointAnalysis, len(data.WorkPoints))
		copy(sortedWorkPoints, data.WorkPoints)
		sort.Slice(sortedWorkPoints, func(i, j int) bool {
			return sortedWorkPoints[i].OriginalPoint.Priority > sortedWorkPoints[j].OriginalPoint.Priority
		})
		
		for _, wp := range sortedWorkPoints {
			status := "⏳"
			if wp.OriginalPoint.Completed {
				status = "✅"
			}
			
			priority := ""
			switch wp.OriginalPoint.Priority {
			case 3:
				priority = " [高]"
			case 2:
				priority = " [中]"
			case 1:
				priority = " [低]"
			}
			
			builder.WriteString(fmt.Sprintf("%s %s%s\n", status, wp.OriginalPoint.Title, priority))
			if wp.OriginalPoint.Description != "" {
				builder.WriteString(fmt.Sprintf("   %s\n", wp.OriginalPoint.Description))
			}
		}
		builder.WriteString("\n")
	}
	
	// Git提交记录
	if len(data.Commits) > 0 {
		builder.WriteString("💻 代码提交\n")
		
		// 按时间排序
		sortedCommits := make([]storage.Commit, len(data.Commits))
		copy(sortedCommits, data.Commits)
		sort.Slice(sortedCommits, func(i, j int) bool {
			return sortedCommits[i].Date.After(sortedCommits[j].Date)
		})
		
		for _, commit := range sortedCommits {
			typeIcon := t.getCommitTypeIcon(commit.Type)
			builder.WriteString(fmt.Sprintf("%s %s (%s)\n", 
				typeIcon, commit.Message, commit.Date.Format("15:04")))
			if commit.FilesChanged > 0 {
				builder.WriteString(fmt.Sprintf("   📊 %d文件 +%d -%d\n", 
					commit.FilesChanged, commit.Additions, commit.Deletions))
			}
		}
		builder.WriteString("\n")
	}
	
	// 分类统计
	if len(data.Statistics.CommitsByType) > 0 || len(data.Statistics.TasksByCategory) > 0 {
		builder.WriteString("📊 分类统计\n")
		
		if len(data.Statistics.CommitsByType) > 0 {
			builder.WriteString("代码提交类型：")
			for commitType, count := range data.Statistics.CommitsByType {
				builder.WriteString(fmt.Sprintf("%s(%d) ", commitType, count))
			}
			builder.WriteString("\n")
		}
		
		if len(data.Statistics.TasksByCategory) > 0 {
			builder.WriteString("任务分类：")
			for category, count := range data.Statistics.TasksByCategory {
				builder.WriteString(fmt.Sprintf("%s(%d) ", category, count))
			}
			builder.WriteString("\n")
		}
		builder.WriteString("\n")
	}
	
	// 生成信息
	builder.WriteString(fmt.Sprintf("📝 报告生成于 %s\n", data.Metadata.GeneratedAt.Format("2006-01-02 15:04:05")))
	
	return builder.String(), nil
}

func (t *StandardTemplate) getCommitTypeIcon(commitType string) string {
	switch commitType {
	case "feature":
		return "✨"
	case "bugfix":
		return "🐛"
	case "docs":
		return "📚"
	case "style":
		return "💄"
	case "refactor":
		return "♻️"
	case "test":
		return "🧪"
	case "chore":
		return "🔧"
	case "perf":
		return "⚡"
	default:
		return "📝"
	}
}

// ========== 简洁模板实现 ==========

func (t *SimpleTemplate) GetName() string {
	return "simple"
}

func (t *SimpleTemplate) GetDescription() string {
	return "简洁日报模板，突出重点信息"
}

func (t *SimpleTemplate) Generate(data *ReportData) (string, error) {
	var builder strings.Builder
	
	// 标题
	builder.WriteString(fmt.Sprintf("%s\n", data.Title))
	builder.WriteString(strings.Repeat("-", 30) + "\n\n")
	
	// 核心数据
	builder.WriteString(fmt.Sprintf("✅ 完成：%d/%d 任务\n", data.Summary.CompletedTasks, data.Summary.TotalTasks))
	builder.WriteString(fmt.Sprintf("💻 提交：%d 次\n", data.Summary.TotalCommits))
	builder.WriteString(fmt.Sprintf("📊 评分：%.1f/100\n\n", data.Summary.ProductivityScore))
	
	// 重点任务
	if len(data.Summary.HighlightTasks) > 0 {
		builder.WriteString("⭐ 重点完成：\n")
		for _, task := range data.Summary.HighlightTasks {
			builder.WriteString(fmt.Sprintf("• %s\n", task))
		}
		builder.WriteString("\n")
	}
	
	// 主要提交
	if len(data.Commits) > 0 {
		builder.WriteString("💻 主要提交：\n")
		for i, commit := range data.Commits {
			if i >= 3 { // 只显示前3个
				break
			}
			builder.WriteString(fmt.Sprintf("• %s\n", commit.Message))
		}
		if len(data.Commits) > 3 {
			builder.WriteString(fmt.Sprintf("• ... 还有 %d 个提交\n", len(data.Commits)-3))
		}
	}
	
	return builder.String(), nil
}

// ========== 详细模板实现 ==========

func (t *DetailedTemplate) GetName() string {
	return "detailed"
}

func (t *DetailedTemplate) GetDescription() string {
	return "详细日报模板，包含完整信息"
}

func (t *DetailedTemplate) Generate(data *ReportData) (string, error) {
	var builder strings.Builder
	
	// 详细标题
	builder.WriteString(fmt.Sprintf("📊 %s\n", data.Title))
	builder.WriteString(fmt.Sprintf("📅 %s (%s)\n", 
		data.Date.Format("2006年01月02日"), 
		data.Date.Format("星期一")))
	builder.WriteString(fmt.Sprintf("🕐 生成时间：%s\n", data.Metadata.GeneratedAt.Format("15:04:05")))
	builder.WriteString(strings.Repeat("=", 60) + "\n\n")
	
	// 详细统计
	builder.WriteString("📈 详细统计\n")
	builder.WriteString(fmt.Sprintf("┌─ 任务完成情况：%d/%d (%.1f%%)\n", 
		data.Summary.CompletedTasks, data.Summary.TotalTasks,
		float64(data.Summary.CompletedTasks)/float64(data.Summary.TotalTasks)*100))
	builder.WriteString(fmt.Sprintf("├─ 代码提交次数：%d\n", data.Summary.TotalCommits))
	builder.WriteString(fmt.Sprintf("├─ 新增代码行数：%d\n", data.Summary.LinesAdded))
	builder.WriteString(fmt.Sprintf("├─ 删除代码行数：%d\n", data.Summary.LinesDeleted))
	builder.WriteString(fmt.Sprintf("├─ 变更文件数量：%d\n", data.Summary.FilesChanged))
	builder.WriteString(fmt.Sprintf("├─ 估算工作时长：%.1f 小时\n", data.Statistics.WorkingHours))
	builder.WriteString(fmt.Sprintf("├─ 效率评分：%.1f/10\n", data.Statistics.EfficiencyScore))
	builder.WriteString(fmt.Sprintf("└─ 生产力评分：%.1f/100\n\n", data.Summary.ProductivityScore))
	
	// 分类详情
	if len(data.Statistics.TasksByCategory) > 0 {
		builder.WriteString("📋 任务分类详情\n")
		for category, count := range data.Statistics.TasksByCategory {
			builder.WriteString(fmt.Sprintf("• %s：%d 个任务\n", category, count))
		}
		builder.WriteString("\n")
	}
	
	if len(data.Statistics.CommitsByType) > 0 {
		builder.WriteString("💻 提交类型详情\n")
		for commitType, count := range data.Statistics.CommitsByType {
			builder.WriteString(fmt.Sprintf("• %s：%d 次提交\n", commitType, count))
		}
		builder.WriteString("\n")
	}
	
	// 完整工作要点列表
	if len(data.WorkPoints) > 0 {
		builder.WriteString("📋 完整工作要点\n")
		for _, wp := range data.WorkPoints {
			status := "⏳ 进行中"
			if wp.OriginalPoint.Completed {
				status = "✅ 已完成"
			}
			
			priority := map[int]string{3: "高", 2: "中", 1: "低", 0: "普通"}[wp.OriginalPoint.Priority]
			
			builder.WriteString(fmt.Sprintf("【%s】%s [优先级:%s]\n", status, wp.OriginalPoint.Title, priority))
			if wp.OriginalPoint.Description != "" {
				builder.WriteString(fmt.Sprintf("    描述：%s\n", wp.OriginalPoint.Description))
			}
			if wp.OriginalPoint.Category != "" {
				builder.WriteString(fmt.Sprintf("    分类：%s\n", wp.OriginalPoint.Category))
			}
			if wp.OriginalPoint.Tags != "" {
				builder.WriteString(fmt.Sprintf("    标签：%s\n", wp.OriginalPoint.Tags))
			}
			builder.WriteString("\n")
		}
	}
	
	// 完整提交记录
	if len(data.Commits) > 0 {
		builder.WriteString("💻 完整提交记录\n")
		for _, commit := range data.Commits {
			builder.WriteString(fmt.Sprintf("[%s] %s\n", commit.Date.Format("15:04"), commit.Message))
			builder.WriteString(fmt.Sprintf("    作者：%s <%s>\n", commit.Author, commit.Email))
			builder.WriteString(fmt.Sprintf("    哈希：%s\n", commit.Hash[:8]))
			builder.WriteString(fmt.Sprintf("    变更：%d文件 +%d -%d\n", 
				commit.FilesChanged, commit.Additions, commit.Deletions))
			builder.WriteString(fmt.Sprintf("    类型：%s\n\n", commit.Type))
		}
	}
	
	// 元数据
	builder.WriteString("ℹ️ 报告元数据\n")
	builder.WriteString(fmt.Sprintf("• 生成器：%s\n", data.Metadata.GeneratedBy))
	builder.WriteString(fmt.Sprintf("• 版本：%s\n", data.Metadata.Version))
	builder.WriteString(fmt.Sprintf("• 模板：%s\n", data.Metadata.Template))
	builder.WriteString(fmt.Sprintf("• 数据源：%s\n", strings.Join(data.Metadata.DataSources, ", ")))
	
	return builder.String(), nil
}

// ========== Markdown模板实现 ==========

func (t *MarkdownTemplate) GetName() string {
	return "markdown"
}

func (t *MarkdownTemplate) GetDescription() string {
	return "Markdown格式模板，适合技术团队"
}

func (t *MarkdownTemplate) Generate(data *ReportData) (string, error) {
	var builder strings.Builder
	
	// Markdown标题
	builder.WriteString(fmt.Sprintf("# %s\n\n", data.Title))
	builder.WriteString(fmt.Sprintf("> 📅 **日期**: %s  \n", data.Date.Format("2006-01-02")))
	builder.WriteString(fmt.Sprintf("> 🕐 **生成时间**: %s  \n", data.Metadata.GeneratedAt.Format("15:04:05")))
	builder.WriteString(fmt.Sprintf("> 📊 **生产力评分**: %.1f/100\n\n", data.Summary.ProductivityScore))
	
	// 概览表格
	builder.WriteString("## 📈 今日概览\n\n")
	builder.WriteString("| 指标 | 数值 |\n")
	builder.WriteString("|------|------|\n")
	builder.WriteString(fmt.Sprintf("| 完成任务 | %d/%d (%.1f%%) |\n", 
		data.Summary.CompletedTasks, data.Summary.TotalTasks,
		float64(data.Summary.CompletedTasks)/float64(data.Summary.TotalTasks)*100))
	builder.WriteString(fmt.Sprintf("| 代码提交 | %d 次 |\n", data.Summary.TotalCommits))
	builder.WriteString(fmt.Sprintf("| 代码变更 | +%d -%d 行 |\n", data.Summary.LinesAdded, data.Summary.LinesDeleted))
	builder.WriteString(fmt.Sprintf("| 变更文件 | %d 个 |\n", data.Summary.FilesChanged))
	builder.WriteString(fmt.Sprintf("| 工作时长 | %.1f 小时 |\n", data.Statistics.WorkingHours))
	builder.WriteString("\n")
	
	// 重点任务
	if len(data.Summary.HighlightTasks) > 0 {
		builder.WriteString("## ⭐ 重点任务\n\n")
		for _, task := range data.Summary.HighlightTasks {
			builder.WriteString(fmt.Sprintf("- **%s**\n", task))
		}
		builder.WriteString("\n")
	}
	
	// 工作要点
	if len(data.WorkPoints) > 0 {
		builder.WriteString("## 📋 工作要点\n\n")
		
		// 按优先级分组
		highPriority := []*workpoint.WorkPointAnalysis{}
		mediumPriority := []*workpoint.WorkPointAnalysis{}
		lowPriority := []*workpoint.WorkPointAnalysis{}
		
		for _, wp := range data.WorkPoints {
			switch wp.OriginalPoint.Priority {
			case 3:
				highPriority = append(highPriority, wp)
			case 2:
				mediumPriority = append(mediumPriority, wp)
			default:
				lowPriority = append(lowPriority, wp)
			}
		}
		
		if len(highPriority) > 0 {
			builder.WriteString("### 🔴 高优先级\n\n")
			for _, wp := range highPriority {
				status := "⏳"
				if wp.OriginalPoint.Completed {
					status = "✅"
				}
				builder.WriteString(fmt.Sprintf("- %s **%s**\n", status, wp.OriginalPoint.Title))
				if wp.OriginalPoint.Description != "" {
					builder.WriteString(fmt.Sprintf("  > %s\n", wp.OriginalPoint.Description))
				}
			}
			builder.WriteString("\n")
		}
		
		if len(mediumPriority) > 0 {
			builder.WriteString("### 🟡 中优先级\n\n")
			for _, wp := range mediumPriority {
				status := "⏳"
				if wp.OriginalPoint.Completed {
					status = "✅"
				}
				builder.WriteString(fmt.Sprintf("- %s **%s**\n", status, wp.OriginalPoint.Title))
			}
			builder.WriteString("\n")
		}
		
		if len(lowPriority) > 0 {
			builder.WriteString("### 🟢 低优先级\n\n")
			for _, wp := range lowPriority {
				status := "⏳"
				if wp.OriginalPoint.Completed {
					status = "✅"
				}
				builder.WriteString(fmt.Sprintf("- %s %s\n", status, wp.OriginalPoint.Title))
			}
			builder.WriteString("\n")
		}
	}
	
	// Git提交记录
	if len(data.Commits) > 0 {
		builder.WriteString("## 💻 Git提交记录\n\n")
		
		// 按类型分组
		commitsByType := make(map[string][]storage.Commit)
		for _, commit := range data.Commits {
			commitType := commit.Type
			if commitType == "" {
				commitType = "other"
			}
			commitsByType[commitType] = append(commitsByType[commitType], commit)
		}
		
		for commitType, commits := range commitsByType {
			icon := t.getCommitTypeIcon(commitType)
			builder.WriteString(fmt.Sprintf("### %s %s\n\n", icon, strings.Title(commitType)))
			
			for _, commit := range commits {
				builder.WriteString(fmt.Sprintf("- **%s** `%s`\n", 
					commit.Message, commit.Date.Format("15:04")))
				if commit.FilesChanged > 0 {
					builder.WriteString(fmt.Sprintf("  - 📊 %d文件 `+%d -%d`\n", 
						commit.FilesChanged, commit.Additions, commit.Deletions))
				}
			}
			builder.WriteString("\n")
		}
	}
	
	// 统计图表
	if len(data.Statistics.CommitsByType) > 0 || len(data.Statistics.TasksByCategory) > 0 {
		builder.WriteString("## 📊 统计分析\n\n")
		
		if len(data.Statistics.CommitsByType) > 0 {
			builder.WriteString("### 提交类型分布\n\n")
			builder.WriteString("| 类型 | 数量 | 占比 |\n")
			builder.WriteString("|------|------|------|\n")
			
			total := 0
			for _, count := range data.Statistics.CommitsByType {
				total += count
			}
			
			for commitType, count := range data.Statistics.CommitsByType {
				percentage := float64(count) / float64(total) * 100
				builder.WriteString(fmt.Sprintf("| %s | %d | %.1f%% |\n", commitType, count, percentage))
			}
			builder.WriteString("\n")
		}
		
		if len(data.Statistics.TasksByCategory) > 0 {
			builder.WriteString("### 任务分类分布\n\n")
			builder.WriteString("| 分类 | 数量 |\n")
			builder.WriteString("|------|------|\n")
			
			for category, count := range data.Statistics.TasksByCategory {
				builder.WriteString(fmt.Sprintf("| %s | %d |\n", category, count))
			}
			builder.WriteString("\n")
		}
	}
	
	// 页脚
	builder.WriteString("---\n")
	builder.WriteString(fmt.Sprintf("*由 %s 自动生成 • %s*\n", 
		data.Metadata.GeneratedBy, data.Metadata.GeneratedAt.Format("2006-01-02 15:04:05")))
	
	return builder.String(), nil
}

func (t *MarkdownTemplate) getCommitTypeIcon(commitType string) string {
	switch commitType {
	case "feature":
		return "✨ 新功能"
	case "bugfix":
		return "🐛 Bug修复"
	case "docs":
		return "📚 文档"
	case "style":
		return "💄 样式"
	case "refactor":
		return "♻️ 重构"
	case "test":
		return "🧪 测试"
	case "chore":
		return "🔧 构建"
	case "perf":
		return "⚡ 性能"
	default:
		return "📝 其他"
	}
} 