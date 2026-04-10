#!/bin/bash
# AI Gateway 联调测试 - Phase 6: Go 管理后台
# 执行方式: bash phase6_admin.sh

set -e

BASE_URL="http://localhost:8080"
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "=========================================="
echo "Phase 6: Go 管理后台测试"
echo "=========================================="

# 6.1 仪表盘
echo -e "\n${YELLOW}[6.1] 仪表盘 /admin-dashboard${NC}"
RESP=$(curl -s http://localhost:8080/admin-dashboard)
if echo "$RESP" | grep -q 'html\|<!DOCTYPE'; then
    echo -e "${GREEN}✓ 仪表盘页面正常${NC}"
else
    echo -e "${RED}✗ 仪表盘页面异常${NC}"
fi

# 6.2 用户列表
echo -e "\n${YELLOW}[6.2] 用户列表 /admin/users${NC}"
RESP=$(curl -s http://localhost:8080/admin/users)
echo "响应: $RESP"

if echo "$RESP" | grep -q '"data"'; then
    echo -e "${GREEN}✓ 用户列表正常${NC}"
else
    echo -e "${RED}✗ 用户列表异常${NC}"
fi

# 6.3 API Key 列表
echo -e "\n${YELLOW}[6.3] API Key 列表 /admin/keys${NC}"
RESP=$(curl -s http://localhost:8080/admin/keys)
echo "响应: $RESP"

if echo "$RESP" | grep -q '"data"'; then
    echo -e "${GREEN}✓ Key 列表正常${NC}"
else
    echo -e "${RED}✗ Key 列表异常${NC}"
fi

# 6.4 用量统计
echo -e "\n${YELLOW}[6.4] 用量统计 /admin/usage${NC}"
RESP=$(curl -s http://localhost:8080/admin/usage)
echo "响应: $RESP"

if echo "$RESP" | grep -q '"data"'; then
    echo -e "${GREEN}✓ 用量统计正常${NC}"
else
    echo -e "${RED}✗ 用量统计异常${NC}"
fi

# 6.5 充值接口
echo -e "\n${YELLOW}[6.5] 余额充值测试${NC}"
# 从 /admin/users 获取 user_id
USERS=$(curl -s http://localhost:8080/admin/users)
USER_ID=$(echo "$USERS" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
echo "User ID: $USER_ID"

if [ -n "$USER_ID" ]; then
    RESP=$(curl -s -X POST http://localhost:8080/admin/api/user/recharge \
        -H "Content-Type: application/json" \
        -d "{\"user_id\":\"$USER_ID\",\"amount\":100}")
    echo "充值响应: $RESP"
    
    if echo "$RESP" | grep -q '"balance"'; then
        echo -e "${GREEN}✓ 充值成功${NC}"
    else
        echo -e "${RED}✗ 充值失败${NC}"
    fi
fi

# 6.6 模型列表
echo -e "\n${YELLOW}[6.6] 模型列表 /admin/models${NC}"
RESP=$(curl -s http://localhost:8080/admin/models)
echo "响应: $RESP"

if echo "$RESP" | grep -q '"data"'; then
    echo -e "${GREEN}✓ 模型列表正常${NC}"
else
    echo -e "${RED}✗ 模型列表异常${NC}"
fi

echo -e "\n=========================================="
echo "Phase 6 完成"
echo "=========================================="
