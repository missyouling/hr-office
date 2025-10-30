package supabase

import (
	"fmt"
	"os"

	supa "github.com/supabase-community/supabase-go"
)

var Client *supa.Client

// InitClient 初始化Supabase客户端
func InitClient() error {
	supabaseURL := os.Getenv("SUPABASE_URL")
	supabaseKey := os.Getenv("SUPABASE_SERVICE_KEY")

	if supabaseURL == "" {
		return fmt.Errorf("SUPABASE_URL environment variable is required")
	}

	if supabaseKey == "" {
		return fmt.Errorf("SUPABASE_SERVICE_KEY environment variable is required")
	}

	client, err := supa.NewClient(supabaseURL, supabaseKey, nil)
	if err != nil {
		return fmt.Errorf("failed to create Supabase client: %w", err)
	}

	Client = client
	return nil
}

// GetClient 获取Supabase客户端实例
func GetClient() *supa.Client {
	return Client
}
