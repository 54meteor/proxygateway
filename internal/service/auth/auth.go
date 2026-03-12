package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"ai-gateway/internal/logger"
	"ai-gateway/internal/model"
	"ai-gateway/internal/storage"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// AuthService 认证服务
// 提供用户和 API Key 的认证管理功能
type AuthService struct {
	db *storage.DB // 数据库实例
}

// NewAuthService 创建认证服务
func NewAuthService(db *storage.DB) *AuthService {
	return &AuthService{db: db}
}

// CreateTestUser 创建测试用户
// 用于开发测试，生成随机邮箱
func (s *AuthService) CreateTestUser() (*model.User, error) {
	// 生成随机邮箱
	email := fmt.Sprintf("test-%s@example.com", uuid.New().String()[:8])
	return s.db.CreateUser(email)
}

// GenerateAPIKey 生成 API Key
// 参数: userID 用户ID, name Key名称
// 返回: (string, error) 生成的原始 Key 和错误
func (s *AuthService) GenerateAPIKey(userID uuid.UUID, name string) (string, error) {
	// 生成随机 UUID 作为 Key
	rawKey := uuid.New().String()
	// 计算哈希存储
	keyHash := HashKey(rawKey)

	// 存储到数据库
	_, err := s.db.CreateAPIKey(userID.String(), keyHash, name, 60)
	if err != nil {
		return "", err
	}

	return rawKey, nil
}

// ValidateAPIKey 验证 API Key（仅返回 userID）
func (s *AuthService) ValidateAPIKey(rawKey string) (string, error) {
	keyHash := HashKey(rawKey)
	logger.Info("[AUTH] ValidateAPIKey: rawKey=%s, hash=%s\n", rawKey, keyHash)

	// 查询数据库
	apiKey, err := s.db.GetAPIKeyByHash(keyHash)
	if err != nil {
		logger.Info("[AUTH] ValidateAPIKey: GetAPIKeyByHash error=%v\n", err)
		return "", fmt.Errorf("invalid API key")
	}

	// 检查是否激活
	if !apiKey.IsActive {
		return "", fmt.Errorf("API key is inactive")
	}

	logger.Info("[AUTH] ValidateAPIKey: found userID=%s\n", apiKey.UserID.String())
	return apiKey.UserID.String(), nil
}

// ValidateAPIKeyFull 验证 API Key 并返回完整信息
// 参数: rawKey 原始 API Key
// 返回: (userID, apiKeyID, error) 用户ID、API Key ID 和错误
func (s *AuthService) ValidateAPIKeyFull(rawKey string) (userID, apiKeyID string, err error) {
	keyHash := HashKey(rawKey)
	logger.Info("[AUTH] ValidateAPIKeyFull: rawKey=%s, hash=%s\n", rawKey, keyHash)

	// 查询数据库
	apiKey, err := s.db.GetAPIKeyByHash(keyHash)
	if err != nil {
		logger.Info("[AUTH] ValidateAPIKeyFull: GetAPIKeyByHash error=%v\n", err)
		return "", "", fmt.Errorf("invalid API key")
	}

	// 检查是否激活
	if !apiKey.IsActive {
		return "", "", fmt.Errorf("API key is inactive")
	}

	logger.Info("[AUTH] ValidateAPIKeyFull: found userID=%s, apiKeyID=%s\n", apiKey.UserID.String(), apiKey.ID.String())
	return apiKey.UserID.String(), apiKey.ID.String(), nil
}

// HashKey 对 Key 进行哈希
// 使用 SHA256 哈希原始 Key 后存储
func HashKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// HashPassword 哈希密码
// 使用 bcrypt 算法加密密码
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword 验证密码
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// ListAllAPIKeys 列出所有 API Keys（调试用）
func (s *AuthService) ListAllAPIKeys() ([]map[string]string, error) {
	return s.db.ListAllAPIKeys()
}

// DebugCheckKey 检查 key 是否存在（调试用）
func (s *AuthService) DebugCheckKey(hash string) (bool, error) {
	return s.db.DebugCheckKey(hash)
}
