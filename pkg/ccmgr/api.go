package ccmgr

import (
	"context"

	"github.com/unbracketed/ccmgr-ultra/internal/config"
	"github.com/unbracketed/ccmgr-ultra/internal/tui"
)

// Client provides the main API for ccmgr-ultra library
type Client struct {
	integration *tui.Integration
	config      *config.Config
	ctx         context.Context
}

// NewClient creates a new ccmgr client with the given configuration
func NewClient(cfg *config.Config) (*Client, error) {
	if cfg == nil {
		var err error
		cfg, err = config.Load()
		if err != nil {
			return nil, err
		}
	}

	integration, err := tui.NewIntegration(cfg)
	if err != nil {
		return nil, err
	}

	return &Client{
		integration: integration,
		config:      cfg,
		ctx:         context.Background(),
	}, nil
}

// NewClientWithContext creates a new ccmgr client with custom context
func NewClientWithContext(ctx context.Context, cfg *config.Config) (*Client, error) {
	client, err := NewClient(cfg)
	if err != nil {
		return nil, err
	}
	client.ctx = ctx
	return client, nil
}

// Sessions returns the session manager interface
func (c *Client) Sessions() SessionManager {
	return &sessionManager{c.integration}
}

// Worktrees returns the worktree manager interface
func (c *Client) Worktrees() WorktreeManager {
	return &worktreeManager{c.integration}
}

// System returns the system status interface
func (c *Client) System() SystemManager {
	return &systemManager{c.integration}
}

// Close gracefully shuts down the client
func (c *Client) Close() error {
	c.integration.Shutdown()
	return nil
}
