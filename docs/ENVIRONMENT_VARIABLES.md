# 环境变量配置指南

项目支持通过环境变量覆盖配置文件中的设置，这是处理敏感信息（如数据库密码、API密钥）的最佳实践。

## 🔐 环境变量优先级

环境变量的优先级**高于**配置文件，加载顺序：
1. **环境变量** (最高优先级)
2. 配置文件 (`config.local.yaml` > `config.prod.yaml` > `config.yaml`)
3. 默认值 (最低优先级)

## 📝 环境变量命名规则

### 标准格式
```bash
DAILY_BRIEF_<SECTION>_<KEY>=value
```

### 示例映射
| 配置项 | 环境变量 |
|--------|----------|
| `app.port` | `DAILY_BRIEF_APP_PORT` |
| `database.dsn` | `DAILY_BRIEF_DATABASE_DSN` |
| `feishu.app_id` | `DAILY_BRIEF_FEISHU_APP_ID` |

### 简化别名
为了方便使用，某些常用配置支持简化的环境变量名：

| 简化环境变量 | 标准环境变量 | 配置项 |
|-------------|-------------|--------|
| `PORT` | `DAILY_BRIEF_APP_PORT` | `app.port` |
| `GIN_MODE` | `DAILY_BRIEF_APP_MODE` | `app.mode` |
| `LOG_LEVEL` | `DAILY_BRIEF_LOGGING_LEVEL` | `logging.level` |
| `FEISHU_APP_ID` | `DAILY_BRIEF_FEISHU_APP_ID` | `feishu.app_id` |

## 🗃️ 数据库配置

### SQLite (开发环境)
```bash
export DAILY_BRIEF_DATABASE_TYPE=sqlite
export DAILY_BRIEF_DATABASE_PATH=./data/daily-brief.db
```

### MySQL (生产环境)
```bash
export DAILY_BRIEF_DATABASE_TYPE=mysql
export DAILY_BRIEF_DATABASE_DSN="user:password@tcp(localhost:3306)/daily_brief?charset=utf8mb4&parseTime=True&loc=Local"

# 或者分别设置
export DB_HOST=localhost
export DB_PORT=3306
export DB_USER=daily_brief_user
export DB_PASSWORD=your_secret_password
export DB_NAME=daily_brief
```

### PostgreSQL (生产环境)
```bash
export DAILY_BRIEF_DATABASE_TYPE=postgres
export DAILY_BRIEF_DATABASE_DSN="host=localhost user=username password=password dbname=daily_brief port=5432 sslmode=disable TimeZone=Asia/Shanghai"
```

## 🤖 飞书配置

```bash
# 启用飞书集成
export DAILY_BRIEF_FEISHU_ENABLED=true

# 飞书应用凭证
export DAILY_BRIEF_FEISHU_APP_ID=cli_a1b2c3d4e5f6g7h8
export DAILY_BRIEF_FEISHU_APP_SECRET=your_feishu_app_secret

# 机器人Webhook地址
export DAILY_BRIEF_FEISHU_WEBHOOK_URL=https://open.feishu.cn/open-apis/bot/v2/hook/your_webhook_token

# 群聊ID（可选）
export DAILY_BRIEF_FEISHU_CHAT_ID=oc_your_chat_id

# 简化别名
export FEISHU_APP_ID=cli_a1b2c3d4e5f6g7h8
export FEISHU_APP_SECRET=your_feishu_app_secret
export FEISHU_WEBHOOK_URL=https://open.feishu.cn/open-apis/bot/v2/hook/your_webhook_token
```

## 📊 应用配置

```bash
# 服务端口
export PORT=8080
export DAILY_BRIEF_APP_PORT=8080

# 运行模式
export GIN_MODE=release
export DAILY_BRIEF_APP_MODE=release

# 应用信息
export DAILY_BRIEF_APP_NAME="Daily Brief Generator"
export DAILY_BRIEF_APP_VERSION="1.0.0"
```

## 📝 日志配置

