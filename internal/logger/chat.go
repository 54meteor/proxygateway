package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ChatLogger 聊天日志记录器
// 记录每次请求的完整 JSON 到独立文件
type ChatLogger struct {
	logDir string
}

// NewChatLogger 创建聊天日志记录器
// 参数: logDir 日志目录
func NewChatLogger(logDir string) *ChatLogger {
	// 创建目录
	os.MkdirAll(logDir, 0755)
	
	return &ChatLogger{
		logDir: logDir,
	}
}

// LogRequest 记录请求
// 参数: userID 用户ID, apiKeyID API Key ID, reqJSON 请求JSON, respJSON 响应JSON
func (l *ChatLogger) LogRequest(userID, apiKeyID, model, reqJSON, respJSON string, promptTokens, completionTokens int, cost float64) {
	// 按日期创建日志文件
	dateStr := time.Now().Format("2006-01-02")
	logFile := filepath.Join(l.logDir, fmt.Sprintf("chat_%s.log", dateStr))
	
	// 打开文件（不存在则创建）
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Printf("Failed to open chat log file: %v\n", err)
		return
	}
	defer f.Close()
	
	// 构造日志内容
	logEntry := fmt.Sprintf(`{
  "time": "%s",
  "user_id": "%s",
  "api_key_id": "%s",
  "model": "%s",
  "prompt_tokens": %d,
  "completion_tokens": %d,
  "total_tokens": %d,
  "cost": %.6f,
  "request": %s,
  "response": %s
}
`,
		time.Now().Format("2006-01-02 15:04:05"),
		userID,
		apiKeyID,
		model,
		promptTokens,
		completionTokens,
		promptTokens + completionTokens,
		cost,
		l.formatJSON(reqJSON),
		l.formatJSON(respJSON),
	)
	
	// 写入文件
	f.WriteString(logEntry)
}

// formatJSON 格式化 JSON 字符串
func (l *ChatLogger) formatJSON(jsonStr string) string {
	if jsonStr == "" {
		return "{}"
	}
	
	// 尝试解析并重新格式化
	var v interface{}
	if err := json.Unmarshal([]byte(jsonStr), &v); err != nil {
		// 解析失败，返回原始内容
		return jsonStr
	}
	
	// 重新序列化为格式化 JSON
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return jsonStr
	}
	return string(b)
}
