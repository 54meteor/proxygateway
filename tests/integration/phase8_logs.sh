#!/bin/bash
# AI Gateway 联调测试 - Phase 8: 日志验证
# 执行方式: bash phase8_logs.sh

set -e

BASE_DIR="/mnt/d/aicode/ai-gateway"
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "=========================================="
echo "Phase 8: 日志验证"
echo "=========================================="

# 8.1 gateway.log 检查
echo -e "\n${YELLOW}[8.1] gateway.log 检查${NC}"
if [ -f "$BASE_DIR/gateway.log" ]; then
    echo "gateway.log 存在"
    LINES=$(wc -l < "$BASE_DIR/gateway.log")
    echo "日志行数: $LINES"
    echo -e "${GREEN}✓ gateway.log 正常${NC}"
else
    echo -e "${YELLOW}⚠ gateway.log 不存在${NC}"
fi

# 8.2 logs 目录检查
echo -e "\n${YELLOW}[8.2] logs 目录检查${NC}"
if [ -d "$BASE_DIR/logs" ]; then
    echo "logs 目录存在"
    ls -la "$BASE_DIR/logs/"
    echo -e "${GREEN}✓ logs 目录正常${NC}"
else
    echo -e "${YELLOW}⚠ logs 目录不存在${NC}"
fi

# 8.3 JSON 日志文件检查
echo -e "\n${YELLOW}[8.3] JSON 日志文件检查${NC}"
JSON_LOGS=$(find "$BASE_DIR/logs" -name "chat_*.log" 2>/dev/null | head -5)
if [ -n "$JSON_LOGS" ]; then
    echo "找到 JSON 日志文件:"
    echo "$JSON_LOGS"
    echo -e "${GREEN}✓ JSON 日志存在${NC}"
    
    # 检查最新一个日志文件的内容格式
    LATEST_LOG=$(find "$BASE_DIR/logs" -name "chat_*.log" 2>/dev/null | sort -r | head -1)
    if [ -n "$LATEST_LOG" ]; then
        echo "最新日志文件: $LATEST_LOG"
        echo "前3行内容:"
        head -3 "$LATEST_LOG"
        
        # 验证 JSON 格式
        if head -1 "$LATEST_LOG" | grep -q '^{"request"'; then
            echo -e "${GREEN}✓ JSON 日志格式正确${NC}"
        else
            echo -e "${YELLOW}⚠ JSON 日志格式可能异常${NC}"
        fi
    fi
else
    echo -e "${YELLOW}⚠ 未找到 JSON 日志文件${NC}"
fi

# 8.4 日志内容字段检查
echo -e "\n${YELLOW}[8.4] 日志内容字段检查${NC}"
if [ -n "$LATEST_LOG" ] && [ -f "$LATEST_LOG" ]; then
    echo "检查必要字段..."
    FIELDS=('"request"' '"response"' '"model"' '"tokens"')
    for field in "${FIELDS[@]}"; do
        if grep -q "$field" "$LATEST_LOG" 2>/dev/null; then
            echo -e "  ${GREEN}✓ 包含 $field${NC}"
        else
            echo -e "  ${RED}✗ 缺少 $field${NC}"
        fi
    done
else
    echo -e "${YELLOW}⚠ 无日志文件可检查${NC}"
fi

echo -e "\n=========================================="
echo "Phase 8 完成"
echo "=========================================="
