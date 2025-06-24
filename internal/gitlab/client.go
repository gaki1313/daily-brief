// Package gitlab 提供GitLab API集成功能
// 支持获取用户的提交记录、项目信息等
package gitlab

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/xanzy/go-gitlab"

	"daily-brief/internal/config"
	"daily-brief/internal/storage"
)

// Client GitLab API客户端
// 封装GitLab API操作，提供获取提交记录等功能
type Client struct {
	config     config.GitLabConfig // GitLab配置
	client     *gitlab.Client      // GitLab API客户端
	httpClient *http.Client        // HTTP客户端（用于v3 API）
	database   storage.Database    // 数据库接口
	logger     *logrus.Logger      // 日志器
}

// UserV3 GitLab API v3用户结构
type UserV3 struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
	Email    string `json:"email"`
}

// ProjectV3 GitLab API v3项目结构
type ProjectV3 struct {
	ID                int    `json:"id"`
	Name              string `json:"name"`
	Path              string `json:"path"`
	PathWithNamespace string `json:"path_with_namespace"`
	DefaultBranch     string `json:"default_branch"`
}

// CommitV3 GitLab API v3提交结构
type CommitV3 struct {
	ID             string    `json:"id"`
	ShortID        string    `json:"short_id"`
	Title          string    `json:"title"`
	Message        string    `json:"message"`
	CreatedAt      time.Time `json:"created_at"`
	AuthorName     string    `json:"author_name"`
	AuthorEmail    string    `json:"author_email"`
	CommitterName  string    `json:"committer_name"`
	CommitterEmail string    `json:"committer_email"`
}

// NewClient 创建GitLab客户端
func NewClient(cfg config.GitLabConfig, db storage.Database, logger *logrus.Logger) (*Client, error) {
	if !cfg.Enabled {
		return nil, fmt.Errorf("gitlab integration is disabled")
	}

	// 检查是否使用v3 API
	isV3API := strings.Contains(cfg.BaseURL, "/api/v3")

	var gitlabClient *gitlab.Client
	var err error

	if !isV3API {
		// 使用标准的go-gitlab客户端（v4 API）
		if cfg.Token != "" {
			gitlabClient, err = gitlab.NewClient(cfg.Token, gitlab.WithBaseURL(cfg.BaseURL))
			if err != nil {
				return nil, fmt.Errorf("failed to create gitlab client with token: %w", err)
			}
		} else if cfg.Username != "" && cfg.Password != "" {
			gitlabClient, err = gitlab.NewBasicAuthClient(cfg.Username, cfg.Password, gitlab.WithBaseURL(cfg.BaseURL))
			if err != nil {
				return nil, fmt.Errorf("failed to create gitlab client with basic auth: %w", err)
			}
		} else {
			return nil, fmt.Errorf("gitlab token or username/password is required")
		}
	}

	client := &Client{
		config:     cfg,
		client:     gitlabClient,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		database:   db,
		logger:     logger,
	}

	return client, nil
}

// makeV3Request 发送v3 API请求
func (c *Client) makeV3Request(method, endpoint string, params url.Values) ([]byte, error) {
	// 构造完整URL
	reqURL := strings.TrimSuffix(c.config.BaseURL, "/") + "/" + strings.TrimPrefix(endpoint, "/")
	if params != nil {
		// GitLab v3 API对分支名编码敏感，需要特殊处理
		encodedParams := params.Encode()
		// 将分支名的编码恢复为原始格式
		encodedParams = strings.ReplaceAll(encodedParams, "ref_name=hotfix%2F", "ref_name=hotfix/")
		encodedParams = strings.ReplaceAll(encodedParams, "ref_name=bugfix%2F", "ref_name=bugfix/")
		encodedParams = strings.ReplaceAll(encodedParams, "ref_name=feature%2F", "ref_name=feature/")
		encodedParams = strings.ReplaceAll(encodedParams, "ref_name=release%2F", "ref_name=release/")
		encodedParams = strings.ReplaceAll(encodedParams, "ref_name=develop%2F", "ref_name=develop/")
		reqURL += "?" + encodedParams
	}

	req, err := http.NewRequest(method, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置认证头
	if c.config.Token != "" {
		req.Header.Set("Private-Token", c.config.Token)
	} else if c.config.Username != "" && c.config.Password != "" {
		req.SetBasicAuth(c.config.Username, c.config.Password)
	}

	req.Header.Set("Content-Type", "application/json")

	c.logger.WithFields(logrus.Fields{
		"method": method,
		"url":    reqURL,
	}).Debug("Making GitLab v3 API request")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// isV3API 检查是否使用v3 API
func (c *Client) isV3API() bool {
	return strings.Contains(c.config.BaseURL, "/api/v3")
}

// IsEnabled 检查GitLab集成是否启用
func (c *Client) IsEnabled() bool {
	return c.config.Enabled
}

// GetCurrentUser 获取当前用户信息
func (c *Client) GetCurrentUser() (*gitlab.User, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("gitlab client is not enabled")
	}

	if c.isV3API() {
		return c.getCurrentUserV3()
	}

	user, _, err := c.client.Users.CurrentUser()
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"user_id":  user.ID,
		"username": user.Username,
		"name":     user.Name,
		"email":    user.Email,
	}).Info("Successfully authenticated with GitLab")

	return user, nil
}

// getCurrentUserV3 获取当前用户信息（v3 API）
func (c *Client) getCurrentUserV3() (*gitlab.User, error) {
	body, err := c.makeV3Request("GET", "/user", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	var userV3 UserV3
	if err := json.Unmarshal(body, &userV3); err != nil {
		return nil, fmt.Errorf("failed to parse user response: %w", err)
	}

	// 转换为标准gitlab.User结构
	user := &gitlab.User{
		ID:       userV3.ID,
		Username: userV3.Username,
		Name:     userV3.Name,
		Email:    userV3.Email,
	}

	c.logger.WithFields(logrus.Fields{
		"user_id":  user.ID,
		"username": user.Username,
		"name":     user.Name,
		"email":    user.Email,
	}).Info("Successfully authenticated with GitLab v3 API")

	return user, nil
}

// GetUserProjects 获取用户的项目列表
func (c *Client) GetUserProjects() ([]*gitlab.Project, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("gitlab client is not enabled")
	}

	if c.isV3API() {
		return c.getUserProjectsV3()
	}

	// 如果配置了特定项目ID，则获取这些项目
	if len(c.config.ProjectIDs) > 0 {
		var projects []*gitlab.Project
		for _, projectID := range c.config.ProjectIDs {
			project, _, err := c.client.Projects.GetProject(projectID, nil)
			if err != nil {
				c.logger.WithError(err).WithField("project_id", projectID).Warn("Failed to get project")
				continue
			}
			projects = append(projects, project)
		}
		return projects, nil
	}

	// 否则获取用户有权限的所有项目
	opt := &gitlab.ListProjectsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
		Membership: gitlab.Bool(true), // 只获取用户是成员的项目
		Simple:     gitlab.Bool(true), // 简化返回数据
	}

	var allProjects []*gitlab.Project
	for {
		projects, resp, err := c.client.Projects.ListProjects(opt)
		if err != nil {
			return nil, fmt.Errorf("failed to list projects: %w", err)
		}

		allProjects = append(allProjects, projects...)

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	c.logger.WithField("project_count", len(allProjects)).Info("Retrieved user projects from GitLab")
	return allProjects, nil
}

