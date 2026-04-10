#!/bin/bash
# AI Gateway 联调测试 - Phase 4: 请求转发
# 执行方式: bash phase4_forward.sh

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
echo "Phase 4: 请求转发测试"
echo "=========================================="

# 4.1 基础聊天请求
echo -e "\n${YELLOW}[4.1] 基础聊天请求${NC}"
RESP=$(curl -s -X POST http://localhost:8080/v1/chat/completions \
    -H "Authorization: Bearer $API_KEY" \
    -H "Content-Type: application/json" \
    -d '{"model":"MiniMax-M2.5","messages":[{"role":"user","content":"用一句话介绍自己"}]}')
echo "响应: $RESP"

# 4.2 检查响应格式
echo -e "\n${YELLOW}[4.2] 检查响应格式${NC}"
if echo "$RESP" | grep -qi '"Choices"'; then
    echo -e "${GREEN}✓ 包含 choices 字段${NC}"
else
    echo -e "${RED}✗ 缺少 choices 字段${NC}"
fi

# 4.3 检查 model 字段
echo -e "\n${YELLOW}[4.3] 检查 model 字段${NC}"
if echo "$RESP" | grep -q '"model":"MiniMax-M2.5"'; then
    echo -e "${GREEN}✓ model 字段正确${NC}"
else
    echo -e "${RED}✗ model 字段异常${NC}"
fi

# 4.4 检查 usage 字段
echo -e "\n${YELLOW}[4.4] 检查 usage 字段${NC}"
if echo "$RESP" | grep -qi '"Usage"'; then
    echo -e "${GREEN}✓ 包含 usage 字段${NC}"
    USAGE=$(echo "$RESP" | grep -o '"usage":{[^}]*}')
    echo "Usage: $USAGE"
else
    echo -e "${RED}✗ 缺少 usage 字段${NC}"
fi

# 4.5 system message 测试
echo -e "\n${YELLOW}[4.5] System Message 测试${NC}"
RESP=$(curl -s -X POST http://localhost:8080/v1/chat/completions \
    -H "Authorization: Bearer $API_KEY" \
    -H "Content-Type: application/json" \
    -d '{"model":"MiniMax-M2.5","messages":[{"role":"system","content":"你是一个友好的助手"},{"role":"user","content":"你好"}]}')
echo "响应: $RESP"

if echo "$RESP" | grep -qi '"Choices"'; then
    echo -e "${GREEN}✓ System message 请求成功${NC}"
else
    echo -e "${RED}✗ System message 请求失败${NC}"
fi

echo -e "\n=========================================="
echo "Phase 4 完成"
echo "=========================================="
