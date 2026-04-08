package adapter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"ai-gateway/internal/config"
	"ai-gateway/internal/model"
)

// MiniMaxAdapter MiniMax 模型适配器
// 实现与 MiniMax API 的交互
type MiniMaxAdapter struct {
	cfg *config.Config // 全局配置
}

// NewMiniMaxAdapter 创建 MiniMax 适配器
func NewMiniMaxAdapter(cfg *config.Config) *MiniMaxAdapter {
	return &MiniMaxAdapter{
		cfg: cfg,
	}
}

// ChatComplete 处理聊天完成请求
// 1. 构造 MiniMax API 请求
// 2. 发送 HTTP 请求
// 3. 解析响应并转换格式
func (a *MiniMaxAdapter) ChatComplete(req model.ChatRequest) (*model.ChatResponse, error) {
	// 1. 构建 API URL
	url := a.cfg.Models.MiniMax.BaseURL + "/chat/completions"
	
	// 2. 转换消息格式（适配 MiniMax API）
	messages := make([]map[string]interface{}, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		}
	}
	
	// 3. 构造请求体
	body := map[string]interface{}{
		"model":    req.Model,
		"messages": messages,
	}
	
	// 添加可选参数
	if req.Temperature > 0 {
		body["temperature"] = req.Temperature
	}
	if req.MaxTokens > 0 {
		body["max_tokens"] = req.MaxTokens
	}
	
	// 序列化为 JSON
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	
	// 4. 创建 HTTP 请求
	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	
	// 设置请求头
	httpReq.Header.Set("Authorization", "Bearer "+a.cfg.Models.MiniMax.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	
	// 5. 发送请求（带超时）
	client := &http.Client{Timeout: time.Duration(a.cfg.Models.MiniMax.Timeout) * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	// 6. 读取响应体
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	
	// 7. 检查响应状态码
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("error: status code %d, body: %s", resp.StatusCode, string(respBody))
	}
	
	// 8. 解析 JSON 响应
	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}
	
	// 9. 转换为模型响应格式
	choices := result["choices"].([]interface{})
	firstChoice := choices[0].(map[string]interface{})
	message := firstChoice["message"].(map[string]interface{})
	
	response := &model.ChatResponse{
		ID:      fmt.Sprintf("chatcmpl-%d", result["created"]),
		Object:  "chat.completion",
		Created: int64(result["created"].(float64)),
		Model:   result["model"].(string),
		Choices: []model.Choice{
			{
				Index: 0,
				Message: model.ChatMessage{
					Role:    message["role"].(string),
					Content: cleanThinking(message["content"].(string)), // 清理思维链标签
				},
				FinishReason: firstChoice["finish_reason"].(string),
			},
		},
	}
	
	// 10. 解析用量信息
	if usage, ok := result["usage"].(map[string]interface{}); ok {
		response.Usage = model.Usage{
			PromptTokens:     int(usage["prompt_tokens"].(float64)),
			CompletionTokens: int(usage["completion_tokens"].(float64)),
			TotalTokens:      int(usage["total_tokens"].(float64)),
		}
	}
	
	return response, nil
}

// CountTokens 计算 Token 数量
// 注意：MiniMax API 需要使用专门的端点计算，这里简化处理
func (a *MiniMaxAdapter) CountTokens(model, text string) (int, error) {
	// 简单估算：中文约 1.5 字符/token，英文约 4 字符/token
	// 实际应调用 API 获取准确值
	return len(text) / 2, nil
}

// GetModelName 获取模型名称
func (a *MiniMaxAdapter) GetModelName() string {
	return "MiniMax-M2.5"
}

// Embeddings 处理文本向量化请求
// 1. 构造 MiniMax Embeddings API 请求
// 2. 发送 HTTP 请求
// 3. 解析响应并转换格式
func (a *MiniMaxAdapter) Embeddings(req model.EmbeddingRequest) (*model.EmbeddingResponse, error) {
	// 1. 构建 API URL
	url := a.cfg.Models.MiniMax.BaseURL + "/embeddings"

	// 2. 构造请求体
	body := map[string]interface{}{
		"input": req.Input,
	}
	if req.Model != "" {
		body["model"] = req.Model
	} else {
		body["model"] = "embedding-model"
	}

	// 序列化为 JSON
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	// 3. 创建 HTTP 请求
	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	// 设置请求头
	httpReq.Header.Set("Authorization", "Bearer "+a.cfg.Models.MiniMax.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	// 4. 发送请求（带超时）
	client := &http.Client{Timeout: time.Duration(a.cfg.Models.MiniMax.Timeout) * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 5. 读取响应体
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// 6. 检查响应状态码
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("error: status code %d, body: %s", resp.StatusCode, string(respBody))
	}

	// 7. 解析 JSON 响应
	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	// 8. 转换为 OpenAI 格式的响应
	response := &model.EmbeddingResponse{
		Object: "list",
		Model:  req.Model,
	}

	// 解析 data 数组
	if data, ok := result["data"].([]interface{}); ok {
		for i, item := range data {
			itemMap := item.(map[string]interface{})
			embedding := itemMap["embedding"].([]interface{})
			floatEmbedding := make([]float64, len(embedding))
			for j, v := range embedding {
				floatEmbedding[j] = v.(float64)
			}
			response.Data = append(response.Data, model.EmbeddingData{
				Object:    "embedding",
				Embedding: floatEmbedding,
				Index:     int(itemMap["index"].(float64)),
			})
			_ = i // suppress unused warning
		}
	}

	// 解析 usage
	if usage, ok := result["usage"].(map[string]interface{}); ok {
		response.Usage = model.EmbeddingUsage{
			PromptTokens: int(usage["prompt_tokens"].(float64)),
			TotalTokens:  int(usage["total_tokens"].(float64)),
		}
	}

	return response, nil
}

// cleanThinking 清理思维链标签
// MiniMax 返回的内容可能包含 <think> 思考标签，需要清理
func cleanThinking(content string) string {
	// 移除 <think> 思考标签及其内容
	content = strings.ReplaceAll(content, "<think>\n", "")
	content = strings.ReplaceAll(content, "</think>\n", "")
	content = strings.ReplaceAll(content, "<think>", "")
	content = strings.ReplaceAll(content, "</think>", "")
	// 清理多余空行
	for strings.Contains(content, "\n\n\n") {
		content = strings.ReplaceAll(content, "\n\n\n", "\n\n")
	}
	return strings.TrimSpace(content)
}