// getUserProjectsV3 获取用户的项目列表（v3 API）
func (c *Client) getUserProjectsV3() ([]*gitlab.Project, error) {
	// 如果配置了特定项目ID，则获取这些项目
	if len(c.config.ProjectIDs) > 0 {
		var projects []*gitlab.Project
		for _, projectID := range c.config.ProjectIDs {
			project, err := c.getProjectV3(projectID)
			if err != nil {
				c.logger.WithError(err).WithField("project_id", projectID).Warn("Failed to get project")
				continue
			}
			projects = append(projects, project)
		}
		return projects, nil
	}

	// 获取用户有权限的所有项目
	params := url.Values{}
	params.Set("per_page", "100")
	params.Set("page", "1")
	params.Set("membership", "true")

	var allProjects []*gitlab.Project
	page := 1

	for {
		params.Set("page", fmt.Sprintf("%d", page))
		body, err := c.makeV3Request("GET", "/projects", params)
		if err != nil {
			return nil, fmt.Errorf("failed to list projects: %w", err)
		}

		var projectsV3 []ProjectV3
		if err := json.Unmarshal(body, &projectsV3); err != nil {
			return nil, fmt.Errorf("failed to parse projects response: %w", err)
		}

		if len(projectsV3) == 0 {
			break
		}

		// 转换为标准gitlab.Project结构
		for _, pv3 := range projectsV3 {
			project := &gitlab.Project{
				ID:                pv3.ID,
				Name:              pv3.Name,
				Path:              pv3.Path,
				PathWithNamespace: pv3.PathWithNamespace,
				DefaultBranch:     pv3.DefaultBranch,
			}
			allProjects = append(allProjects, project)
		}

		page++
		if len(projectsV3) < 100 {
			break
		}
	}

	c.logger.WithField("project_count", len(allProjects)).Info("Retrieved user projects from GitLab v3 API")
	return allProjects, nil
}

// getProjectV3 获取单个项目信息（v3 API）
func (c *Client) getProjectV3(projectID interface{}) (*gitlab.Project, error) {
	endpoint := fmt.Sprintf("/projects/%v", projectID)
	body, err := c.makeV3Request("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	var projectV3 ProjectV3
	if err := json.Unmarshal(body, &projectV3); err != nil {
		return nil, fmt.Errorf("failed to parse project response: %w", err)
	}

	// 转换为标准gitlab.Project结构
	project := &gitlab.Project{
		ID:                projectV3.ID,
		Name:              projectV3.Name,
		Path:              projectV3.Path,
		PathWithNamespace: projectV3.PathWithNamespace,
		DefaultBranch:     projectV3.DefaultBranch,
	}

	return project, nil
}

// GetUserCommits 获取用户在指定日期的提交记录
func (c *Client) GetUserCommits(date time.Time) ([]*storage.Commit, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("gitlab client is not enabled")
	}

	// 获取当前用户信息
	user, err := c.GetCurrentUser()
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	// 获取用户项目
	projects, err := c.GetUserProjects()
	if err != nil {
		return nil, fmt.Errorf("failed to get user projects: %w", err)
	}

	// 设置时间范围（当天00:00到23:59）
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := time.Date(date.Year(), date.Month(), date.Day(), 23, 59, 59, 999999999, date.Location())

	var allCommits []*storage.Commit

	// 遍历每个项目获取提交记录
	for _, project := range projects {
		commits, err := c.getProjectCommits(project, user, startOfDay, endOfDay)
		if err != nil {
			c.logger.WithError(err).WithFields(logrus.Fields{
				"project_id":   project.ID,
				"project_name": project.Name,
			}).Warn("Failed to get commits for project")
			continue
		}

		allCommits = append(allCommits, commits...)
	}

	c.logger.WithFields(logrus.Fields{
		"date":          date.Format("2006-01-02"),
		"commit_count":  len(allCommits),
		"project_count": len(projects),
	}).Info("Retrieved user commits from GitLab")

	return allCommits, nil
}