```bash
# 日志级别
export LOG_LEVEL=info
export DAILY_BRIEF_LOGGING_LEVEL=info

# 日志文件
export LOG_FILE=/var/log/daily-brief/app.log
export DAILY_BRIEF_LOGGING_FILE=/var/log/daily-brief/app.log

# 日志格式
export DAILY_BRIEF_LOGGING_FORMAT=json

# 日志轮转配置
export DAILY_BRIEF_LOGGING_MAX_SIZE=100
export DAILY_BRIEF_LOGGING_MAX_AGE=30
export DAILY_BRIEF_LOGGING_MAX_BACKUPS=10
export DAILY_BRIEF_LOGGING_COMPRESS=true
```

## 🔧 其他配置

### Git扫描配置
```bash
export DAILY_BRIEF_GIT_SCAN_INTERVAL=30m
export DAILY_BRIEF_GIT_MAX_COMMITS_PER_DAY=100
export DAILY_BRIEF_GIT_DEFAULT_BRANCH=main
```

### 日报配置
```bash
export DAILY_BRIEF_REPORT_AUTO_SEND=true
export DAILY_BRIEF_REPORT_SEND_TIME=18:00
export DAILY_BRIEF_REPORT_TIMEZONE=Asia/Shanghai
export DAILY_BRIEF_REPORT_DEFAULT_TEMPLATE=professional
```

## 🐳 Docker环境变量

使用Docker运行时的环境变量示例：

```dockerfile
# Dockerfile
ENV DAILY_BRIEF_APP_PORT=8080
ENV DAILY_BRIEF_DATABASE_TYPE=mysql
ENV DAILY_BRIEF_LOGGING_LEVEL=info
ENV GIN_MODE=release
```

```yaml
# docker-compose.yml
version: '3.8'
services:
  daily-brief:
    image: daily-brief:latest
    environment:
      - PORT=8080
      - GIN_MODE=release
      - DAILY_BRIEF_DATABASE_TYPE=mysql
      - DAILY_BRIEF_DATABASE_DSN=user:password@tcp(mysql:3306)/daily_brief?charset=utf8mb4&parseTime=True&loc=Local
      - DAILY_BRIEF_FEISHU_ENABLED=true
      - FEISHU_APP_ID=cli_your_app_id
      - FEISHU_APP_SECRET=your_app_secret
      - FEISHU_WEBHOOK_URL=https://open.feishu.cn/open-apis/bot/v2/hook/your_token
    ports:
      - "8080:8080"
    depends_on:
      - mysql
  
  mysql:
    image: mysql:8.0
    environment:
      - MYSQL_ROOT_PASSWORD=root_password
      - MYSQL_DATABASE=daily_brief
      - MYSQL_USER=daily_brief_user
      - MYSQL_PASSWORD=user_password
```

## 🚀 部署最佳实践

### 1. 开发环境
创建 `.env` 文件（不要提交到Git）：
```bash
# .env
PORT=3000
GIN_MODE=debug
DAILY_BRIEF_DATABASE_TYPE=sqlite
DAILY_BRIEF_DATABASE_PATH=./data/dev.db
LOG_LEVEL=debug
```

### 2. 生产环境
使用系统环境变量或密钥管理服务：
```bash
# 系统环境变量
sudo tee /etc/environment <<EOF
DAILY_BRIEF_APP_PORT=8080
DAILY_BRIEF_DATABASE_DSN="user:$(cat /etc/db-password)@tcp(db-host:3306)/daily_brief"
FEISHU_APP_SECRET="$(cat /etc/feishu-secret)"
EOF
```

### 3. Kubernetes
使用Secret和ConfigMap：
```yaml
# secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: daily-brief-secrets
type: Opaque
stringData:
  DAILY_BRIEF_DATABASE_DSN: "user:password@tcp(mysql:3306)/daily_brief"
  FEISHU_APP_SECRET: "your_secret_here"

---
# configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: daily-brief-config
data:
  DAILY_BRIEF_APP_PORT: "8080"
  DAILY_BRIEF_LOGGING_LEVEL: "info"
  GIN_MODE: "release"
```

## 🔍 环境变量验证

启动应用时会显示使用的配置文件：
```bash
$ ./daily-brief
Using config file: /etc/daily-brief/config.prod.yaml
{"level":"info","msg":"Starting Daily Brief Generator...","time":"2025-06-23T18:00:00+08:00"}
```

可以通过健康检查接口验证配置：
```bash
curl http://localhost:8080/health
# 响应包含版本和配置信息
``` 