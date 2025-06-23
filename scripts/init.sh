#!/bin/bash

# Daily Brief Generator 项目初始化脚本
# 自动检查环境依赖并创建必要的目录结构

set -e

echo "🚀 Initializing Daily Brief Generator..."

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 检查Go是否安装
check_go() {
    if command -v go &> /dev/null; then
        GO_VERSION=$(go version | cut -d' ' -f3)
        echo -e "${GREEN}✓${NC} Go is installed: $GO_VERSION"
        
        # 检查Go版本是否符合要求
        REQUIRED_VERSION="go1.21"
        if [[ "$GO_VERSION" < "$REQUIRED_VERSION" ]]; then
            echo -e "${YELLOW}⚠${NC} Warning: Go version $GO_VERSION detected, recommended $REQUIRED_VERSION or higher"
        fi
    else
        echo -e "${RED}✗${NC} Go is not installed or not in PATH"
        echo -e "${YELLOW}Please install Go 1.21 or higher from: https://golang.org/dl/${NC}"
        echo -e "${YELLOW}Or install via Homebrew: brew install go${NC}"
        exit 1
    fi
}

# 创建目录结构
create_directories() {
    echo -e "${BLUE}📁 Creating directory structure...${NC}"
    
    directories=(
        "bin"
        "data"
        "logs" 
        "web/static"
        "web/templates"
        "configs/templates"
        "test"
        "docs"
        "scripts"
    )
    
    for dir in "${directories[@]}"; do
        if [[ ! -d "$dir" ]]; then
            mkdir -p "$dir"
            echo "   Created: $dir"
        else
            echo "   Exists: $dir"
        fi
    done
}

# 初始化Go模块（如果Go可用）
init_go_module() {
    if command -v go &> /dev/null; then
        echo -e "${BLUE}📦 Initializing Go module...${NC}"
        
        if [[ ! -f "go.mod" ]]; then
            go mod init daily-brief
            echo "   Created go.mod"
        else
            echo "   go.mod already exists"
        fi
        
        echo -e "${BLUE}📦 Downloading dependencies...${NC}"
        go mod tidy
        echo "   Dependencies downloaded"
        
        echo -e "${BLUE}🔍 Verifying dependencies...${NC}"
        go mod verify
        echo "   Dependencies verified"
    else
        echo -e "${YELLOW}⚠${NC} Skipping Go module initialization (Go not available)"
    fi
}

# 创建基础HTML模板文件
create_templates() {
    echo -e "${BLUE}🎨 Creating basic templates...${NC}"
    
    # 创建基础HTML模板
    if [[ ! -f "web/templates/index.html" ]]; then
        cat > web/templates/index.html << 'EOF'
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.title}}</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .header { text-align: center; margin-bottom: 30px; }
        .version { color: #666; font-size: 14px; }
        .nav { margin: 20px 0; }
        .nav a { margin-right: 20px; text-decoration: none; color: #007bff; }
        .nav a:hover { text-decoration: underline; }
    </style>
</head>
<body>
    <div class="header">
        <h1>{{.title}}</h1>
        <p class="version">Version: {{.version}}</p>
    </div>
    
    <div class="nav">
        <a href="/">首页</a>
        <a href="/dashboard">仪表板</a>
        <a href="/health">健康检查</a>
        <a href="/api/v1/repositories">API文档</a>
    </div>
    
    <div class="content">
        <h2>欢迎使用日报生成器</h2>
        <p>这是一个基于Go语言开发的智能日报生成器，能够自动收集Git提交记录和用户要点，生成专业的工作日报。</p>
        
        <h3>主要功能</h3>
        <ul>
            <li>自动收集Git提交记录</li>
            <li>管理工作要点和计划</li>
            <li>生成专业日报</li>
            <li>飞书机器人集成</li>
        </ul>
    </div>
</body>
</html>
EOF
        echo "   Created web/templates/index.html"
    fi
    
    # 创建仪表板模板
    if [[ ! -f "web/templates/dashboard.html" ]]; then
        cat > web/templates/dashboard.html << 'EOF'
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.title}}</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .header { text-align: center; margin-bottom: 30px; }
        .nav { margin: 20px 0; }
        .nav a { margin-right: 20px; text-decoration: none; color: #007bff; }
        .nav a:hover { text-decoration: underline; }
        .dashboard { display: grid; grid-template-columns: 1fr 1fr; gap: 20px; }
        .card { border: 1px solid #ddd; padding: 20px; border-radius: 8px; }
        .card h3 { margin-top: 0; }
    </style>
</head>
<body>
    <div class="header">
        <h1>Dashboard</h1>
    </div>
    
    <div class="nav">
        <a href="/">首页</a>
        <a href="/dashboard">仪表板</a>
        <a href="/health">健康检查</a>
        <a href="/api/v1/repositories">API文档</a>
    </div>
    
    <div class="dashboard">
        <div class="card">
            <h3>Git仓库</h3>
            <p>管理和扫描Git仓库</p>
            <button onclick="location.href='/api/v1/repositories'">查看仓库</button>
        </div>
        
        <div class="card">
            <h3>工作要点</h3>
            <p>管理日常工作要点</p>
            <button onclick="location.href='/api/v1/points'">查看要点</button>
        </div>
        
        <div class="card">
            <h3>日报管理</h3>
            <p>生成和管理日报</p>
            <button onclick="location.href='/api/v1/reports'">查看日报</button>
        </div>
        
        <div class="card">
            <h3>集成设置</h3>
            <p>配置第三方集成</p>
            <button onclick="location.href='/api/v1/integrations/status'">查看状态</button>
        </div>
    </div>
</body>
</html>
EOF
        echo "   Created web/templates/dashboard.html"
    fi
}

# 设置权限
set_permissions() {
    echo -e "${BLUE}🔐 Setting permissions...${NC}"
    
    # 设置脚本执行权限
    chmod +x scripts/*.sh 2>/dev/null || true
    
    # 设置数据目录权限
    chmod 755 data logs 2>/dev/null || true
}

# 主函数
main() {
    echo -e "${GREEN}Daily Brief Generator - Project Initialization${NC}"
    echo "=============================================="
    
    check_go
    create_directories
    init_go_module
    create_templates
    set_permissions
    
    echo ""
    echo -e "${GREEN}✅ Project initialization completed successfully!${NC}"
    echo ""
    echo -e "${BLUE}Next steps:${NC}"
    echo "1. Review and update configs/config.yaml"
    echo "2. Run 'make dev' to start development server"
    echo "3. Or run 'make build' to build the application"
    echo "4. Visit http://localhost:8080 to access the application"
    echo ""
    echo -e "${YELLOW}Note: If Go is not installed, please install it first:${NC}"
    echo "   macOS: brew install go"
    echo "   Linux: sudo apt-get install golang-go"
    echo "   Windows: Download from https://golang.org/dl/"
}

# 运行主函数
main "$@" 