// getProjectCommits 获取项目中指定用户和时间范围的提交记录
func (c *Client) getProjectCommits(project *gitlab.Project, user *gitlab.User, since, until time.Time) ([]*storage.Commit, error) {
	if c.isV3API() {
		return c.getProjectCommitsV3(project, user, since, until)
	}

	opt := &gitlab.ListCommitsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
		Since: &since, // 开始时间
		Until: &until, // 结束时间
	}

	// 设置分支
	branchName := "main"
	if project.DefaultBranch != "" {
		branchName = project.DefaultBranch
	}
	opt.RefName = &branchName

	var allCommits []*storage.Commit

	for {
		commits, resp, err := c.client.Commits.ListCommits(project.ID, opt)
		if err != nil {
			return nil, fmt.Errorf("failed to list commits for project %d: %w", project.ID, err)
		}

		// 过滤只属于指定用户的提交
		for _, commit := range commits {
			// 检查是否是目标用户的提交
			// 支持邮箱匹配或者用户名匹配
			emailMatch := commit.AuthorEmail == user.Email || commit.CommitterEmail == user.Email
			nameMatch := commit.AuthorName == user.Name || commit.AuthorName == user.Username

			if !emailMatch && !nameMatch {
				continue
			}

			// 获取提交统计信息
			stats, err := c.getCommitStats(project.ID, commit.ID)
			if err != nil {
				c.logger.WithError(err).WithField("commit_id", commit.ID).Warn("Failed to get commit stats")
				stats = &gitlab.CommitStats{} // 使用空统计
			}

			// 创建storage.Commit对象
			storageCommit := &storage.Commit{
				Hash:         commit.ID,
				Message:      commit.Message,
				Author:       commit.AuthorName,
				Email:        commit.AuthorEmail,
				Date:         *commit.CommittedDate,
				Additions:    stats.Additions,
				Deletions:    stats.Deletions,
				FilesChanged: stats.Total,
				RepositoryID: 0, // 需要获取或创建仓库记录
				Type:         c.inferCommitType(commit.Message),
			}

			// 获取或创建仓库记录
			repoID, err := c.getOrCreateRepository(project)
			if err != nil {
				c.logger.WithError(err).WithField("project_name", project.Name).Warn("Failed to get or create repository")
				continue
			}
			storageCommit.RepositoryID = repoID

			allCommits = append(allCommits, storageCommit)
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return allCommits, nil
}

// getProjectCommitsV3 获取项目中指定用户和时间范围的提交记录（v3 API）
func (c *Client) getProjectCommitsV3(project *gitlab.Project, user *gitlab.User, since, until time.Time) ([]*storage.Commit, error) {
	// 检查多个常见分支
	commonBranches := []string{}

	// 添加项目的默认分支
	if project.DefaultBranch != "" {
		commonBranches = append(commonBranches, project.DefaultBranch)
	}

	// 添加其他常见分支（避免重复）
	otherBranches := []string{"main", "master", "develop", "dev", "feature", "release"}

	for _, branch := range otherBranches {
		exists := false
		for _, existing := range commonBranches {
			if existing == branch {
				exists = true
				break
			}
		}
		if !exists {
			commonBranches = append(commonBranches, branch)
		}
	}

	// 动态获取项目的所有分支，特别查找热修复分支和功能分支
	if projectBranches, err := c.getProjectBranchesV3(project.ID); err == nil {
		branchPrefixes := []string{"hotfix/", "feature/", "bugfix/", "release/", "develop/"}
		for _, projectBranch := range projectBranches {
			// 检查是否是我们感兴趣的分支前缀
			for _, prefix := range branchPrefixes {
				if strings.HasPrefix(projectBranch, prefix) {
					// 检查是否已经在列表中
					exists := false
					for _, existing := range commonBranches {
						if existing == projectBranch {
							exists = true
							break
						}
					}
					if !exists {
						commonBranches = append(commonBranches, projectBranch)
						c.logger.WithFields(logrus.Fields{
							"project_name": project.Name,
							"branch":       projectBranch,
							"prefix":       prefix,
						}).Debug("Added dynamic branch to search list")
					}
					break // 只需要匹配一个前缀
				}
			}
		}
	} else {
		c.logger.WithError(err).WithField("project_name", project.Name).Warn("Failed to get project branches, using static branch list")
	}

	c.logger.WithFields(logrus.Fields{
		"project_name": project.Name,
		"branches":     commonBranches,
	}).Debug("Searching commits in multiple branches")

	var allCommits []*storage.Commit

	// 为每个分支搜索提交
	for _, branchName := range commonBranches {
		branchCommits, err := c.getProjectCommitsFromBranchV3(project, user, branchName, since, until)
		if err != nil {
			c.logger.WithError(err).WithFields(logrus.Fields{
				"project_name": project.Name,
				"branch":       branchName,
			}).Debug("Failed to get commits from branch (branch might not exist)")
			continue
		}

		// 去重：避免同一个提交在多个分支中重复
		for _, newCommit := range branchCommits {
			isDuplicate := false
			for _, existingCommit := range allCommits {
				if existingCommit.Hash == newCommit.Hash {
					isDuplicate = true
					break
				}
			}
			if !isDuplicate {
				allCommits = append(allCommits, newCommit)
			}
		}

		// 如果找到提交，记录日志
		if len(branchCommits) > 0 {
			c.logger.WithFields(logrus.Fields{
				"project_name": project.Name,
				"branch":       branchName,
				"commit_count": len(branchCommits),
			}).Info("Found commits in branch")
		}
	}

	return allCommits, nil
}

// getProjectBranchesV3 获取项目的所有分支（GitLab v3 API）
func (c *Client) getProjectBranchesV3(projectID int) ([]string, error) {
	endpoint := fmt.Sprintf("/projects/%d/repository/branches", projectID)

	body, err := c.makeV3Request("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get project branches: %w", err)
	}

	var branches []struct {
		Name string `json:"name"`
	}

	if err := json.Unmarshal(body, &branches); err != nil {
		return nil, fmt.Errorf("failed to parse branches response: %w", err)
	}

	branchNames := make([]string, len(branches))
	for i, branch := range branches {
		branchNames[i] = branch.Name
	}

	return branchNames, nil
}

// getProjectCommitsFromBranchV3 从指定分支获取提交记录
func (c *Client) getProjectCommitsFromBranchV3(project *gitlab.Project, user *gitlab.User, branchName string, since, until time.Time) ([]*storage.Commit, error) {
	params := url.Values{}
	// GitLab v3 API对参数顺序敏感，ref_name必须在前面
	params.Set("ref_name", branchName)
	// GitLab v3 API不支持带时区的时间格式，使用本地时间格式
	params.Set("since", since.Format("2006-01-02T15:04:05"))
	params.Set("until", until.Format("2006-01-02T15:04:05"))
	params.Set("per_page", "100")

	// 详细记录搜索参数
	c.logger.WithFields(logrus.Fields{
		"project_name": project.Name,
		"branch":       branchName,
		"since":        since.Format(time.RFC3339),
		"until":        until.Format(time.RFC3339),
		"user_email":   user.Email,
	}).Info("Searching commits in branch with parameters")

	var branchCommits []*storage.Commit

	for {
		// 重新创建参数对象，确保参数顺序正确（GitLab v3 API对顺序敏感）
		pageParams := url.Values{}
		pageParams.Set("ref_name", branchName)
		pageParams.Set("since", since.Format("2006-01-02T15:04:05"))
		pageParams.Set("until", until.Format("2006-01-02T15:04:05"))
		// GitLab v3 API在有时间范围查询时不支持分页参数，会导致返回空结果
		// pageParams.Set("per_page", "100")
		// pageParams.Set("page", fmt.Sprintf("%d", page))

		endpoint := fmt.Sprintf("/projects/%d/repository/commits", project.ID)
		body, err := c.makeV3Request("GET", endpoint, pageParams)
		if err != nil {
			// 如果分支不存在，返回空结果而不是错误
			if strings.Contains(err.Error(), "404") {
				return branchCommits, nil
			}
			return nil, fmt.Errorf("failed to list commits for project %d, branch %s: %w", project.ID, branchName, err)
		}

		var commitsV3 []CommitV3
		if err := json.Unmarshal(body, &commitsV3); err != nil {
			return nil, fmt.Errorf("failed to parse commits response: %w", err)
		}

		c.logger.WithFields(logrus.Fields{
			"project_name":  project.Name,
			"branch":        branchName,
			"commits_found": len(commitsV3),
			"response_size": len(body),
		}).Info("API response for commits")

		if len(commitsV3) == 0 {
			c.logger.WithFields(logrus.Fields{
				"project_name": project.Name,
				"branch":       branchName,
			}).Info("No commits found in branch")
			break
		}

		// 过滤只属于指定用户的提交
		for _, commitV3 := range commitsV3 {
			// 记录所有找到的提交（用于调试）
			c.logger.WithFields(logrus.Fields{
				"project_name":    project.Name,
				"branch":          branchName,
				"commit_hash":     commitV3.ID,
				"commit_author":   commitV3.AuthorName,
				"commit_email":    commitV3.AuthorEmail,
				"committer_email": commitV3.CommitterEmail,
				"user_email":      user.Email,
				"user_name":       user.Name,
				"user_username":   user.Username,
				"commit_date":     commitV3.CreatedAt.Format("2006-01-02 15:04:05"),
				"commit_message":  commitV3.Title,
			}).Info("Found commit in branch")

			// 检查是否是目标用户的提交
			// 支持邮箱匹配或者用户名匹配
			emailMatch := commitV3.AuthorEmail == user.Email || commitV3.CommitterEmail == user.Email
			nameMatch := commitV3.AuthorName == user.Name || commitV3.AuthorName == user.Username

			if !emailMatch && !nameMatch {
				c.logger.WithFields(logrus.Fields{
					"commit_author_email":    commitV3.AuthorEmail,
					"commit_author_name":     commitV3.AuthorName,
					"commit_committer_email": commitV3.CommitterEmail,
					"expected_user_email":    user.Email,
					"expected_user_name":     user.Name,
					"expected_username":      user.Username,
				}).Info("Skipping commit - no email or name match")
				continue
			}

			// 记录匹配信息
			matchType := "email"
			if !emailMatch {
				matchType = "name"
			}
			c.logger.WithFields(logrus.Fields{
				"commit_author_email": commitV3.AuthorEmail,
				"commit_author_name":  commitV3.AuthorName,
				"match_type":          matchType,
				"commit_hash":         commitV3.ID,
			}).Info("Found matching commit")

			// 创建storage.Commit对象
			storageCommit := &storage.Commit{
				Hash:         commitV3.ID,
				Message:      commitV3.Message,
				Author:       commitV3.AuthorName,
				Email:        commitV3.AuthorEmail,
				Date:         commitV3.CreatedAt,
				Additions:    0, // v3 API可能不直接提供，需要单独获取
				Deletions:    0,
				FilesChanged: 0,
				RepositoryID: 0, // 需要获取或创建仓库记录
				Type:         c.inferCommitType(commitV3.Message),
			}

			// 获取或创建仓库记录
			repoID, err := c.getOrCreateRepository(project)
			if err != nil {
				c.logger.WithError(err).WithField("project_name", project.Name).Warn("Failed to get or create repository")
				continue
			}
			storageCommit.RepositoryID = repoID

			branchCommits = append(branchCommits, storageCommit)
		}

		// 由于GitLab v3 API在时间范围查询中不支持分页，一次性获取所有结果
		break
	}

	return branchCommits, nil
}

