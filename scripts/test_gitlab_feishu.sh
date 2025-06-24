#!/bin/bash

# GitLab日报飞书推送测试脚本
# 测试将GitLab日报发送到飞书群聊的功能

set -e

# 配置变量
SERVER_URL="${1:-http://localhost:3000}"
DATE="${2:-$(date +%Y-%m-%d)}"
TEMPLATE="${3:-simple}"
MESSAGE_TYPE="${4:-text}"

echo "=========================================="
echo "GitLab日报飞书推送测试"
echo "=========================================="
echo "服务器地址: $SERVER_URL"
echo "日期: $DATE"
echo "模板: $TEMPLATE"
echo "消息类型: $MESSAGE_TYPE"
echo ""

# 检查服务器状态
echo "1. 检查服务器状态..."
if ! curl -s "$SERVER_URL/health" > /dev/null; then
    echo "❌ 服务器未启动或无法访问"
    echo "请确保服务器运行在 $SERVER_URL"
    exit 1
fi
echo "✅ 服务器运行正常"

# 检查飞书集成状态
echo ""
echo "2. 检查飞书集成状态..."
FEISHU_STATUS=$(curl -s "$SERVER_URL/api/v1/integrations/feishu/status")
FEISHU_ENABLED=$(echo "$FEISHU_STATUS" | grep -o '"enabled":true' || echo "false")
if [ "$FEISHU_ENABLED" == "false" ]; then
    echo "❌ 飞书集成未启用"
    echo "请检查config.local.yaml中的飞书配置"
    exit 1
fi
echo "✅ 飞书集成已启用"

# 检查GitLab集成状态
echo ""
echo "3. 检查GitLab集成状态..."
GITLAB_STATUS=$(curl -s "$SERVER_URL/api/v1/integrations/gitlab/status")
GITLAB_ENABLED=$(echo "$GITLAB_STATUS" | grep -o '"enabled":true' || echo "false")
if [ "$GITLAB_ENABLED" == "false" ]; then
    echo "⚠️  GitLab集成未启用，将使用本地Git数据"
else
    echo "✅ GitLab集成已启用"
fi

