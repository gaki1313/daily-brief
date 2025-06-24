#!/bin/bash

# 测试新的GitLab飞书推送API
# 这个脚本用于验证GitLab日报能正确推送到飞书，并且显示的是GitLab数据而不是本地Git数据

echo "=== 测试GitLab飞书推送API ==="
echo "当前时间: $(date)"
echo ""

# API基础URL
BASE_URL="http://localhost:8080/api/v1"

# 测试日期（今天）
DATE=$(date +%Y-%m-%d)
echo "测试日期: $DATE"
echo ""

echo "=== 1. 检查GitLab集成状态 ==="
curl -s "$BASE_URL/integrations/gitlab/status" | jq . || echo "获取GitLab状态失败"
echo ""

echo "=== 2. 检查飞书集成状态 ==="
curl -s "$BASE_URL/integrations/feishu/status" | jq . || echo "获取飞书状态失败"
echo ""

echo "=== 3. 同步GitLab提交数据 ==="
curl -s -X POST "$BASE_URL/integrations/gitlab/sync/$DATE" | jq . || echo "同步GitLab数据失败"
echo ""

echo "=== 4. 生成GitLab日报 ==="
curl -s -X POST "$BASE_URL/reports/gitlab/generate" \
     -H "Content-Type: application/json" \
     -d "{
       \"date\": \"$DATE\",
       \"template\": \"markdown\"
     }" | jq . || echo "生成GitLab日报失败"
echo ""

echo "=== 5. 预览GitLab日报内容 ==="
curl -s "$BASE_URL/reports/gitlab/preview?date=$DATE&template=markdown" | jq . || echo "预览GitLab日报失败"
echo ""

echo "=== 6. 使用新API推送GitLab日报到飞书 ==="
echo "正在发送GitLab日报到飞书..."
SEND_RESULT=$(curl -s -X POST "$BASE_URL/integrations/feishu/send/gitlab-report" \
     -H "Content-Type: application/json" \
     -d "{
       \"date\": \"$DATE\",
       \"template\": \"markdown\",
       \"message_type\": \"text\"
     }")

echo "发送结果:"
echo "$SEND_RESULT" | jq .
echo ""

# 检查发送是否成功
SUCCESS=$(echo "$SEND_RESULT" | jq -r '.code')
if [ "$SUCCESS" = "200" ]; then
    echo "✅ GitLab日报发送成功！"
    MESSAGE_ID=$(echo "$SEND_RESULT" | jq -r '.data.message_id')
    DURATION=$(echo "$SEND_RESULT" | jq -r '.data.duration')
    echo "   消息ID: $MESSAGE_ID"
    echo "   发送耗时: ${DURATION}ms"
    echo "   数据源: $(echo "$SEND_RESULT" | jq -r '.data.data_source')"
else
    echo "❌ GitLab日报发送失败"
    echo "错误信息: $(echo "$SEND_RESULT" | jq -r '.message')"
fi
echo ""

echo "=== 7. 获取发送日志 ==="
curl -s "$BASE_URL/integrations/logs?platform=feishu&limit=5" | jq . || echo "获取日志失败"
echo ""

echo "=== 8. 对比标准日报推送 ==="
echo "为了对比，现在使用标准API发送日报..."
STANDARD_RESULT=$(curl -s -X POST "$BASE_URL/integrations/feishu/send/report" \
     -H "Content-Type: application/json" \
     -d "{
       \"date\": \"$DATE\",
       \"template\": \"markdown\",
       \"message_type\": \"text\"
     }")

echo "标准日报发送结果:"
echo "$STANDARD_RESULT" | jq .
echo ""

echo "=== 测试完成 ==="
echo "请检查飞书群聊，确认："
echo "1. 收到了两条消息"
echo "2. 第一条（GitLab专用）显示的是GitLab提交信息"
echo "3. 第二条（标准）可能混合了本地Git和GitLab信息"
echo "4. 注意对比两条消息的提交来源差异" 