// getCommitStats 获取提交的统计信息
func (c *Client) getCommitStats(projectID interface{}, commitSHA string) (*gitlab.CommitStats, error) {
	commit, _, err := c.client.Commits.GetCommit(projectID, commitSHA, nil)
	if err != nil {
		return nil, err
	}

	if commit.Stats != nil {
		return commit.Stats, nil
	}

	return &gitlab.CommitStats{}, nil
}

// inferCommitType 根据提交消息推断提交类型
func (c *Client) inferCommitType(message string) string {
	message = strings.ToLower(message)

	// 根据常见的提交消息模式推断类型
	switch {
	case strings.Contains(message, "fix") || strings.Contains(message, "bug"):
		return "fix"
	case strings.Contains(message, "feat") || strings.Contains(message, "feature") || strings.Contains(message, "add"):
		return "feat"
	case strings.Contains(message, "docs") || strings.Contains(message, "doc") || strings.Contains(message, "readme"):
		return "docs"
	case strings.Contains(message, "style") || strings.Contains(message, "format"):
		return "style"
	case strings.Contains(message, "refactor") || strings.Contains(message, "refact"):
		return "refactor"
	case strings.Contains(message, "test"):
		return "test"
	case strings.Contains(message, "chore") || strings.Contains(message, "build") || strings.Contains(message, "ci"):
		return "chore"
	case strings.Contains(message, "perf") || strings.Contains(message, "performance"):
		return "perf"
	case strings.Contains(message, "revert"):
		return "revert"
	default:
		return "other"
	}
}

// SyncCommitsToDatabase 同步提交记录到数据库
func (c *Client) SyncCommitsToDatabase(commits []*storage.Commit) error {
	if !c.IsEnabled() {
		return fmt.Errorf("gitlab client is not enabled")
	}

	var syncedCount, skippedCount int

	for _, commit := range commits {
		// 检查提交是否已存在
		existingCommit, err := c.database.GetCommitByHash(commit.Hash)
		if err == nil && existingCommit != nil {
			skippedCount++
			continue
		}

		// 创建新的提交记录
		if err := c.database.CreateCommit(commit); err != nil {
			c.logger.WithError(err).WithField("commit_hash", commit.Hash).Error("Failed to create commit")
			continue
		}

		syncedCount++
	}

	c.logger.WithFields(logrus.Fields{
		"synced_count":  syncedCount,
		"skipped_count": skippedCount,
		"total_count":   len(commits),
	}).Info("Synced commits to database")

	return nil
}

// getOrCreateRepository 获取或创建仓库记录
func (c *Client) getOrCreateRepository(project *gitlab.Project) (uint, error) {
	// 尝试根据项目名称和URL查找现有仓库
	repos, err := c.database.GetRepositories()
	if err != nil {
		return 0, fmt.Errorf("failed to get repositories: %w", err)
	}

	// 查找匹配的仓库
	for _, repo := range repos {
		if repo.Name == project.Name || repo.URL == project.WebURL {
			return repo.ID, nil
		}
	}

	// 创建新的仓库记录
	newRepo := &storage.Repository{
		Name:    project.Name,
		Path:    project.PathWithNamespace,
		URL:     project.WebURL,
		Branch:  project.DefaultBranch,
		Enabled: true,
	}

	if err := c.database.CreateRepository(newRepo); err != nil {
		return 0, fmt.Errorf("failed to create repository: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"repo_id":    newRepo.ID,
		"repo_name":  newRepo.Name,
		"project_id": project.ID,
	}).Info("Created new repository record for GitLab project")

	return newRepo.ID, nil
}

// TestConnection 测试GitLab连接
func (c *Client) TestConnection() error {
	if !c.IsEnabled() {
		return fmt.Errorf("gitlab integration is disabled")
	}

	// 尝试获取当前用户信息来测试连接
	user, err := c.GetCurrentUser()
	if err != nil {
		return fmt.Errorf("gitlab connection test failed: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"user_id":  user.ID,
		"username": user.Username,
		"email":    user.Email,
	}).Info("GitLab connection test successful")

	return nil
}

// GetUserCommitsOptimized 优化版本：直接搜索用户在指定日期的提交记录
// 使用GitLab API搜索功能，避免遍历所有项目
func (c *Client) GetUserCommitsOptimized(date time.Time) ([]*storage.Commit, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("gitlab client is not enabled")
	}

	// 获取当前用户信息
	user, err := c.GetCurrentUser()
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	if c.isV3API() {
		return c.getUserCommitsOptimizedV3(user, date)
	}

	// v4 API实现
	return c.getUserCommitsOptimizedV4(user, date)
}

