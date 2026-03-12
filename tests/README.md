# AI Gateway 测试指南

## 前置条件

1. 安装 Python 依赖:
```bash
cd tests
pip install -r requirements.txt
```

2. 启动服务:
```bash
# 在 ai-gateway 目录
set MINIMAX_API_KEY=sk-cp-Swe5I2kwT0_HQVVgjvhrCMLQyPc4cEwJjBEqw3KCKSQKae7k07XOOidQUOtW4muI3OjJoQG2cu9JPW-xwhAlBA5q8m6Aay3jHAPrMQmD0JrH7Yhzk4H8Ixo
server.exe
```

## 运行测试

```bash
cd tests
python test_gateway.py
```

## 测试项目

| 测试项 | 说明 |
|--------|------|
| 健康检查 | /health 接口 |
| 获取模型列表 | /v1/models 接口 |
| 初始化用户 | /debug/init 接口 |
| 无效 Key 拒绝 | 验证 API Key 鉴权 |
| 聊天接口 | /v1/chat/completions 接口 |
| 多模型测试 | 测试不同模型 |

## 预期输出

```
==================================================
 AI Gateway 测试脚本
==================================================

ℹ 测试 /health
✓ 健康检查通过
ℹ 测试 /v1/models
✓ 获取到 3 个模型: ['MiniMax-Text-01', 'abab6.5s-chat', 'abab6.5g-chat']
ℹ 初始化测试用户...
✓ 用户初始化成功, API Key: xxx-xxx-xxx
ℹ 测试无效 API Key...
✓ 无效 Key 正确拒绝
ℹ 测试聊天接口 (模型: abab6.5s-chat)
✓ 回复: 你好！我是...

==================================================
 测试结果汇总
==================================================

✓ PASS - 健康检查
✓ PASS - 获取模型列表
✓ PASS - 初始化用户
✓ PASS - 无效 Key 拒绝
✓ PASS - 聊天接口

总计: 5/5 通过
```
