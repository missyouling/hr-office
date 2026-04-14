package storage

func init() {
	DefaultRegistry.Register("local", func() Driver { return &LocalDriver{} })
	DefaultRegistry.Register("s3", func() Driver { return &S3Driver{} })
	DefaultRegistry.Register("webdav", func() Driver { return &WebDAVDriver{} })
	DefaultRegistry.Register("nas", func() Driver { return &WebDAVDriver{} })
	DefaultRegistry.Register("google_drive", func() Driver { return NewOAuthDriver("google_drive") })
	DefaultRegistry.Register("onedrive", func() Driver { return NewOAuthDriver("onedrive") })
	DefaultRegistry.Register("aliyun_drive", func() Driver { return NewOAuthDriver("aliyun_drive") })
	DefaultRegistry.Register("cmcc_cloud", func() Driver { return NewOAuthDriver("cmcc_cloud") })
	DefaultRegistry.Register("115_drive", func() Driver { return NewOAuthDriver("115_drive") })
}