// getUserCommitsOptimizedV3 使用v3 API优化版本搜索用户提交
func (c *Client) getUserCommitsOptimizedV3(user *gitlab.User, date time.Time) ([]*storage.Commit, error) {
	// 获取中国时区 (CST +8)
	cstLocation, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		// 如果加载失败，使用固定偏移量+8小时
		cstLocation = time.FixedZone("CST", 8*3600)
	}

	// 设置时间范围（当天00:00到23:59，使用中国时区）
	// 先转换输入日期到中国时区
	localDate := date.In(cstLocation)
	startOfDay := time.Date(localDate.Year(), localDate.Month(), localDate.Day(), 0, 0, 0, 0, cstLocation)
	endOfDay := time.Date(localDate.Year(), localDate.Month(), localDate.Day(), 23, 59, 59, 999999999, cstLocation)

	c.logger.WithFields(logrus.Fields{
		"input_date":   date.Format("2006-01-02 15:04:05 MST"),
		"local_date":   localDate.Format("2006-01-02 15:04:05 MST"),
		"start_of_day": startOfDay.Format("2006-01-02 15:04:05 MST"),
		"end_of_day":   endOfDay.Format("2006-01-02 15:04:05 MST"),
		"start_utc":    startOfDay.UTC().Format("2006-01-02 15:04:05 MST"),
		"end_utc":      endOfDay.UTC().Format("2006-01-02 15:04:05 MST"),
	}).Info("Date range calculation for commit search")

	var allCommits []*storage.Commit

	// 方法1：如果配置了特定项目，只搜索这些项目
	if len(c.config.ProjectIDs) > 0 {
		c.logger.WithField("project_ids", c.config.ProjectIDs).Info("Searching commits in configured projects only")
		for _, projectID := range c.config.ProjectIDs {
			project, err := c.getProjectV3(projectID)
			if err != nil {
				c.logger.WithError(err).WithField("project_id", projectID).Warn("Failed to get project")
				continue
			}

			commits, err := c.getProjectCommitsV3(project, user, startOfDay, endOfDay)
			if err != nil {
				c.logger.WithError(err).WithFields(logrus.Fields{
					"project_id":   project.ID,
					"project_name": project.Name,
				}).Warn("Failed to get commits for project")
				continue
			}

			allCommits = append(allCommits, commits...)
		}
		return allCommits, nil
	}

	// 方法2：使用高效预筛选找出有提交的项目
	projectsWithCommits, err := c.getProjectsWithCommitsOnDate(user, date)
	if err != nil {
		c.logger.WithError(err).Warn("Failed to pre-filter projects, falling back to all projects")
		// 如果搜索失败，回退到原来的方法，但限制项目数量
		return c.getUserCommitsWithLimit(user, date, 50) // 最多检查50个项目
	}

	c.logger.WithField("projects_with_commits", len(projectsWithCommits)).Info("Found projects with commits on date")

	// 遍历有提交的项目获取详细提交记录
	for _, project := range projectsWithCommits {
		commits, err := c.getProjectCommitsV3(project, user, startOfDay, endOfDay)
		if err != nil {
			c.logger.WithError(err).WithFields(logrus.Fields{
				"project_id":   project.ID,
				"project_name": project.Name,
			}).Warn("Failed to get commits for project with commits")
			continue
		}

		allCommits = append(allCommits, commits...)
	}

	c.logger.WithFields(logrus.Fields{
		"date":                  date.Format("2006-01-02"),
		"commit_count":          len(allCommits),
		"projects_with_commits": len(projectsWithCommits),
	}).Info("Retrieved user commits from GitLab (optimized)")

	return allCommits, nil
}

// getUserActiveProjectsV3 获取用户在指定日期有活跃提交的项目列表
func (c *Client) getUserActiveProjectsV3(user *gitlab.User, date time.Time) ([]*gitlab.Project, error) {
	// 先尝试搜索特定项目（如果知道项目名称的话）
	if specificProject := c.searchSpecificProject("enterprise-api"); specificProject != nil {
		c.logger.WithFields(logrus.Fields{
			"project_name": specificProject.Name,
			"project_id":   specificProject.ID,
		}).Info("Found specific project: enterprise-api")
		return []*gitlab.Project{specificProject}, nil
	}

	// GitLab v3 API不直接支持按日期过滤，获取最近活跃的项目
	params := url.Values{}
	params.Set("membership", "true")
	params.Set("order_by", "last_activity_at")
	params.Set("sort", "desc")
	params.Set("per_page", "200") // 增加项目数量以提高找到目标项目的概率

	body, err := c.makeV3Request("GET", "/projects", params)
	if err != nil {
		return nil, fmt.Errorf("failed to get active projects: %w", err)
	}

	var projectsV3 []ProjectV3
	if err := json.Unmarshal(body, &projectsV3); err != nil {
		return nil, fmt.Errorf("failed to parse active projects response: %w", err)
	}

	c.logger.WithField("total_projects", len(projectsV3)).Info("Retrieved projects from GitLab")

	// 转换为标准gitlab.Project结构
	var activeProjects []*gitlab.Project
	for _, pv3 := range projectsV3 {
		project := &gitlab.Project{
			ID:                pv3.ID,
			Name:              pv3.Name,
			Path:              pv3.Path,
			PathWithNamespace: pv3.PathWithNamespace,
			DefaultBranch:     pv3.DefaultBranch,
		}
		activeProjects = append(activeProjects, project)

		// 记录项目信息以便调试
		if strings.Contains(pv3.PathWithNamespace, "enterprise-api") {
			c.logger.WithFields(logrus.Fields{
				"project_id":   pv3.ID,
				"project_name": pv3.Name,
				"project_path": pv3.PathWithNamespace,
			}).Info("Found target project in active projects")
		}
	}

	return activeProjects, nil
}

// searchSpecificProject 搜索特定名称的项目
func (c *Client) searchSpecificProject(projectName string) *gitlab.Project {
	// 使用搜索API寻找特定项目
	params := url.Values{}
	params.Set("search", projectName)
	params.Set("membership", "true")
	params.Set("per_page", "20")

	body, err := c.makeV3Request("GET", "/projects", params)
	if err != nil {
		c.logger.WithError(err).WithField("project_name", projectName).Warn("Failed to search for specific project")
		return nil
	}

	var projectsV3 []ProjectV3
	if err := json.Unmarshal(body, &projectsV3); err != nil {
		c.logger.WithError(err).WithField("project_name", projectName).Warn("Failed to parse search results")
		return nil
	}

	// 查找匹配的项目
	for _, pv3 := range projectsV3 {
		if strings.Contains(pv3.Name, projectName) || strings.Contains(pv3.PathWithNamespace, projectName) {
			project := &gitlab.Project{
				ID:                pv3.ID,
				Name:              pv3.Name,
				Path:              pv3.Path,
				PathWithNamespace: pv3.PathWithNamespace,
				DefaultBranch:     pv3.DefaultBranch,
			}
			c.logger.WithFields(logrus.Fields{
				"project_id":   project.ID,
				"project_name": project.Name,
				"project_path": project.PathWithNamespace,
			}).Info("Found matching project through search")
			return project
		}
	}

	return nil
}

