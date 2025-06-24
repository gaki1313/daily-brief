#!/bin/bash

# GitLab 日报生成测试脚本
# 用于测试GitLab日报生成API的功能

set -e

# 配置变量
SERVER_URL="${1:-http://localhost:3000}"
DATE="${2:-$(date +%Y-%m-%d)}"
TEMPLATE="${3:-standard}"

echo "=========================================="
echo "GitLab 日报生成测试"
echo "=========================================="
echo "服务器地址: $SERVER_URL"
echo "日期: $DATE"
echo "模板: $TEMPLATE"
echo ""

# 检查服务器是否运行
echo "1. 检查服务器状态..."
if ! curl -s "$SERVER_URL/health" > /dev/null; then
    echo "❌ 服务器未启动或无法访问"
    echo "请确保服务器运行在 $SERVER_URL"
    exit 1
fi
echo "✅ 服务器运行正常"

# 检查GitLab集成状态
echo ""
echo "2. 检查GitLab集成状态..."
GITLAB_STATUS=$(curl -s "$SERVER_URL/api/v1/integrations/gitlab/status" | jq -r '.data.enabled // false')
if [ "$GITLAB_STATUS" != "true" ]; then
    echo "❌ GitLab集成未启用"
    echo "请检查config.local.yaml中的GitLab配置"
    exit 1
fi
echo "✅ GitLab集成已启用"

# 测试GitLab连接
echo ""
echo "3. 测试GitLab连接..."
GITLAB_TEST=$(curl -s -X POST "$SERVER_URL/api/v1/integrations/gitlab/test" | jq -r '.code')
if [ "$GITLAB_TEST" != "200" ]; then
    echo "⚠️  GitLab连接测试失败，但继续进行日报生成测试"
else
    echo "✅ GitLab连接测试成功"
fi

# 获取GitLab用户信息
echo ""
echo "4. 获取GitLab用户信息..."
USER_INFO=$(curl -s "$SERVER_URL/api/v1/integrations/gitlab/user")
USER_NAME=$(echo "$USER_INFO" | jq -r '.data.name // "Unknown"')
USER_EMAIL=$(echo "$USER_INFO" | jq -r '.data.email // "unknown@example.com"')
echo "GitLab用户: $USER_NAME ($USER_EMAIL)"

# 获取GitLab项目列表
echo ""
echo "5. 获取GitLab项目列表..."
PROJECTS=$(curl -s "$SERVER_URL/api/v1/integrations/gitlab/projects")
PROJECT_COUNT=$(echo "$PROJECTS" | jq -r '.data | length // 0')
echo "可访问项目数量: $PROJECT_COUNT"

# 获取指定日期的GitLab提交记录
echo ""
echo "6. 获取GitLab提交记录..."
COMMITS=$(curl -s "$SERVER_URL/api/v1/integrations/gitlab/commits/$DATE")
COMMIT_COUNT=$(echo "$COMMITS" | jq -r '.data | length // 0')
echo "当日提交数量: $COMMIT_COUNT"

# 预览GitLab日报
echo ""
echo "7. 预览GitLab日报..."
PREVIEW_RESPONSE=$(curl -s "$SERVER_URL/api/v1/reports/gitlab/preview?date=$DATE&template=$TEMPLATE")
PREVIEW_STATUS=$(echo "$PREVIEW_RESPONSE" | jq -r '.code')

if [ "$PREVIEW_STATUS" == "200" ]; then
    echo "✅ GitLab日报预览生成成功"
    PREVIEW_CONTENT=$(echo "$PREVIEW_RESPONSE" | jq -r '.data.content')
    echo ""
    echo "========== 日报预览 =========="
    echo "$PREVIEW_CONTENT"
    echo "=============================="
else
    echo "❌ GitLab日报预览生成失败"
    echo "错误信息: $(echo "$PREVIEW_RESPONSE" | jq -r '.message')"
fi

# 生成GitLab日报
echo ""
echo "8. 生成并保存GitLab日报..."
GENERATE_PAYLOAD=$(cat <<EOF
{
    "date": "$DATE",
    "template": "$TEMPLATE"
}
EOF
)

GENERATE_RESPONSE=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -d "$GENERATE_PAYLOAD" \
    "$SERVER_URL/api/v1/reports/gitlab/generate")

GENERATE_STATUS=$(echo "$GENERATE_RESPONSE" | jq -r '.code')

if [ "$GENERATE_STATUS" == "200" ]; then
    echo "✅ GitLab日报生成并保存成功"
    REPORT_ID=$(echo "$GENERATE_RESPONSE" | jq -r '.data.id')
    REPORT_TITLE=$(echo "$GENERATE_RESPONSE" | jq -r '.data.title')
    echo "日报ID: $REPORT_ID"
    echo "日报标题: $REPORT_TITLE"
else
    echo "❌ GitLab日报生成失败"
    echo "错误信息: $(echo "$GENERATE_RESPONSE" | jq -r '.message')"
fi

# 获取GitLab日报数据
echo ""
echo "9. 获取GitLab日报数据..."
DATA_RESPONSE=$(curl -s "$SERVER_URL/api/v1/reports/gitlab/data/$DATE")
DATA_STATUS=$(echo "$DATA_RESPONSE" | jq -r '.code')

if [ "$DATA_STATUS" == "200" ]; then
    echo "✅ GitLab日报数据获取成功"
    TOTAL_COMMITS=$(echo "$DATA_RESPONSE" | jq -r '.data.summary.total_commits')
    LINES_ADDED=$(echo "$DATA_RESPONSE" | jq -r '.data.summary.lines_added')
    LINES_DELETED=$(echo "$DATA_RESPONSE" | jq -r '.data.summary.lines_deleted')
    PRODUCTIVITY_SCORE=$(echo "$DATA_RESPONSE" | jq -r '.data.summary.productivity_score')
    
    echo "统计信息："
    echo "  - 总提交数: $TOTAL_COMMITS"
    echo "  - 新增代码行: $LINES_ADDED"
    echo "  - 删除代码行: $LINES_DELETED"
    echo "  - 生产力评分: $PRODUCTIVITY_SCORE"
else
    echo "❌ GitLab日报数据获取失败"
    echo "错误信息: $(echo "$DATA_RESPONSE" | jq -r '.message')"
fi

echo ""
echo "=========================================="
echo "GitLab 日报生成测试完成"
echo "=========================================="

# 显示使用建议
echo ""
echo "💡 使用建议:"
echo "1. 确保在config.local.yaml中正确配置了GitLab信息"
echo "2. 使用有效的GitLab Personal Access Token"
echo "3. 确保token有足够的权限访问项目和提交记录"
echo "4. 可以通过以下API使用GitLab日报功能:"
echo "   - 预览: GET $SERVER_URL/api/v1/reports/gitlab/preview?date=YYYY-MM-DD"
echo "   - 生成: POST $SERVER_URL/api/v1/reports/gitlab/generate"
echo "   - 数据: GET $SERVER_URL/api/v1/reports/gitlab/data/YYYY-MM-DD" 