# 测试飞书连接
echo ""
echo "4. 测试飞书连接..."
FEISHU_TEST=$(curl -s -X POST "$SERVER_URL/api/v1/integrations/feishu/test")
FEISHU_TEST_CODE=$(echo "$FEISHU_TEST" | grep -o '"code":[0-9]*' | cut -d':' -f2)
if [ "$FEISHU_TEST_CODE" != "200" ]; then
    echo "⚠️  飞书连接测试失败，但继续进行推送测试"
    echo "错误信息: $(echo "$FEISHU_TEST" | grep -o '"message":"[^"]*"' | cut -d'"' -f4)"
else
    echo "✅ 飞书连接测试成功"
fi

# 预览要发送的日报内容
echo ""
echo "5. 预览日报内容..."
if [ "$GITLAB_ENABLED" != "false" ]; then
    PREVIEW_URL="$SERVER_URL/api/v1/reports/gitlab/preview?date=$DATE&template=$TEMPLATE"
    echo "使用GitLab专用日报..."
else
    PREVIEW_URL="$SERVER_URL/api/v1/reports/preview?date=$DATE&template=$TEMPLATE"
    echo "使用标准日报（包含本地Git数据）..."
fi

PREVIEW_RESPONSE=$(curl -s "$PREVIEW_URL")
PREVIEW_STATUS=$(echo "$PREVIEW_RESPONSE" | grep -o '"code":[0-9]*' | cut -d':' -f2)

if [ "$PREVIEW_STATUS" == "200" ]; then
    echo "✅ 日报内容预览成功"
    # 提取内容并显示前几行
    CONTENT=$(echo "$PREVIEW_RESPONSE" | sed 's/.*"content":"\([^"]*\)".*/\1/' | sed 's/\\n/\n/g')
    echo ""
    echo "========== 日报内容预览 =========="
    echo "$CONTENT" | head -10
    echo "..."
    echo "（内容已截断，完整内容将发送到飞书）"
    echo "=============================="
else
    echo "❌ 日报内容预览失败"
    echo "错误信息: $(echo "$PREVIEW_RESPONSE" | grep -o '"message":"[^"]*"' | cut -d'"' -f4)"
    exit 1
fi

# 发送日报到飞书
echo ""
echo "6. 发送日报到飞书..."
SEND_PAYLOAD=$(cat <<EOF
{
    "date": "$DATE",
    "template": "$TEMPLATE",
    "message_type": "$MESSAGE_TYPE"
}
EOF
)

SEND_RESPONSE=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -d "$SEND_PAYLOAD" \
    "$SERVER_URL/api/v1/integrations/feishu/send/report")

SEND_STATUS=$(echo "$SEND_RESPONSE" | grep -o '"code":[0-9]*' | cut -d':' -f2)

if [ "$SEND_STATUS" == "200" ]; then
    echo "✅ 日报已成功发送到飞书"
    MESSAGE_ID=$(echo "$SEND_RESPONSE" | grep -o '"message_id":"[^"]*"' | cut -d'"' -f4)
    SENT_AT=$(echo "$SEND_RESPONSE" | grep -o '"sent_at":"[^"]*"' | cut -d'"' -f4)
    DURATION=$(echo "$SEND_RESPONSE" | grep -o '"duration":[0-9]*' | cut -d':' -f2)
    
    echo "发送详情："
    echo "  - 消息ID: $MESSAGE_ID"
    echo "  - 发送时间: $SENT_AT"
    echo "  - 耗时: ${DURATION}ms"
else
    echo "❌ 发送到飞书失败"
    echo "错误信息: $(echo "$SEND_RESPONSE" | grep -o '"message":"[^"]*"' | cut -d'"' -f4)"
fi

# 检查发送历史
echo ""
echo "7. 检查发送历史..."
LOGS_RESPONSE=$(curl -s "$SERVER_URL/api/v1/integrations/logs?platform=feishu&limit=5")
LOGS_STATUS=$(echo "$LOGS_RESPONSE" | grep -o '"code":[0-9]*' | cut -d':' -f2)

if [ "$LOGS_STATUS" == "200" ]; then
    echo "✅ 发送历史获取成功"
    LOG_COUNT=$(echo "$LOGS_RESPONSE" | grep -o '"count":[0-9]*' | cut -d':' -f2)
    echo "最近发送记录数: $LOG_COUNT"
else
    echo "⚠️  发送历史获取失败"
fi

# 测试不同模板和消息类型
echo ""
echo "8. 测试不同模板发送..."

echo "发送Markdown格式的详细日报..."
MARKDOWN_PAYLOAD=$(cat <<EOF
{
    "date": "$DATE",
    "template": "markdown",
    "message_type": "text"
}
EOF
)

MARKDOWN_RESPONSE=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -d "$MARKDOWN_PAYLOAD" \
    "$SERVER_URL/api/v1/integrations/feishu/send/report")

MARKDOWN_STATUS=$(echo "$MARKDOWN_RESPONSE" | grep -o '"code":[0-9]*' | cut -d':' -f2)

if [ "$MARKDOWN_STATUS" == "200" ]; then
    echo "✅ Markdown格式日报发送成功"
else
    echo "⚠️  Markdown格式日报发送失败"
fi

echo ""
echo "=========================================="
echo "GitLab日报飞书推送测试完成"
echo "=========================================="

# 显示使用说明
echo ""
echo "💡 飞书推送功能说明:"
echo "1. 支持多种消息类型："
echo "   - text: 纯文本消息（推荐）"
echo "   - card: 卡片消息（更丰富的格式）"
echo ""
echo "2. 支持多种报告模板："
echo "   - simple: 简洁版本"
echo "   - standard: 标准版本"
echo "   - detailed: 详细版本"
echo "   - markdown: Markdown格式"
echo ""
echo "3. API使用示例："
echo "   # 发送今天的简洁日报"
echo "   curl -X POST -H \"Content-Type: application/json\" \\"
echo "     -d '{\"date\":\"$(date +%Y-%m-%d)\",\"template\":\"simple\"}' \\"
echo "     \"$SERVER_URL/api/v1/integrations/feishu/send/report\""
echo ""
echo "   # 发送GitLab专用日报"
echo "   curl -X POST -H \"Content-Type: application/json\" \\"
echo "     -d '{\"date\":\"$(date +%Y-%m-%d)\",\"template\":\"standard\"}' \\"
echo "     \"$SERVER_URL/api/v1/integrations/feishu/send/report\""
echo ""
echo "4. 配置要求："
echo "   - 在config.local.yaml中配置飞书Webhook URL"
echo "   - 确保机器人有权限发送消息到目标群聊"
echo "   - 建议使用定时任务自动发送日报" 