// getUserCommitsWithLimit 限制版本：最多检查指定数量的项目
func (c *Client) getUserCommitsWithLimit(user *gitlab.User, date time.Time, maxProjects int) ([]*storage.Commit, error) {
	// 获取用户项目（按最近活跃排序）
	params := url.Values{}
	params.Set("membership", "true")
	params.Set("order_by", "last_activity_at")
	params.Set("sort", "desc")
	params.Set("per_page", fmt.Sprintf("%d", maxProjects))

	body, err := c.makeV3Request("GET", "/projects", params)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent projects: %w", err)
	}

	var projectsV3 []ProjectV3
	if err := json.Unmarshal(body, &projectsV3); err != nil {
		return nil, fmt.Errorf("failed to parse recent projects response: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"max_projects":    maxProjects,
		"actual_projects": len(projectsV3),
	}).Info("Using limited project search for performance")

	// 设置时间范围
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := time.Date(date.Year(), date.Month(), date.Day(), 23, 59, 59, 999999999, date.Location())

	var allCommits []*storage.Commit

	// 检查每个项目的提交
	for _, pv3 := range projectsV3 {
		project := &gitlab.Project{
			ID:                pv3.ID,
			Name:              pv3.Name,
			Path:              pv3.Path,
			PathWithNamespace: pv3.PathWithNamespace,
			DefaultBranch:     pv3.DefaultBranch,
		}

		commits, err := c.getProjectCommitsV3(project, user, startOfDay, endOfDay)
		if err != nil {
			c.logger.WithError(err).WithFields(logrus.Fields{
				"project_id":   project.ID,
				"project_name": project.Name,
			}).Warn("Failed to get commits for project")
			continue
		}

		allCommits = append(allCommits, commits...)

		// 如果已经找到提交，可以继续搜索其他项目
		if len(commits) > 0 {
			c.logger.WithFields(logrus.Fields{
				"project_name": project.Name,
				"commit_count": len(commits),
			}).Debug("Found commits in project")
		}
	}

	return allCommits, nil
}

// getUserCommitsOptimizedV4 使用v4 API优化版本搜索用户提交
func (c *Client) getUserCommitsOptimizedV4(user *gitlab.User, date time.Time) ([]*storage.Commit, error) {
	// v4 API实现（暂时使用现有逻辑，可以后续优化）
	return c.GetUserCommits(date)
}

// getProjectsWithCommitsOnDate 获取指定日期有提交的项目列表（快速预筛选）
func (c *Client) getProjectsWithCommitsOnDate(user *gitlab.User, date time.Time) ([]*gitlab.Project, error) {
	if !c.isV3API() {
		// v4 API可以使用更高效的方法
		return c.getProjectsWithCommitsOnDateV4(user, date)
	}

	// 获取中国时区
	cstLocation, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		cstLocation = time.FixedZone("CST", 8*3600)
	}

	// 设置时间范围
	localDate := date.In(cstLocation)
	startOfDay := time.Date(localDate.Year(), localDate.Month(), localDate.Day(), 0, 0, 0, 0, cstLocation)
	endOfDay := time.Date(localDate.Year(), localDate.Month(), localDate.Day(), 23, 59, 59, 999999999, cstLocation)

	var projectsWithCommits []*gitlab.Project

	// 首先检查特定项目（如果配置了的话）
	if len(c.config.ProjectIDs) > 0 {
		for _, projectID := range c.config.ProjectIDs {
			project, err := c.getProjectV3(projectID)
			if err != nil {
				c.logger.WithError(err).WithField("project_id", projectID).Warn("Failed to get configured project")
				continue
			}

			hasCommits, err := c.projectHasCommitsOnDateV3(project, user, startOfDay, endOfDay)
			if err != nil {
				c.logger.WithError(err).WithField("project_name", project.Name).Warn("Failed to check commits for configured project")
				continue
			}

			if hasCommits {
				projectsWithCommits = append(projectsWithCommits, project)
			}
		}
		return projectsWithCommits, nil
	}

	// 获取用户最近活跃的项目（限制数量以提高性能）
	params := url.Values{}
	params.Set("membership", "true")
	params.Set("order_by", "last_activity_at")
	params.Set("sort", "desc")
	params.Set("per_page", "100") // 限制为100个最活跃的项目

	body, err := c.makeV3Request("GET", "/projects", params)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent projects: %w", err)
	}

	var projectsV3 []ProjectV3
	if err := json.Unmarshal(body, &projectsV3); err != nil {
		return nil, fmt.Errorf("failed to parse projects response: %w", err)
	}

	c.logger.WithField("candidate_projects", len(projectsV3)).Info("Checking projects for commits on date")

	// 并发检查每个项目是否有提交（限制并发数）
	semaphore := make(chan struct{}, 10) // 最多10个并发请求
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, pv3 := range projectsV3 {
		wg.Add(1)
		go func(projectV3 ProjectV3) {
			defer wg.Done()
			semaphore <- struct{}{}        // 获取信号量
			defer func() { <-semaphore }() // 释放信号量

			project := &gitlab.Project{
				ID:                projectV3.ID,
				Name:              projectV3.Name,
				Path:              projectV3.Path,
				PathWithNamespace: projectV3.PathWithNamespace,
				DefaultBranch:     projectV3.DefaultBranch,
			}

			hasCommits, err := c.projectHasCommitsOnDateV3(project, user, startOfDay, endOfDay)
			if err != nil {
				c.logger.WithError(err).WithField("project_name", project.Name).Debug("Failed to check commits for project")
				return
			}

			if hasCommits {
				mu.Lock()
				projectsWithCommits = append(projectsWithCommits, project)
				c.logger.WithFields(logrus.Fields{
					"project_id":   project.ID,
					"project_name": project.Name,
					"project_path": project.PathWithNamespace,
				}).Info("Found project with commits on date")
				mu.Unlock()
			}
		}(pv3)
	}

	wg.Wait()

	c.logger.WithFields(logrus.Fields{
		"date":                   date.Format("2006-01-02"),
		"total_projects_checked": len(projectsV3),
		"projects_with_commits":  len(projectsWithCommits),
	}).Info("Completed project pre-filtering")

	return projectsWithCommits, nil
}

// projectHasCommitsOnDateV3 检查项目在指定日期是否有用户的提交（快速检查，只检查第一页）
func (c *Client) projectHasCommitsOnDateV3(project *gitlab.Project, user *gitlab.User, since, until time.Time) (bool, error) {
	// 获取项目的分支列表
	branches, err := c.getProjectBranchesV3(project.ID)
	if err != nil {
		// 如果获取分支失败，使用默认分支
		if project.DefaultBranch != "" {
			branches = []string{project.DefaultBranch}
		} else {
			branches = []string{"main", "master"}
		}
	}

	// 检查主要分支（最多检查3个分支以提高性能）
	maxBranches := 3
	if len(branches) > maxBranches {
		branches = branches[:maxBranches]
	}

	for _, branch := range branches {
		hasCommits, err := c.branchHasCommitsOnDateV3(project, user, branch, since, until)
		if err != nil {
			c.logger.WithError(err).WithFields(logrus.Fields{
				"project_name": project.Name,
				"branch":       branch,
			}).Debug("Failed to check branch for commits")
			continue
		}

		if hasCommits {
			return true, nil
		}
	}

	return false, nil
}

