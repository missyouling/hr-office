package supabase

import (
	"fmt"
	"os"

	supa "github.com/supabase-community/supabase-go"
)

// Client wraps Supabase client
type Client struct {
	*supa.Client
}

// NewClient creates a new Supabase client
func NewClient() (*Client, error) {
	supabaseURL := os.Getenv("SUPABASE_URL")
	if supabaseURL == "" {
		return nil, fmt.Errorf("SUPABASE_URL environment variable is required")
	}

	supabaseKey := os.Getenv("SUPABASE_SERVICE_ROLE_KEY")
	if supabaseKey == "" {
		return nil, fmt.Errorf("SUPABASE_SERVICE_ROLE_KEY environment variable is required")
	}

	client, err := supa.NewClient(supabaseURL, supabaseKey, &supa.ClientOptions{})
	if err != nil {
		return nil, fmt.Errorf("create supabase client: %w", err)
	}

	return &Client{Client: client}, nil
}
