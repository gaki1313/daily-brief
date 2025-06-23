# 配置管理最佳实践

基于我们项目的配置管理系统，这里总结了Go项目中配置管理的最佳实践。

## 🎯 核心原则

### 1. **分层配置** - 不同环境使用不同配置
- `config.yaml` - 基础默认配置
- `config.dev.yaml` - 开发环境配置  
- `config.prod.yaml` - 生产环境配置
- `config.local.yaml` - 个人本地配置（不提交到Git）

### 2. **敏感信息分离** - 密码和密钥通过环境变量
- ❌ 不要：在配置文件中写明文密码
- ✅ 推荐：使用环境变量或密钥管理服务

### 3. **配置验证** - 启动时验证配置有效性
- 检查必需参数
- 验证数据格式
- 提供友好错误信息

## 📁 目录结构

```
configs/
├── config.yaml          # 默认配置（提交到Git）
├── config.dev.yaml      # 开发环境（提交到Git）
├── config.prod.yaml     # 生产环境模板（提交到Git，但不含敏感信息）
├── config.local.yaml    # 个人配置（不提交到Git）
└── templates/           # 配置模板
    ├── config.prod.template.yaml
    └── config.k8s.template.yaml
```

## 🔐 敏感信息处理

### ❌ 错误做法
```yaml
# 不要在配置文件中写明文密码
database:
  dsn: "user:password123@tcp(localhost:3306)/db"
feishu:
  app_secret: "real_secret_key_here"
```

### ✅ 正确做法
```yaml  
# 配置文件中使用占位符或留空
database:
  dsn: "${DB_USER}:${DB_PASSWORD}@tcp(${DB_HOST}:${DB_PORT})/${DB_NAME}"
feishu:
  app_secret: ""  # 通过环境变量设置
```

```bash
# 通过环境变量设置敏感信息
export DB_PASSWORD="real_password"
export FEISHU_APP_SECRET="real_secret"
```

## 🌍 不同环境的配置策略

### 开发环境
```yaml
# config.dev.yaml
app:
  mode: "debug"
  port: 3000
database:
  type: "sqlite"
  path: "./data/dev.db"
logging:
  level: "debug"
  format: "text"
feishu:
  enabled: false  # 开发环境不启用
```

### 生产环境
```yaml
# config.prod.yaml
app:
  mode: "release"
  port: 8080
database:
  type: "mysql"
  dsn: ""  # 通过环境变量设置
logging:
  level: "info"
  format: "json"
  file: "/var/log/daily-brief/app.log"
feishu:
  enabled: true
  app_secret: ""  # 通过环境变量设置
```

## 🔄 配置加载流程

```go
func Load() (*Config, error) {
    v := viper.New()
    
    // 1. 设置默认值
    setDefaults(v)
    
    // 2. 加载配置文件
    loadConfigFile(v)
    
    // 3. 读取环境变量（覆盖配置文件）
    setupEnvironmentVariables(v)
    
    // 4. 解析到结构体
    var config Config
    v.Unmarshal(&config)
    
    // 5. 验证配置
    return config.Validate()
}
```

## 🛠️ 实际应用示例

### 本地开发
```bash
# 创建个人配置文件
cat > configs/config.local.yaml <<EOF
app:
  port: 3001  # 避免端口冲突
database:
  path: "./data/$(whoami).db"  # 个人数据库
logging:
  level: "debug"
EOF

# 启动开发服务器
make dev
```

### Docker部署
```dockerfile
# Dockerfile
ENV GIN_MODE=release
ENV DAILY_BRIEF_APP_PORT=8080
ENV DAILY_BRIEF_DATABASE_TYPE=sqlite
ENV DAILY_BRIEF_DATABASE_PATH=/data/daily-brief.db

# 敏感信息通过运行时环境变量设置
# docker run -e FEISHU_APP_SECRET=xxx daily-brief
```

### Kubernetes部署
```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: daily-brief
        env:
        # 普通配置
        - name: DAILY_BRIEF_APP_PORT
          value: "8080"
        # 敏感信息从Secret读取
        - name: FEISHU_APP_SECRET
          valueFrom:
            secretKeyRef:
              name: daily-brief-secrets
              key: feishu-app-secret
        - name: DAILY_BRIEF_DATABASE_DSN
          valueFrom:
            secretKeyRef:
              name: daily-brief-secrets  
              key: database-dsn
```

## 🔍 配置调试技巧

### 1. 配置验证
```bash
# 验证配置文件语法
go run cmd/main.go -validate-config

# 查看最终配置（隐藏敏感信息）
go run cmd/main.go -dump-config
```

### 2. 环境变量检查
```bash
# 查看所有相关环境变量
env | grep DAILY_BRIEF

# 测试配置加载
DAILY_BRIEF_APP_PORT=9999 go run cmd/main.go -dry-run
```

### 3. 配置文件优先级测试
```bash
# 创建测试配置
echo "app: {port: 7777}" > config.test.yaml

# 指定使用测试配置
CONFIG_FILE=config.test.yaml go run cmd/main.go
```

## ⚡ 性能和安全建议

### 性能优化
1. **配置缓存** - 避免重复解析
2. **懒加载** - 按需加载配置模块
3. **配置热重载** - 支持运行时更新非敏感配置

### 安全建议
1. **权限控制** - 配置文件适当的文件权限
2. **密钥轮换** - 定期更换API密钥
3. **审计日志** - 记录配置变更历史
4. **环境隔离** - 不同环境使用不同的密钥

## 📚 相关工具推荐

- **Viper** - Go配置管理库（本项目使用）
- **Consul** - 分布式配置中心
- **HashiCorp Vault** - 密钥管理服务
- **Kubernetes Secrets** - K8s原生密钥管理
- **AWS Parameter Store** - AWS配置和密钥存储
- **Azure Key Vault** - Azure密钥管理服务

通过遵循这些最佳实践，你可以构建一个安全、灵活、易维护的配置管理系统。 