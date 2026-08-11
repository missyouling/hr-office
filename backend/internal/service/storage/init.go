package storage

import "log"

func init() {
	// 正式驱动（完整实现文件操作）
	DefaultRegistry.Register("local", func() Driver { return &LocalDriver{} })
	DefaultRegistry.Register("s3", func() Driver { return &S3Driver{} })
	DefaultRegistry.Register("webdav", func() Driver { return &WebDAVDriver{} })
	DefaultRegistry.Register("nas", func() Driver { return &WebDAVDriver{} })

	// 实验性驱动（OAuth 驱动当前仅支持连通性测试，文件操作未实现）
	// 注册时添加 "-experimental" 后缀以区分状态
	expDrivers := map[string]DriverFactory{
		"google_drive": func() Driver { return NewOAuthDriver("google_drive") },
		"onedrive":     func() Driver { return NewOAuthDriver("onedrive") },
		"aliyun_drive": func() Driver { return NewOAuthDriver("aliyun_drive") },
		"cmcc_cloud":   func() Driver { return NewOAuthDriver("cmcc_cloud") },
		"115_drive":    func() Driver { return NewOAuthDriver("115_drive") },
	}
	for name, factory := range expDrivers {
		experimentalName := name + "-experimental"
		DefaultRegistry.Register(experimentalName, factory)
		log.Printf("[StorageRegistry] registered experimental driver: %s (OAuth file operations not yet implemented)", name)
	}
}
