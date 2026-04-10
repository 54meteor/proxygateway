#!/bin/bash
# AI Gateway 联调测试 - Phase 5: 计费
# 执行方式: bash phase5_billing.sh

set -e

BASE_URL="http://localhost:8080"
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# 加载 Phase 2 保存的 API Key
if [ -f /tmp/test_api_key.txt ]; then
    source /tmp/test_api_key.txt
else
    echo -e "${RED}请先运行 Phase 2 初始化用户${NC}"
    exit 1
fi

echo "=========================================="
echo "Phase 5: 计费测试"
echo "=========================================="

# 5.1 查询当前余额
echo -e "\n${YELLOW}[5.1] 查询当前余额${NC}"
# 使用 /debug/check 端点查询 Key 状态
RESP=$(curl -s http://localhost:8080/debug/check?key=$API_KEY)
echo "响应: $RESP"

# 提取余额
if echo "$RESP" | grep -q '"balance"'; then
    BALANCE=$(echo "$RESP" | grep -o '"balance":[0-9.]*' | cut -d':' -f2)
    echo -e "${GREEN}✓ 当前余额: $BALANCE${NC}"
fi

# 5.2 执行请求后查询余额变化
echo -e "\n${YELLOW}[5.2] 执行请求后检查余额变化${NC}"

# 记录请求前余额
BEFORE_BALANCE=$BALANCE

# 发送一个请求
echo "发送测试请求..."
curl -s -X POST http://localhost:8080/v1/chat/completions \
    -H "Authorization: Bearer $API_KEY" \
    -H "Content-Type: application/json" \
    -d '{"model":"MiniMax-M2.5","messages":[{"role":"user","content":"测试"}]}' > /dev/null

sleep 1

# 查询请求后余额
RESP=$(curl -s http://localhost:8080/debug/check?key=$API_KEY)
AFTER_BALANCE=$(echo "$RESP" | grep -o '"balance":[0-9.]*' | cut -d':' -f2)
echo "请求前余额: $BEFORE_BALANCE"
echo "请求后余额: $AFTER_BALANCE"

if [ "$BEFORE_BALANCE" != "$AFTER_BALANCE" ]; then
    echo -e "${GREEN}✓ 余额已扣除${NC}"
else
    echo -e "${YELLOW}⚠ 余额未变化（可能为0或查询失败）${NC}"
fi

# 5.3 用量记录查询
echo -e "\n${YELLOW}[5.3] 用量记录查询${NC}"
RESP=$(curl -s -H "Authorization: Bearer $API_KEY" http://localhost:8080/v1/usage)
echo "响应: $RESP"

if echo "$RESP" | grep -q '"usage"'; then
    echo -e "${GREEN}✓ 用量查询成功${NC}"
else
    echo -e "${RED}✗ 用量查询失败${NC}"
fi

echo -e "\n=========================================="
echo "Phase 5 完成"
echo "=========================================="
