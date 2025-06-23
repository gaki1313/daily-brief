# 快速入门指南

## 🎯 项目概述

日报生成器是一个基于Go语言开发的智能日报生成系统，可以自动收集Git提交记录和用户工作要点，生成专业的日报并通过飞书推送。

## 📋 前置要求

### 必需软件
- **Go 1.21+**: 编程语言运行环境
- **Git**: 版本控制系统

### 可选软件
- **Make**: 构建工具（用于运行Makefile命令）
- **golangci-lint**: 代码检查工具
- **Docker**: 容器化部署（可选）

## 🚀 安装Go语言

### macOS
```bash
# 使用Homebrew安装（推荐）
brew install go

# 或下载安装包
# 访问 https://golang.org/dl/ 下载最新版本
```

### Linux (Ubuntu/Debian)
```bash
# 使用包管理器安装
sudo apt update
sudo apt install golang-go

# 或下载最新版本
wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
```

### Windows
1. 访问 https://golang.org/dl/
2. 下载Windows安装包
3. 运行安装程序
4. 重启命令提示符/PowerShell

### 验证安装
```bash
go version
# 应该输出类似：go version go1.21.5 darwin/amd64
```

## 📁 项目结构

```
daily-brief/
├── cmd/                    # 命令行入口
│   └── main.go            # 主程序入口
├── internal/              # 内部包（不对外暴露）
│   ├── config/           # 配置管理
│   ├── collector/        # 数据收集器
│   ├── generator/        # 日报生成器
│   ├── integration/      # 第三方集成
│   ├── storage/          # 数据存储
│   └── server/           # HTTP服务器
├── pkg/                   # 公共包（可对外暴露）
│   ├── logger/           # 日志工具
│   ├── utils/            # 通用工具
│   └── errors/           # 错误处理
├── configs/              # 配置文件
│   ├── config.yaml       # 主配置文件
│   └── templates/        # 日报模板
├── web/                  # Web界面
│   ├── static/           # 静态资源
│   └── templates/        # HTML模板
├── scripts/              # 构建和部署脚本
├── test/                 # 测试文件
├── data/                 # 数据文件（运行时创建）
├── logs/                 # 日志文件（运行时创建）
├── bin/                  # 编译输出（运行时创建）
├── go.mod               # Go模块定义
├── go.sum               # 依赖版本锁定
├── Makefile             # 构建脚本
├── README.md            # 项目文档
├── QUICKSTART.md        # 快速入门指南
└── .gitignore           # Git忽略文件
```

## 🛠️ 初始化项目

### 方法1：自动初始化（推荐）
```bash
# 运行初始化脚本
./scripts/init.sh
```

### 方法2：手动初始化
```bash
# 1. 创建必要目录
mkdir -p bin data logs web/static web/templates configs/templates test docs

# 2. 初始化Go模块
go mod tidy

# 3. 下载依赖
go mod download

# 4. 验证依赖
go mod verify
```

## 🏃‍♂️ 运行项目

### 开发模式
```bash
# 使用Makefile（推荐）
make dev

# 或直接运行Go命令
go run cmd/main.go
```

### 生产模式
```bash
# 构建二进制文件
make build

# 运行编译后的程序
./bin/daily-brief
```

## 🔧 配置应用

### 基础配置
编辑 `configs/config.yaml` 文件：

```yaml
# 应用基础配置
app:
  name: "Daily Brief Generator"
  version: "1.0.0"
  port: 8080
  mode: "debug"  # 开发时使用debug，生产时使用release

# 数据库配置
database:
  type: "sqlite"
  path: "./data/daily-brief.db"

# 飞书配置（可选）
feishu:
  enabled: false  # 暂时禁用，稍后配置
  app_id: ""
  app_secret: ""
  webhook_url: ""
```

### 环境变量配置（可选）
```bash
# 设置环境变量
export DAILY_BRIEF_APP_PORT=8080
export DAILY_BRIEF_APP_MODE=debug
export LOG_LEVEL=debug
```

## 🌐 访问应用

启动应用后，可以通过以下地址访问：

- **Web界面**: http://localhost:8080
- **仪表板**: http://localhost:8080/dashboard
- **健康检查**: http://localhost:8080/health
- **API文档**: http://localhost:8080/api/v1/

### API端点
- `GET /health` - 健康检查
- `GET /api/v1/repositories` - 获取仓库列表
- `POST /api/v1/repositories` - 创建新仓库
- `GET /api/v1/points` - 获取工作要点
- `POST /api/v1/points` - 创建工作要点

## 📊 测试API

### 使用curl测试
```bash
# 健康检查
curl http://localhost:8080/health

# 获取仓库列表
curl http://localhost:8080/api/v1/repositories

# 创建新仓库
curl -X POST http://localhost:8080/api/v1/repositories \
  -H "Content-Type: application/json" \
  -d '{"name":"test-repo","path":"/path/to/repo","branch":"main"}'

# 创建工作要点
curl -X POST http://localhost:8080/api/v1/points \
  -H "Content-Type: application/json" \
  -d '{"title":"完成项目初始化","category":"completed","priority":2}'
```

### 使用浏览器测试
1. 访问 http://localhost:8080
2. 点击导航链接查看不同页面
3. 访问 http://localhost:8080/api/v1/repositories 查看API响应

## 🛠️ 开发工具

### Make命令
```bash
make help        # 显示所有可用命令
make dev         # 启动开发服务器
make build       # 构建应用程序
make test        # 运行测试
make fmt         # 格式化代码
make lint        # 代码检查
make clean       # 清理构建文件
```

### 推荐的开发工具
- **VS Code**: 推荐安装Go扩展
- **GoLand**: JetBrains的Go IDE
- **Vim/Neovim**: 配置vim-go插件

## 🐛 常见问题

### Q: 运行时提示"command not found: go"
**A:** Go未安装或未添加到PATH。请按照上面的安装指南安装Go。

### Q: 端口8080已被占用
**A:** 修改配置文件中的端口号，或使用环境变量：
```bash
export DAILY_BRIEF_APP_PORT=8081
```

### Q: 数据库文件权限错误
**A:** 确保data目录有写入权限：
```bash
chmod 755 data/
```

### Q: 依赖下载失败
**A:** 检查网络连接，或配置Go代理：
```bash
go env -w GOPROXY=https://goproxy.cn,direct
```

## 📚 下一步

1. **配置Git仓库**: 添加要监控的Git仓库
2. **创建工作要点**: 添加日常工作内容
3. **配置飞书集成**: 设置飞书机器人推送
4. **生成第一份日报**: 测试日报生成功能
5. **自定义模板**: 根据需要调整日报模板

## 📖 更多文档

- [详细配置指南](docs/configuration.md)
- [API文档](docs/api.md)
- [部署指南](docs/deployment.md)
- [开发指南](docs/development.md)

## 💡 获取帮助

如果遇到问题，请：
1. 查看日志文件：`tail -f logs/app.log`
2. 检查配置文件语法
3. 运行 `make help` 查看可用命令
4. 访问健康检查端点确认服务状态 