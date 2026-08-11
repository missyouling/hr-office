package storage

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
)

// encryptPrefix 加密数据前缀，用于区分明文与密文
const encryptPrefix = "ENC:"

// 包级 AES 密钥缓存（启动时从 SYSTEM_AES_KEY 加载）
var aesKey []byte

func init() {
	key, err := GetAESKey()
	if err != nil {
		log.Printf("[StorageEncryption] SYSTEM_AES_KEY not configured, storing and reading configs as plaintext: %v", err)
		return
	}
	aesKey = key
	log.Printf("[StorageEncryption] AES-256-GCM encryption enabled for storage config credentials")
}

// 需要脱敏的敏感字段名
var maskFields = map[string]bool{
	"access_key":    true,
	"secret_key":    true,
	"password":      true,
	"access_token":  true,
	"refresh_token": true,
	"token":         true,
	"secret":        true,
	"api_key":       true,
	"private_key":   true,
}

// GetAESKey 从环境变量 SYSTEM_AES_KEY 读取并验证 32 字节长度
func GetAESKey() ([]byte, error) {
	keyStr := os.Getenv("SYSTEM_AES_KEY")
	if keyStr == "" {
		return nil, errors.New("SYSTEM_AES_KEY not set")
	}
	key := []byte(keyStr)
	if len(key) != 32 {
		return nil, fmt.Errorf("SYSTEM_AES_KEY must be 32 bytes, got %d", len(key))
	}
	return key, nil
}

// EncryptConfig 使用 AES-256-GCM 加密配置 JSON（用于入库）
// 若 aesKey 为空，则降级为明文存储（打印 warn 日志）
func EncryptConfig(configJSON []byte, aesKey []byte) ([]byte, error) {
	if len(aesKey) == 0 {
		log.Printf("[StorageEncryption] WARN: no AES key, storing config as plaintext")
		return configJSON, nil
	}
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}
	ciphertext := gcm.Seal(nonce, nonce, configJSON, nil)
	result := encryptPrefix + base64.StdEncoding.EncodeToString(ciphertext)
	return []byte(result), nil
}

// DecryptConfig 解密配置 JSON（用于运行时读取）
// 若数据不以 "ENC:" 开头，直接返回原文（兼容旧明文数据）
func DecryptConfig(encrypted []byte, aesKey []byte) ([]byte, error) {
	if !bytes.HasPrefix(encrypted, []byte(encryptPrefix)) {
		return encrypted, nil
	}
	if len(aesKey) == 0 {
		return nil, errors.New("cannot decrypt: SYSTEM_AES_KEY not configured, but data is encrypted")
	}
	b64Data := encrypted[len(encryptPrefix):]
	ciphertext, err := base64.StdEncoding.DecodeString(string(b64Data))
	if err != nil {
		return nil, fmt.Errorf("base64 decode encrypted config: %w", err)
	}
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher for decrypt: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM for decrypt: %w", err)
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short for GCM")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("AES-GCM decrypt: %w", err)
	}
	return plaintext, nil
}

// MaskConfig 脱敏配置 JSON，将敏感字段替换为 "***"（用于返回前端）
func MaskConfig(configJSON []byte) ([]byte, error) {
	var config map[string]interface{}
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return configJSON, nil // 解析失败返回原文，避免阻断展示
	}
	masked := maskMap(config)
	return json.Marshal(masked)
}

// maskMap 递归脱敏 map 中的敏感字段
func maskMap(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(m))
	for k, v := range m {
		if maskFields[k] {
			result[k] = "***"
			continue
		}
		if childMap, ok := v.(map[string]interface{}); ok {
			result[k] = maskMap(childMap)
		} else {
			result[k] = v
		}
	}
	return result
}
