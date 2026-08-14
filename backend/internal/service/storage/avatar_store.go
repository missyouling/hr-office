package storage

import (
	"fmt"
	"os"
	"time"
)

// AvatarStore 用户头像专用本地存储。
// 与 StorageManager/StorageConfig 完全解耦：不依赖数据库中的存储配置，
// 直接操作专用根目录，避免通用默认本地存储配置损坏时头像功能不可用。
type AvatarStore struct {
	rootDir string
}

// NewAvatarStore 创建头像存储并确保根目录存在。
func NewAvatarStore(rootDir string) (*AvatarStore, error) {
	if rootDir == "" {
		return nil, fmt.Errorf("avatar root dir is empty")
	}
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		return nil, fmt.Errorf("create avatar root dir: %w", err)
	}
	return &AvatarStore{rootDir: rootDir}, nil
}

// Save 保存头像数据，返回相对根目录的路径。
// 文件名由服务端生成（userID_纳秒时间戳.扩展名），用户无法控制文件名，
// 天然规避路径穿越与文件名注入。
func (s *AvatarStore) Save(userID uint, ext string, data []byte) (string, error) {
	name := fmt.Sprintf("%d_%d%s", userID, time.Now().UnixNano(), ext)
	fullPath, err := safePathUnderBase(s.rootDir, name)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(fullPath, data, 0o644); err != nil {
		return "", fmt.Errorf("write avatar file: %w", err)
	}
	return name, nil
}

// Read 读取头像文件内容（带路径穿越防护）。
func (s *AvatarStore) Read(relPath string) ([]byte, error) {
	fullPath, err := safePathUnderBase(s.rootDir, relPath)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(fullPath)
}

// Delete 删除头像文件（带路径穿越防护）；文件不存在视为成功。
func (s *AvatarStore) Delete(relPath string) error {
	fullPath, err := safePathUnderBase(s.rootDir, relPath)
	if err != nil {
		return err
	}
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete avatar file: %w", err)
	}
	return nil
}