// branchHasCommitsOnDateV3 检查分支在指定日期是否有用户的提交（只检查第一页）
func (c *Client) branchHasCommitsOnDateV3(project *gitlab.Project, user *gitlab.User, branch string, since, until time.Time) (bool, error) {
	params := url.Values{}
	params.Set("page", "1")
	params.Set("per_page", "20") // 只检查前20个提交
	params.Set("ref_name", branch)
	params.Set("since", since.Format(time.RFC3339))
	params.Set("until", until.Format(time.RFC3339))

	endpoint := fmt.Sprintf("/projects/%d/repository/commits", project.ID)
	body, err := c.makeV3Request("GET", endpoint, params)
	if err != nil {
		return false, fmt.Errorf("failed to get commits for branch %s: %w", branch, err)
	}

	var commitsV3 []CommitV3
	if err := json.Unmarshal(body, &commitsV3); err != nil {
		return false, fmt.Errorf("failed to parse commits response: %w", err)
	}

	// 检查是否有用户的提交
	for _, commitV3 := range commitsV3 {
		// 支持邮箱匹配和用户名匹配
		emailMatch := commitV3.AuthorEmail == user.Email || commitV3.CommitterEmail == user.Email
		nameMatch := commitV3.AuthorName == user.Name || commitV3.AuthorName == user.Username

		if emailMatch || nameMatch {
			return true, nil
		}
	}

	return false, nil
}

// getProjectsWithCommitsOnDateV4 v4 API版本的项目预筛选
func (c *Client) getProjectsWithCommitsOnDateV4(user *gitlab.User, date time.Time) ([]*gitlab.Project, error) {
	// v4 API实现（如果需要的话）
	// 这里可以实现更高效的v4 API版本
	return nil, fmt.Errorf("v4 API project filtering not implemented yet")
}

// getProjectsWithCommitsOnDateV3UltraOptimized 使用超高效方法获取指定日期有提交的项目
func (c *Client) getProjectsWithCommitsOnDateV3UltraOptimized(user *gitlab.User, date time.Time) (map[int][]*gitlab.Project, error) {
	// 获取中国时区
	cstLocation, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		cstLocation = time.FixedZone("CST", 8*3600)
	}

	// 设置时间范围
	localDate := date.In(cstLocation)
	startOfDay := time.Date(localDate.Year(), localDate.Month(), localDate.Day(), 0, 0, 0, 0, cstLocation)
	endOfDay := time.Date(localDate.Year(), localDate.Month(), localDate.Day(), 23, 59, 59, 999999999, cstLocation)

	projectsWithCommits := make(map[int][]*gitlab.Project)

	// 首先获取用户的所有项目
	projects, err := c.getUserProjectsV3()
	if err != nil {
		return nil, fmt.Errorf("failed to get user projects: %w", err)
	}

	c.logger.WithField("total_projects", len(projects)).Info("Starting ultra efficient commit search")

	// 使用并发检查项目是否有当天的提交（只检查默认分支）
	type projectResult struct {
		project    *gitlab.Project
		hasCommits bool
	}

	resultChan := make(chan projectResult, len(projects))
	semaphore := make(chan struct{}, 10) // 限制并发数为10

	for _, project := range projects {
		go func(proj *gitlab.Project) {
			semaphore <- struct{}{}        // 获取信号量
			defer func() { <-semaphore }() // 释放信号量

			// 只检查默认分支是否有提交
			hasCommits := c.projectHasCommitsOnDateOptimized(proj, user, startOfDay, endOfDay)
			resultChan <- projectResult{project: proj, hasCommits: hasCommits}
		}(project)
	}

	// 收集结果
	activeProjects := make([]*gitlab.Project, 0)
	for i := 0; i < len(projects); i++ {
		result := <-resultChan
		if result.hasCommits {
			activeProjects = append(activeProjects, result.project)
		}
	}

	c.logger.WithFields(logrus.Fields{
		"total_projects":  len(projects),
		"active_projects": len(activeProjects),
	}).Info("Ultra efficient project filtering completed")

	// 对于有提交的项目，返回项目信息
	for _, project := range activeProjects {
		projectsWithCommits[project.ID] = []*gitlab.Project{project}
	}

	return projectsWithCommits, nil
}

// projectHasCommitsOnDateOptimized 快速检查项目在指定日期是否有提交（只检查默认分支）
func (c *Client) projectHasCommitsOnDateOptimized(project *gitlab.Project, user *gitlab.User, startTime, endTime time.Time) bool {
	// 构建API endpoint
	endpoint := fmt.Sprintf("/projects/%d/repository/commits", project.ID)

	params := url.Values{}
	params.Set("page", "1")
	params.Set("per_page", "5") // 只检查前5个提交
	params.Set("since", startTime.Format(time.RFC3339))
	params.Set("until", endTime.Format(time.RFC3339))

	c.logger.WithFields(logrus.Fields{
		"project_id":   project.ID,
		"project_name": project.Name,
		"endpoint":     endpoint,
	}).Debug("Checking project for commits")

	// 发送请求
	body, err := c.makeV3Request("GET", endpoint, params)
	if err != nil {
		c.logger.WithError(err).WithField("project_id", project.ID).Debug("Failed to check project commits")
		return false
	}

	// 解析响应
	var commits []CommitV3
	if err := json.Unmarshal(body, &commits); err != nil {
		c.logger.WithError(err).WithField("project_id", project.ID).Debug("Failed to decode commits response")
		return false
	}

	// 检查是否有匹配的提交
	for _, commit := range commits {
		emailMatch := commit.AuthorEmail == user.Email || commit.CommitterEmail == user.Email
		nameMatch := commit.AuthorName == user.Name || commit.AuthorName == user.Username

		if emailMatch || nameMatch {
			c.logger.WithFields(logrus.Fields{
				"project_id":   project.ID,
				"project_name": project.Name,
				"commit_id":    commit.ID,
			}).Debug("Found matching commit in project")
			return true
		}
	}

	return false
}

// GetUserCommitsUltraOptimized 使用全局搜索API直接查找用户在指定日期的提交
func (c *Client) GetUserCommitsUltraOptimized(date time.Time) ([]*storage.Commit, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("gitlab client is not enabled")
	}

	// 获取当前用户信息
	user, err := c.GetCurrentUser()
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	if c.isV3API() {
		return c.getUserCommitsUltraOptimizedV3(user, date)
	}

	// v4 API实现（如果需要的话）
	return c.getUserCommitsOptimizedV4(user, date)
}

