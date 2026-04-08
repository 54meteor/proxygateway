package auth

import (
	"os"
	"path/filepath"
	"testing"

	"ai-gateway/internal/storage"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestDB(t *testing.T) *storage.DB {
	tmp := filepath.Join(os.TempDir(), "test_auth_"+t.Name()+".db")
	db, err := storage.NewDB(tmp)
	require.NoError(t, err)
	err = db.InitSchema()
	require.NoError(t, err)
	t.Cleanup(func() {
		db.Close()
		os.Remove(tmp)
	})
	return db
}

func TestNewAuthService(t *testing.T) {
	db := newTestDB(t)
	svc := NewAuthService(db)
	assert.NotNil(t, svc)
}

func TestGenerateAPIKey(t *testing.T) {
	db := newTestDB(t)
	svc := NewAuthService(db)

	user, err := db.CreateUser("apikey@test.com")
	require.NoError(t, err)

	rawKey, err := svc.GenerateAPIKey(user.ID, "test-key")
	require.NoError(t, err)
	assert.NotEmpty(t, rawKey)
	assert.Len(t, rawKey, 36) // UUID 格式

	// 验证 Key 可以被找到
	keyHash := HashKey(rawKey)
	found, err := db.GetAPIKeyByHash(keyHash)
	require.NoError(t, err)
	assert.Equal(t, user.ID, found.UserID)
}

func TestValidateAPIKey_Valid(t *testing.T) {
	db := newTestDB(t)
	svc := NewAuthService(db)

	user, err := db.CreateUser("validate@test.com")
	require.NoError(t, err)

	rawKey, err := svc.GenerateAPIKey(user.ID, "test-key")
	require.NoError(t, err)

	userID, err := svc.ValidateAPIKey(rawKey)
	require.NoError(t, err)
	assert.Equal(t, user.ID.String(), userID)
}

func TestValidateAPIKey_Invalid(t *testing.T) {
	db := newTestDB(t)
	svc := NewAuthService(db)

	_, err := svc.ValidateAPIKey("invalid-key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
}

func TestValidateAPIKey_InactiveKey(t *testing.T) {
	db := newTestDB(t)
	svc := NewAuthService(db)

	user, err := db.CreateUser("inactive@test.com")
	require.NoError(t, err)

	rawKey, err := svc.GenerateAPIKey(user.ID, "test-key")
	require.NoError(t, err)

	// 将 key 设为 inactive
	keyHash := HashKey(rawKey)
	_, err = db.Exec("UPDATE api_keys SET is_active = 0 WHERE key_hash = ?", keyHash)
	require.NoError(t, err)

	// 由于 GetAPIKeyByHash 查询条件包含 is_active=1，
	// 已停用的 key 不会被找到，返回 "invalid API key"
	_, err = svc.ValidateAPIKey(rawKey)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
}

func TestValidateAPIKeyFull_Valid(t *testing.T) {
	db := newTestDB(t)
	svc := NewAuthService(db)

	user, err := db.CreateUser("full@test.com")
	require.NoError(t, err)

	rawKey, err := svc.GenerateAPIKey(user.ID, "test-key")
	require.NoError(t, err)

	userID, apiKeyID, err := svc.ValidateAPIKeyFull(rawKey)
	require.NoError(t, err)
	assert.Equal(t, user.ID.String(), userID)
	assert.NotEmpty(t, apiKeyID)
}

func TestValidateAPIKeyFull_Invalid(t *testing.T) {
	db := newTestDB(t)
	svc := NewAuthService(db)

	_, _, err := svc.ValidateAPIKeyFull("invalid-key")
	assert.Error(t, err)
}

func TestHashKey(t *testing.T) {
	key := "test-api-key-12345"
	hash := HashKey(key)

	assert.NotEmpty(t, hash)
	assert.Len(t, hash, 64) // SHA256 hex 格式

	// 相同输入产生相同输出
	hash2 := HashKey(key)
	assert.Equal(t, hash, hash2)

	// 不同输入产生不同输出
	hash3 := HashKey("different-key")
	assert.NotEqual(t, hash, hash3)
}

func TestHashKey_DifferentKeys(t *testing.T) {
	keys := []string{
		"key-1",
		"key-2",
		"a-complete-different-key-altogether",
	}

	hashes := make([]string, len(keys))
	for i, k := range keys {
		hashes[i] = HashKey(k)
	}

	// 所有 hash 都不同
	for i := 0; i < len(hashes); i++ {
		for j := i + 1; j < len(hashes); j++ {
			assert.NotEqual(t, hashes[i], hashes[j], "Hash collision between key %d and %d", i, j)
		}
	}
}

func TestCreateTestUser(t *testing.T) {
	db := newTestDB(t)
	svc := NewAuthService(db)

	user, err := svc.CreateTestUser()
	require.NoError(t, err)
	assert.NotNil(t, user)
	assert.Contains(t, user.Email, "test-")
	assert.Contains(t, user.Email, "@example.com")
}

func TestListAllAPIKeys(t *testing.T) {
	db := newTestDB(t)
	svc := NewAuthService(db)

	user, err := db.CreateUser("listkeys@test.com")
	require.NoError(t, err)

	// 创建多个 key
	_, err = svc.GenerateAPIKey(user.ID, "key-1")
	require.NoError(t, err)
	_, err = svc.GenerateAPIKey(user.ID, "key-2")
	require.NoError(t, err)

	keys, err := svc.ListAllAPIKeys()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(keys), 2)
}

func TestDebugCheckKey(t *testing.T) {
	db := newTestDB(t)
	svc := NewAuthService(db)

	user, err := db.CreateUser("debugcheck@test.com")
	require.NoError(t, err)

	rawKey, err := svc.GenerateAPIKey(user.ID, "debug-key")
	require.NoError(t, err)

	keyHash := HashKey(rawKey)
	exists, err := svc.DebugCheckKey(keyHash)
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = svc.DebugCheckKey("nonexistent-hash")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestValidateAPIKey_Empty(t *testing.T) {
	db := newTestDB(t)
	svc := NewAuthService(db)

	_, err := svc.ValidateAPIKey("")
	assert.Error(t, err)
}

func TestValidateAPIKey_UuidFormat(t *testing.T) {
	db := newTestDB(t)
	svc := NewAuthService(db)

	// 尝试使用真实 UUID 格式的 key
	validUUID := uuid.New().String()
	userID, err := svc.ValidateAPIKey(validUUID)
	assert.Error(t, err) // 不存在于数据库
	_ = userID
}
