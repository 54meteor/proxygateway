#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
AI Gateway 测试脚本
用法: python test_gateway.py
"""

import requests
import json
import sys

# 配置
BASE_URL = "http://localhost:8080"
API_KEY = "sk-cp-Swe5I2kwT0_HQVVgjvhrCMLQyPc4cEwJjBEqw3KCKSQKae7k07XOOidQUOtW4muI3OjJoQG2cu9JPW-xwhAlBA5q8m6Aay3jHAPrMQmD0JrH7Yhzk4H8Ixo"


class Colors:
    GREEN = '\033[92m'
    RED = '\033[91m'
    YELLOW = '\033[93m'
    BLUE = '\033[94m'
    RESET = '\033[0m'


def print_success(msg):
    print(f"{Colors.GREEN}✓ {msg}{Colors.RESET}")


def print_error(msg):
    print(f"{Colors.RED}✗ {msg}{Colors.RESET}")


def print_info(msg):
    print(f"{Colors.BLUE}ℹ {msg}{Colors.RESET}")


def test_health():
    """测试健康检查"""
    print_info("测试 /health")
    resp = requests.get(f"{BASE_URL}/health")
    if resp.status_code == 200 and resp.json().get("status") == "ok":
        print_success("健康检查通过")
        return True
    print_error(f"健康检查失败: {resp.text}")
    return False


def test_list_models():
    """测试获取模型列表"""
    print_info("测试 /v1/models")
    resp = requests.get(f"{BASE_URL}/v1/models")
    if resp.status_code == 200:
        data = resp.json()
        models = data.get("data", [])
        print_success(f"获取到 {len(models)} 个模型: {models}")
        return True
    print_error(f"获取模型列表失败: {resp.text}")
    return False


def init_test_user():
    """初始化测试用户"""
    print_info("初始化测试用户...")
    resp = requests.post(f"{BASE_URL}/debug/init")
    if resp.status_code == 200:
        data = resp.json()
        global API_KEY
        API_KEY = data.get("api_key")
        print_success(f"用户初始化成功, API Key: {API_KEY}")
        return True
    print_error(f"初始化失败: {resp.text}")
    return False


def test_chat(model: str = "MiniMax-M2.5", message: str = "你好，请用一句话介绍自己"):
    """测试聊天接口"""
    if not API_KEY:
        print_error("请先初始化用户")
        return False

    print_info(f"测试聊天接口 (模型: {model})")
    
    headers = {
        "Authorization": f"Bearer {API_KEY}",
        "Content-Type": "application/json"
    }
    
    payload = {
        "model": model,
        "messages": [
            {"role": "user", "content": message}
        ]
    }
    
    resp = requests.post(
        f"{BASE_URL}/v1/chat/completions",
        headers=headers,
        json=payload,
        timeout=60
    )
    
    if resp.status_code == 200:
        data = resp.json()
        # 打印响应
        if "choices" in data and len(data["choices"]) > 0:
            content = data["choices"][0]["message"]["content"]
            print_success(f"回复: {content[:100]}...")
            
            # 打印 usage
            if "usage" in data:
                usage = data["usage"]
                print_info(f"Token 使用: prompt={usage.get('prompt_tokens')}, "
                          f"completion={usage.get('completion_tokens')}, "
                          f"total={usage.get('total_tokens')}")
        return True
    else:
        print_error(f"聊天失败: {resp.status_code} - {resp.text}")
        return False


def test_stream_chat():
    """测试流式聊天"""
    if not API_KEY:
        print_error("请先初始化用户")
        return False

    print_info("测试流式聊天...")
    
    headers = {
        "Authorization": f"Bearer {API_KEY}",
        "Content-Type": "application/json"
    }
    
    payload = {
        "model": "MiniMax-M2.5",
        "messages": [
            {"role": "user", "content": "给我讲个笑话"}
        ],
        "stream": True
    }
    
    resp = requests.post(
        f"{BASE_URL}/v1/chat/completions",
        headers=headers,
        json=payload,
        stream=True,
        timeout=60
    )
    
    if resp.status_code == 200:
        print_success("流式响应正常")
        # 流式输出
        for line in resp.iter_lines():
            if line:
                line = line.decode('utf-8')
                if line.startswith('data: '):
                    print(f"  {line[:80]}...")
        return True
    else:
        print_error(f"流式聊天失败: {resp.status_code} - {resp.text}")
        return False


def test_invalid_key():
    """测试无效 API Key"""
    print_info("测试无效 API Key...")
    
    headers = {
        "Authorization": "Bearer invalid-key-12345",
        "Content-Type": "application/json"
    }
    
    payload = {
        "model": "abab6.5s-chat",
        "messages": [{"role": "user", "content": "Hi"}]
    }
    
    resp = requests.post(
        f"{BASE_URL}/v1/chat/completions",
        headers=headers,
        json=payload
    )
    
    if resp.status_code == 401:
        print_success("无效 Key 正确拒绝")
        return True
    else:
        print_error(f"应该返回 401, 实际: {resp.status_code}")
        return False


def test_models():
    """测试不同模型"""
    models = [
        "MiniMax-M2.5",
    ]
    
    print_info(f"测试 {len(models)} 个模型...")
    
    for model in models:
        print(f"\n--- 测试模型: {model} ---")
        if test_chat(model, "说一个字"):
            print_success(f"{model} 可用")
        else:
            print_error(f"{model} 不可用")


def run_all_tests():
    """运行所有测试"""
    print("\n" + "="*50)
    print(" AI Gateway 测试脚本")
    print("="*50 + "\n")
    
    results = []
    
    # 1. 健康检查
    results.append(("健康检查", test_health()))
    
    # 2. 获取模型列表
    results.append(("获取模型列表", test_list_models()))
    
    # 3. 初始化用户
    results.append(("初始化用户", init_test_user()))
    
    # 4. 无效 Key 测试
    results.append(("无效 Key 拒绝", test_invalid_key()))
    
    # 5. 聊天测试
    results.append(("聊天接口", test_chat()))
    
    # # 6. 测试多个模型
    # print("\n" + "="*50)
    # print_info("测试不同模型...")
    # test_models()
    
    # # 总结
    # print("\n" + "="*50)
    # print(" 测试结果汇总")
    # print("="*50)
    
    passed = sum(1 for _, r in results if r)
    total = len(results)
    
    for name, result in results:
        status = "✓ PASS" if result else "✗ FAIL"
        color = Colors.GREEN if result else Colors.RED
        print(f"{color}{status}{Colors.RESET} - {name}")
    
    print(f"\n总计: {passed}/{total} 通过")
    
    return passed == total


if __name__ == "__main__":
    try:
        success = run_all_tests()
        sys.exit(0 if success else 1)
    except requests.exceptions.ConnectionError:
        print_error(f"无法连接到 {BASE_URL}, 请确保服务已启动")
        print_info("运行: server.exe")
        sys.exit(1)
    except Exception as e:
        print_error(f"测试异常: {e}")
        sys.exit(1)
