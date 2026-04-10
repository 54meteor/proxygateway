-- chat.lua - ChatCompletions 压测脚本
-- 用法: wrk -t4 -c50 -d30s --latency -s chat.lua http://localhost:8080/v1/chat/completions

wrk.method = "POST"
wrk.headers["Content-Type"] = "application/json"
wrk.headers["Authorization"] = "Bearer test-api-key-12345678"

local counter = 0
local request_id = 0

request = function()
    counter = counter + 1
    request_id = request_id + 1
    
    local body = string.format([[{
        "model": "MiniMax-Text-01",
        "messages": [
            {"role": "system", "content": "You are a helpful assistant."},
            {"role": "user", "content": "Hello, this is request %d. Tell me a short joke."}
        ],
        "max_tokens": 100,
        "temperature": 0.7
    }]], counter)
    
    return wrk.format(nil, nil, nil, body)
end

response = function(status, headers, body)
    if status ~= 200 then
        io.write(string.format("ERROR: status=%d body=%s\n", status, body))
    end
end

done = function(summary, latency, requests)
    io.write("\n=== 压测结果 ===\n")
    io.write(string.format("总请求数: %d\n", summary.requests))
    io.write(string.format("总耗时: %.2fs\n", summary.duration / 1000000))
    io.write(string.format("QPS: %.2f\n", summary.requests / (summary.duration / 1000000)))
    io.write(string.format("错误率: %.2f%%\n", (summary.errors / summary.requests) * 100))
    io.write("\n延迟分布:\n")
    io.write(string.format("  P50: %.2fms\n", latency:percentile(50) / 1000))
    io.write(string.format("  P90: %.2fms\n", latency:percentile(90) / 1000))
    io.write(string.format("  P99: %.2fms\n", latency:percentile(99) / 1000))
    io.write(string.format("  P999: %.2fms\n", latency:percentile(99.9) / 1000))
end
