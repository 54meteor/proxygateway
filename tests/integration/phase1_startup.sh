#!/bin/bash
# AI Gateway 联调测试 - Phase 1: 服务启动
# 执行方式: bash phase1_startup.sh

set -e

BASE_URL="http://localhost:8080"
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "=========================================="
echo "Phase 1: 服务启动验证"
echo "=========================================="

# 1.1 服务启动检查
echo -e "\n${YELLOW}[1.1] 检查 server 进程${NC}"
if pgrep -f "./server" > /dev/null; then
    echo -e "${GREEN}✓ server 进程运行中${NC}"
else
    echo -e "${RED}✗ server 未运行${NC}"
    exit 1
fi

# 1.2 Health 检查
echo -e "\n${YELLOW}[1.2] Health 检查${NC}"
HEALTH_RESP=$(curl -s http://localhost:8080/health)
echo "响应: $HEALTH_RESP"

if echo "$HEALTH_RESP" | grep -q '"status":"ok"'; then
    echo -e "${GREEN}✓ Health 检查通过${NC}"
else
    echo -e "${RED}✗ Health 检查失败${NC}"
    exit 1
fi

# 1.3 检查数据库状态
echo -e "\n${YELLOW}[1.3] 数据库状态${NC}"
if echo "$HEALTH_RESP" | grep -q '"database":{"status":"ok"}'; then
    echo -e "${GREEN}✓ 数据库正常${NC}"
else
    echo -e "${RED}✗ 数据库异常${NC}"
fi

# 1.4 检查 MiniMax 上游
echo -e "\n${YELLOW}[1.4] MiniMax 上游状态${NC}"
if echo "$HEALTH_RESP" | grep -q '"minimax":{"status":"ok"'; then
    echo -e "${GREEN}✓ MiniMax 上游正常${NC}"
else
    echo -e "${RED}✗ MiniMax 上游异常${NC}"
fi

echo -e "\n=========================================="
echo "Phase 1 完成"
echo "=========================================="