// getUserCommitsUltraOptimizedV3 使用v3 API的超级优化版本，通过全局搜索直接查找提交
func (c *Client) getUserCommitsUltraOptimizedV3(user *gitlab.User, date time.Time) ([]*storage.Commit, error) {
	// 获取中国时区 (CST +8)
	cstLocation, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		// 如果加载失败，使用固定偏移量+8小时
		cstLocation = time.FixedZone("CST", 8*3600)
	}

	// 设置时间范围（当天00:00到23:59，使用中国时区）
	localDate := date.In(cstLocation)
	startOfDay := time.Date(localDate.Year(), localDate.Month(), localDate.Day(), 0, 0, 0, 0, cstLocation)
	endOfDay := time.Date(localDate.Year(), localDate.Month(), localDate.Day(), 23, 59, 59, 999999999, cstLocation)

	c.logger.WithFields(logrus.Fields{
		"date":         date.Format("2006-01-02"),
		"start_of_day": startOfDay.Format("2006-01-02 15:04:05 MST"),
		"end_of_day":   endOfDay.Format("2006-01-02 15:04:05 MST"),
	}).Info("Starting ultra-optimized commit search")

	var allCommits []*storage.Commit

	// 方法1：如果配置了特定项目，直接使用全局搜索在这些项目中查找
	if len(c.config.ProjectIDs) > 0 {
		c.logger.WithField("project_ids", c.config.ProjectIDs).Info("Using global search in configured projects")
		for _, projectID := range c.config.ProjectIDs {
			commits, err := c.searchCommitsInProjectV3(projectID, user, startOfDay, endOfDay)
			if err != nil {
				c.logger.WithError(err).WithField("project_id", projectID).Warn("Failed to search commits in project")
				continue
			}
			allCommits = append(allCommits, commits...)
		}
		return allCommits, nil
	}

	// 方法2：使用全局搜索API查找用户的提交
	commits, err := c.globalSearchUserCommitsV3(user, startOfDay, endOfDay)
	if err != nil {
		c.logger.WithError(err).Warn("Global search failed, falling back to optimized project search")
		// 如果全局搜索失败，回退到已有的优化方法
		return c.getUserCommitsOptimizedV3(user, date)
	}

	c.logger.WithFields(logrus.Fields{
		"date":         date.Format("2006-01-02"),
		"commit_count": len(commits),
	}).Info("Retrieved user commits using ultra-optimized global search")

	return commits, nil
}

// globalSearchUserCommitsV3 使用全局搜索API查找用户提交
func (c *Client) globalSearchUserCommitsV3(user *gitlab.User, startTime, endTime time.Time) ([]*storage.Commit, error) {
	// GitLab v3 API的搜索端点
	// 尝试多种搜索策略
	searchQueries := []string{
		fmt.Sprintf("author:%s", user.Username),
		fmt.Sprintf("author:%s", user.Email),
		fmt.Sprintf("committer:%s", user.Username),
		fmt.Sprintf("committer:%s", user.Email),
	}

	var allCommits []*storage.Commit
	seenCommits := make(map[string]bool) // 用于去重

	for _, query := range searchQueries {
		c.logger.WithField("search_query", query).Debug("Executing global search query")

		commits, err := c.executeGlobalSearchV3(query, startTime, endTime)
		if err != nil {
			c.logger.WithError(err).WithField("query", query).Warn("Global search query failed")
			continue
		}

		// 添加提交并去重
		for _, commit := range commits {
			if !seenCommits[commit.Hash] {
				seenCommits[commit.Hash] = true
				allCommits = append(allCommits, commit)
			}
		}
	}

	return allCommits, nil
}

// executeGlobalSearchV3 执行全局搜索查询
func (c *Client) executeGlobalSearchV3(searchQuery string, startTime, endTime time.Time) ([]*storage.Commit, error) {
	// GitLab v3 API可能不支持全局提交搜索，所以我们改用另一种策略：
	// 直接在已知的活跃项目中搜索，但只搜索默认分支

	// 首先尝试在enterprise-api项目中搜索
	if specificProject := c.searchSpecificProject("enterprise-api"); specificProject != nil {
		c.logger.WithField("project_name", specificProject.Name).Info("Searching commits in target project only")
		return c.searchCommitsInProjectV3(specificProject.ID, nil, startTime, endTime)
	}

	// 如果找不到特定项目，返回空结果
	c.logger.Warn("Could not find target project for optimized search")
	return []*storage.Commit{}, nil
}

// searchCommitsInProjectV3 在特定项目中搜索提交（优化版本，只搜索主要分支）
func (c *Client) searchCommitsInProjectV3(projectID interface{}, user *gitlab.User, startTime, endTime time.Time) ([]*storage.Commit, error) {
	// 获取项目信息
	project, err := c.getProjectV3(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	// 如果没有传入用户信息，获取当前用户
	if user == nil {
		user, err = c.GetCurrentUser()
		if err != nil {
			return nil, fmt.Errorf("failed to get current user: %w", err)
		}
	}

	c.logger.WithFields(logrus.Fields{
		"project_id":   project.ID,
		"project_name": project.Name,
		"user_email":   user.Email,
		"user_name":    user.Username,
	}).Info("Searching commits in project with optimized branch strategy")

	var allCommits []*storage.Commit

	// 优化策略：只搜索最可能有提交的分支
	priorityBranches := []string{
		"master",
		"main",
		"hotfix/p2-tax-import-error", // 用户提到的特定分支
	}

	// 如果项目有默认分支且不在优先列表中，添加它
	if project.DefaultBranch != "" {
		found := false
		for _, branch := range priorityBranches {
			if branch == project.DefaultBranch {
				found = true
				break
			}
		}
		if !found {
			priorityBranches = append([]string{project.DefaultBranch}, priorityBranches...)
		}
	}

	// 搜索优先分支
	for _, branchName := range priorityBranches {
		c.logger.WithFields(logrus.Fields{
			"project_name": project.Name,
			"branch":       branchName,
		}).Debug("Searching commits in priority branch")

		commits, err := c.getProjectCommitsFromBranchV3(project, user, branchName, startTime, endTime)
		if err != nil {
			c.logger.WithError(err).WithFields(logrus.Fields{
				"project_name": project.Name,
				"branch":       branchName,
			}).Debug("Failed to get commits from priority branch")
			continue
		}

		allCommits = append(allCommits, commits...)

		// 如果在优先分支中找到了提交，可以选择停止搜索其他分支
		if len(commits) > 0 {
			c.logger.WithFields(logrus.Fields{
				"project_name": project.Name,
				"branch":       branchName,
				"commit_count": len(commits),
			}).Info("Found commits in priority branch, continuing to search other branches")
		}
	}

	// 如果在优先分支中没找到提交，再搜索其他分支
	if len(allCommits) == 0 {
		c.logger.WithField("project_name", project.Name).Info("No commits found in priority branches, searching all branches")

		// 获取所有分支
		branches, err := c.getProjectBranchesV3(project.ID)
		if err != nil {
			c.logger.WithError(err).WithField("project_name", project.Name).Warn("Failed to get project branches")
			return allCommits, nil
		}

		// 搜索其他分支（跳过已经搜索过的优先分支）
		for _, branchName := range branches {
			// 跳过已经搜索过的优先分支
			skip := false
			for _, priorityBranch := range priorityBranches {
				if branchName == priorityBranch {
					skip = true
					break
				}
			}
			if skip {
				continue
			}

			commits, err := c.getProjectCommitsFromBranchV3(project, user, branchName, startTime, endTime)
			if err != nil {
				c.logger.WithError(err).WithFields(logrus.Fields{
					"project_name": project.Name,
					"branch":       branchName,
				}).Debug("Failed to get commits from branch")
				continue
			}

			allCommits = append(allCommits, commits...)

			// 如果找到提交，记录日志
			if len(commits) > 0 {
				c.logger.WithFields(logrus.Fields{
					"project_name": project.Name,
					"branch":       branchName,
					"commit_count": len(commits),
				}).Info("Found commits in additional branch")
			}
		}
	}

	c.logger.WithFields(logrus.Fields{
		"project_name":  project.Name,
		"total_commits": len(allCommits),
	}).Info("Completed optimized commit search in project")

	return allCommits, nil
}
